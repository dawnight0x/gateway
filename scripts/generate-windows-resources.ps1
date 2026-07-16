param(
  [int]$Major = 0,
  [int]$Minor = 6,
  [int]$Patch = 0,
  [int]$Build = 0,
  [string]$ProductVersion = ""
)

$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
$Go = Join-Path $Root ".tools\go\bin\go.exe"
if (-not (Test-Path -LiteralPath $Go)) {
  $Go = (Get-Command go -ErrorAction Stop).Source
}
$GatewayDir = Join-Path $Root "cmd\gateway"
$BackupDir = Join-Path $Root "cmd\gateway-backup"
$Icon = Join-Path $GatewayDir "app.ico"
$ManifestTemplate = Join-Path $GatewayDir "app.manifest"
$GatewayVersionInfo = Join-Path $GatewayDir "versioninfo.json"
$BackupVersionInfo = Join-Path $BackupDir "versioninfo.json"
$GatewayResource = Join-Path $GatewayDir "rsrc_windows_amd64.syso"
$BackupResource = Join-Path $BackupDir "rsrc_windows_amd64.syso"
$GatewayManifest = Join-Path ([System.IO.Path]::GetTempPath()) ("gateway-manifest-" + [Guid]::NewGuid().ToString("N") + ".xml")
$BackupManifest = Join-Path ([System.IO.Path]::GetTempPath()) ("gateway-backup-manifest-" + [Guid]::NewGuid().ToString("N") + ".xml")

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
  Invoke-VersionResource $GatewayResource $GatewayVersionInfo $GatewayManifest
  Invoke-VersionResource $BackupResource $BackupVersionInfo $BackupManifest
} finally {
  Pop-Location
  foreach ($manifest in @($GatewayManifest, $BackupManifest)) {
    if (Test-Path -LiteralPath $manifest) {
      Remove-Item -LiteralPath $manifest -Force
    }
  }
}

Write-Host "Generated $Icon"
Write-Host "Generated $GatewayResource (FileVersion=$ManifestVersion ProductVersion=$ProductVersion)"
Write-Host "Generated $BackupResource (FileVersion=$ManifestVersion ProductVersion=$ProductVersion)"
