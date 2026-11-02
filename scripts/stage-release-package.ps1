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

function Write-AsciiLinesLF([string]$Path, [string[]]$Lines) {
  $text = if ($Lines.Count -gt 0) { ($Lines -join "`n") + "`n" } else { "" }
  [System.IO.File]::WriteAllText($Path, $text, [System.Text.Encoding]::ASCII)
}

$destinationPrefix = $Destination.TrimEnd([System.IO.Path]::DirectorySeparatorChar) + [System.IO.Path]::DirectorySeparatorChar
if ($Destination -eq [System.IO.Path]::GetPathRoot($Destination) -or
    $Destination -eq $Root -or
    $Root.StartsWith($destinationPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
  throw "refusing to stage a release at filesystem root: $Destination"
}
New-Item -ItemType Directory -Force -Path $Destination | Out-Null

$sourceManifestLines = @()
$sourcePaths = @(& git -C $Root ls-files --cached --others --exclude-standard -- . ':!agent.md' ':!AGENTS.md') | Sort-Object -Unique
if ($LASTEXITCODE -ne 0) {
  throw "git ls-files failed with exit code $LASTEXITCODE"
}
$sourceChanges = @(& git -C $Root status --porcelain=v1 --untracked-files=normal -- . ':!agent.md' ':!AGENTS.md')
if ($LASTEXITCODE -ne 0) {
  throw "git status failed with exit code $LASTEXITCODE"
}
foreach ($relative in $sourcePaths) {
  $gitPath = $relative.Replace('\', '/')
  $sourcePath = Join-Path $Root $relative
  # Build inputs are the working tree, not HEAD. Include new source files and
  # omit deleted inputs; ignore rules keep local data and credentials excluded.
  if (-not (Test-Path -LiteralPath $sourcePath -PathType Leaf)) {
    continue
  }
  $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $sourcePath).Hash.ToLowerInvariant()
  $sourceManifestLines += "$hash  $gitPath"
}

$sourceManifestText = ($sourceManifestLines -join "`n") + "`n"
$sha256 = [System.Security.Cryptography.SHA256]::Create()
try {
  $sourceDigestBytes = $sha256.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($sourceManifestText))
  $sourceDigest = [BitConverter]::ToString($sourceDigestBytes).Replace('-', '').ToLowerInvariant()
} finally {
  $sha256.Dispose()
}

# docs/ is no longer tracked in the repository; package it only when present
# locally so CI checkouts without docs/ can still stage a complete package.
$readmeDistribution = Join-Path $Root "docs\README-distribution.md"
if (Test-Path -LiteralPath $readmeDistribution) {
  Copy-Item -LiteralPath $readmeDistribution -Destination (Join-Path $Destination "README.md")
}
Copy-Item -LiteralPath (Join-Path $Root "config.example.yaml") -Destination (Join-Path $Destination "config.example.yaml")
Copy-Item -LiteralPath (Join-Path $Root "CHANGELOG.md") -Destination (Join-Path $Destination "CHANGELOG.md")
Copy-Item -LiteralPath (Join-Path $Root "SECURITY.md") -Destination (Join-Path $Destination "SECURITY.md")
$docsDestination = Join-Path $Destination "docs"
$docSources = @("linux.md", "operations.md", "protocol-compatibility.md", "upgrade.md") |
  ForEach-Object { Join-Path $Root "docs\$_" } | Where-Object { Test-Path -LiteralPath $_ }
if ($docSources.Count -gt 0) {
  New-Item -ItemType Directory -Force -Path $docsDestination | Out-Null
  foreach ($docSource in $docSources) {
    Copy-Item -LiteralPath $docSource -Destination (Join-Path $docsDestination (Split-Path -Leaf $docSource))
  }
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
  sourceMode = "working-tree"
  sourceDirty = ($sourceChanges.Count -gt 0)
}
$metadata | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $Destination "VERSION.json") -Encoding utf8
Write-AsciiLinesLF (Join-Path $Destination "SOURCE-MANIFEST.txt") $sourceManifestLines

$checksumPath = Join-Path $Destination "SHA256SUMS"
$checksumLines = Get-ChildItem -LiteralPath $Destination -File -Recurse |
  Where-Object { $_.FullName -ne $checksumPath } |
  Sort-Object FullName |
  ForEach-Object {
    $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $_.FullName).Hash.ToLowerInvariant()
    $fullPath = $_.FullName
    $relative = $fullPath.Substring($Destination.Length).TrimStart([System.IO.Path]::DirectorySeparatorChar).Replace('\', '/')
    "$hash  $relative"
  }
Write-AsciiLinesLF $checksumPath $checksumLines
