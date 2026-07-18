param(
  [Nullable[int]]$Major = $null,
  [Nullable[int]]$Minor = $null,
  [Nullable[int]]$Patch = $null,
  [int]$Build = 0,
  [string]$ProductVersion = ""
)

$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
$ReleaseMetadataPath = Join-Path $Root "release.json"
if ($null -eq $Major -or $null -eq $Minor -or $null -eq $Patch) {
  try {
    $releaseMetadata = Get-Content -Raw -LiteralPath $ReleaseMetadataPath | ConvertFrom-Json -ErrorAction Stop
  } catch {
    throw "read release metadata $ReleaseMetadataPath`: $($_.Exception.Message)"
  }
  $releaseMatch = [regex]::Match([string]$releaseMetadata.version, '^(?<major>0|[1-9][0-9]*)\.(?<minor>0|[1-9][0-9]*)\.(?<patch>0|[1-9][0-9]*)$')
  if (-not $releaseMatch.Success) {
    throw "release metadata version must use major.minor.patch format"
  }
  if ($null -eq $Major) { $Major = [int]$releaseMatch.Groups['major'].Value }
  if ($null -eq $Minor) { $Minor = [int]$releaseMatch.Groups['minor'].Value }
  if ($null -eq $Patch) { $Patch = [int]$releaseMatch.Groups['patch'].Value }
}
$Go = Join-Path $Root ".tools\go\bin\go.exe"
if (-not (Test-Path -LiteralPath $Go)) {
  $Go = (Get-Command go -ErrorAction Stop).Source
}
$GatewayDir = Join-Path $Root "cmd\gateway"
$BackupDir = Join-Path $Root "cmd\gateway-backup"
$Icon = Join-Path $GatewayDir "app.ico"
$ManifestTemplate = Join-Path $GatewayDir "app.manifest"
$GatewayVersionInfoTemplate = Join-Path $GatewayDir "versioninfo.json"
$BackupVersionInfoTemplate = Join-Path $BackupDir "versioninfo.json"
$GatewayResource = Join-Path $GatewayDir "rsrc_windows_amd64.syso"
$BackupResource = Join-Path $BackupDir "rsrc_windows_amd64.syso"
$GatewayManifest = Join-Path ([System.IO.Path]::GetTempPath()) ("gateway-manifest-" + [Guid]::NewGuid().ToString("N") + ".xml")
$BackupManifest = Join-Path ([System.IO.Path]::GetTempPath()) ("gateway-backup-manifest-" + [Guid]::NewGuid().ToString("N") + ".xml")
$GatewayVersionInfo = Join-Path ([System.IO.Path]::GetTempPath()) ("gateway-version-" + [Guid]::NewGuid().ToString("N") + ".json")
$BackupVersionInfo = Join-Path ([System.IO.Path]::GetTempPath()) ("gateway-backup-version-" + [Guid]::NewGuid().ToString("N") + ".json")

foreach ($part in @($Major, $Minor, $Patch, $Build)) {
  if ($part -lt 0 -or $part -gt [uint16]::MaxValue) {
    throw "Windows version components must be between 0 and $([uint16]::MaxValue)"
  }
}
if ([string]::IsNullOrWhiteSpace($ProductVersion)) {
  $ProductVersion = "$Major.$Minor.$Patch"
}
$ManifestVersion = "$Major.$Minor.$Patch.$Build"

function Write-VersionedManifest([string]$Destination, [string]$IdentityName) {
  [xml]$document = [System.IO.File]::ReadAllText($ManifestTemplate)
  $identity = $document.SelectSingleNode("/*[local-name()='assembly']/*[local-name()='assemblyIdentity']")
  if (-not $identity) {
    throw "assemblyIdentity was not found in $ManifestTemplate"
  }
  $identity.SetAttribute("name", $IdentityName)
  $identity.SetAttribute("version", $ManifestVersion)

  $settings = [System.Xml.XmlWriterSettings]::new()
  $settings.Encoding = [System.Text.UTF8Encoding]::new($false)
  $settings.Indent = $true
  $writer = [System.Xml.XmlWriter]::Create($Destination, $settings)
  try {
    $document.Save($writer)
  } finally {
    $writer.Dispose()
  }
}

function Write-VersionInfo([string]$Template, [string]$Destination) {
  $info = Get-Content -Raw -LiteralPath $Template | ConvertFrom-Json -ErrorAction Stop
  foreach ($version in @($info.FixedFileInfo.FileVersion, $info.FixedFileInfo.ProductVersion)) {
    $version.Major = $Major
    $version.Minor = $Minor
    $version.Patch = $Patch
    $version.Build = $Build
  }
  $info.StringFileInfo.FileVersion = $ManifestVersion
  $info.StringFileInfo.ProductVersion = $ProductVersion
  $info | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $Destination -Encoding utf8
}

function Invoke-VersionResource([string]$Output, [string]$VersionInfo, [string]$Manifest) {
  & $Go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.5.0 `
    -64 `
    -o $Output `
    -icon $Icon `
    -manifest $Manifest `
    -ver-major $Major `
    -ver-minor $Minor `
    -ver-patch $Patch `
    -ver-build $Build `
    -product-version $ProductVersion `
    $VersionInfo
  if ($LASTEXITCODE -ne 0) {
    throw "Windows resource generation failed for $VersionInfo with exit code $LASTEXITCODE"
  }
}

Push-Location -LiteralPath $Root
try {
  & $Go run -buildvcs=false (Join-Path $Root "scripts\generate-windows-icon.go") $Icon
  if ($LASTEXITCODE -ne 0) {
    throw "icon generation failed with exit code $LASTEXITCODE"
  }

  Write-VersionedManifest $GatewayManifest "LocalAIGateway"
  Write-VersionedManifest $BackupManifest "LocalAIGateway.Backup"
  Write-VersionInfo $GatewayVersionInfoTemplate $GatewayVersionInfo
  Write-VersionInfo $BackupVersionInfoTemplate $BackupVersionInfo
  Invoke-VersionResource $GatewayResource $GatewayVersionInfo $GatewayManifest
  Invoke-VersionResource $BackupResource $BackupVersionInfo $BackupManifest
} finally {
  Pop-Location
  foreach ($temporary in @($GatewayManifest, $BackupManifest, $GatewayVersionInfo, $BackupVersionInfo)) {
    if (Test-Path -LiteralPath $temporary) {
      Remove-Item -LiteralPath $temporary -Force
    }
  }
}

Write-Host "Generated $Icon"
Write-Host "Generated $GatewayResource (FileVersion=$ManifestVersion ProductVersion=$ProductVersion)"
Write-Host "Generated $BackupResource (FileVersion=$ManifestVersion ProductVersion=$ProductVersion)"
