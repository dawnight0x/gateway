$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
$Go = Join-Path $Root ".tools\go\bin\go.exe"
$Icon = Join-Path $Root "cmd\gateway\app.ico"
$Resource = Join-Path $Root "cmd\gateway\rsrc_windows_amd64.syso"

Set-Location -LiteralPath $Root

& $Go run -buildvcs=false .\scripts\generate-windows-icon.go $Icon
if ($LASTEXITCODE -ne 0) {
  throw "icon generation failed with exit code $LASTEXITCODE"
}

& $Go run github.com/akavel/rsrc@v0.10.2 -arch amd64 -ico $Icon -o $Resource
if ($LASTEXITCODE -ne 0) {
  throw "Windows resource generation failed with exit code $LASTEXITCODE"
}

Write-Host "Generated $Icon"
Write-Host "Generated $Resource"
