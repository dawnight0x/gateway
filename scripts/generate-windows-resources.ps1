param(
  [int]$Major = 2,
  [int]$Minor = 0,
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
$Icon = Join-Path $GatewayDir "app.ico"
$Manifest = Join-Path $GatewayDir "app.manifest"
$VersionInfo = Join-Path $GatewayDir "versioninfo.json"
$Resource = Join-Path $GatewayDir "rsrc_windows_amd64.syso"

if ([string]::IsNullOrWhiteSpace($ProductVersion)) {
  $ProductVersion = "$Major.$Minor.$Patch.$Build"
}

Push-Location -LiteralPath $Root
try {
  & $Go run -buildvcs=false .\scripts\generate-windows-icon.go $Icon
  if ($LASTEXITCODE -ne 0) {
    throw "icon generation failed with exit code $LASTEXITCODE"
  }

  # goversioninfo embeds the icon, VERSIONINFO block, and application manifest
  # into a single .syso. rsrc only embeds the icon, leaving the exe without
  # file/product version metadata or DPI/OS-compatibility manifest.
  & $Go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.5.0 `
    -64 `
    -o $Resource `
    -icon $Icon `
    -manifest $Manifest `
    -ver-major $Major `
    -ver-minor $Minor `
    -ver-patch $Patch `
    -ver-build $Build `
    -product-version $ProductVersion `
    $VersionInfo
  if ($LASTEXITCODE -ne 0) {
    throw "Windows resource generation failed with exit code $LASTEXITCODE"
  }
} finally {
  Pop-Location
}

Write-Host "Generated $Icon"
Write-Host "Generated $Resource (FileVersion=$Major.$Minor.$Patch.$Build ProductVersion=$ProductVersion)"
