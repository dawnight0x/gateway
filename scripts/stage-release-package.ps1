param(
  [Parameter(Mandatory = $true)]
  [string]$Destination,
  [Parameter(Mandatory = $true)]
  [string]$Version,
  [Parameter(Mandatory = $true)]
  [string]$Commit,
  [Parameter(Mandatory = $true)]
  [string]$BuiltAt,
  [Parameter(Mandatory = $true)]
  [string]$Platform,
  [string]$PackageName = "local-ai-gateway"
)

$ErrorActionPreference = "Stop"
$Root = [System.IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot))
$Destination = [System.IO.Path]::GetFullPath($Destination)
$destinationPrefix = $Destination.TrimEnd([System.IO.Path]::DirectorySeparatorChar) + [System.IO.Path]::DirectorySeparatorChar
if ($Destination -eq [System.IO.Path]::GetPathRoot($Destination) -or
    $Destination -eq $Root -or
    $Root.StartsWith($destinationPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
  throw "refusing to stage a release at filesystem root: $Destination"
}
New-Item -ItemType Directory -Force -Path $Destination | Out-Null

$sourceManifestLines = @()
$sourcePaths = @(& git -C $Root ls-files --cached --others --exclude-standard) | Sort-Object -Unique
if ($LASTEXITCODE -ne 0) {
  throw "git ls-files failed with exit code $LASTEXITCODE"
}
foreach ($relative in $sourcePaths) {
  $sourcePath = [System.IO.Path]::GetFullPath((Join-Path $Root $relative))
  if ($sourcePath.StartsWith($destinationPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
    continue
  }
  if (Test-Path -LiteralPath $sourcePath -PathType Leaf) {
    $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $sourcePath).Hash.ToLowerInvariant()
    $sourceManifestLines += "$hash  $($relative.Replace('\', '/'))"
  }
}

$sourceManifestText = ($sourceManifestLines -join "`n") + "`n"
$sourceDigestBytes = [System.Security.Cryptography.SHA256]::HashData([System.Text.Encoding]::UTF8.GetBytes($sourceManifestText))
$sourceDigest = [Convert]::ToHexString($sourceDigestBytes).ToLowerInvariant()

Copy-Item -LiteralPath (Join-Path $Root "docs\README-distribution.md") -Destination (Join-Path $Destination "README.md")
Copy-Item -LiteralPath (Join-Path $Root "config.example.yaml") -Destination (Join-Path $Destination "config.example.yaml")
$docsDestination = Join-Path $Destination "docs"
New-Item -ItemType Directory -Force -Path $docsDestination | Out-Null
foreach ($doc in @("linux.md", "operations.md", "protocol-compatibility.md")) {
  Copy-Item -LiteralPath (Join-Path $Root "docs\$doc") -Destination (Join-Path $docsDestination $doc)
}

& (Join-Path $Root "scripts\generate-sbom.ps1") `
  -OutputPath (Join-Path $Destination "sbom.cdx.json") `
  -Version $Version `
  -Commit $Commit `
  -Platform $Platform `
  -PackageName $PackageName
if ($LASTEXITCODE -ne 0) {
  throw "SBOM generation failed with exit code $LASTEXITCODE"
}

$metadata = [ordered]@{
  name = $PackageName
  version = $Version
  commit = $Commit
  builtAt = $BuiltAt
  platform = $Platform
  defaultPort = 18787
  sourceSha256 = $sourceDigest
}
$metadata | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $Destination "VERSION.json") -Encoding utf8
$sourceManifestLines | Set-Content -LiteralPath (Join-Path $Destination "SOURCE-MANIFEST.txt") -Encoding ascii

$checksumPath = Join-Path $Destination "SHA256SUMS"
$checksumLines = Get-ChildItem -LiteralPath $Destination -File -Recurse |
  Where-Object { $_.FullName -ne $checksumPath } |
  Sort-Object FullName |
  ForEach-Object {
    $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $_.FullName).Hash.ToLowerInvariant()
    $relative = [System.IO.Path]::GetRelativePath($Destination, $_.FullName).Replace('\', '/')
    "$hash  $relative"
  }
$checksumLines | Set-Content -LiteralPath $checksumPath -Encoding ascii
