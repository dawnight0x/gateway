$ErrorActionPreference = "Stop"
$root = [System.IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot))
$testParent = Join-Path $root "bin"
$testRoot = Join-Path $testParent ("provenance-test-" + [Guid]::NewGuid().ToString("N"))
$fixture = Join-Path $testRoot "fixture"
$stage = Join-Path $testRoot "stage"
$previousPath = $env:PATH

function Write-Fixture([string]$Relative, [string]$Text) {
  $path = Join-Path $fixture $Relative
  New-Item -ItemType Directory -Force -Path (Split-Path -Parent $path) | Out-Null
  [System.IO.File]::WriteAllText($path, $Text, [System.Text.UTF8Encoding]::new($false))
}

function Invoke-FixtureGit([string[]]$Arguments) {
  & git -C $fixture @Arguments
  if ($LASTEXITCODE -ne 0) { throw "fixture git command failed: $Arguments" }
}

try {
  New-Item -ItemType Directory -Path $fixture -Force | Out-Null
  $localGo = Join-Path $root ".tools/go/bin"
  if (Test-Path -LiteralPath $localGo) {
    $env:PATH = $localGo + [System.IO.Path]::PathSeparator + $previousPath
  }
  foreach ($relative in @(
    "scripts/stage-release-package.ps1", "scripts/generate-sbom.ps1",
    "docs/README-distribution.md", "docs/linux.md", "docs/operations.md", "docs/protocol-compatibility.md", "docs/upgrade.md",
    "config.example.yaml", "CHANGELOG.md", "SECURITY.md", "package-lock.json"
  )) {
    $destination = Join-Path $fixture $relative
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $destination) | Out-Null
    Copy-Item -LiteralPath (Join-Path $root $relative) -Destination $destination
  }
  Write-Fixture "go.mod" "module example.test/release-probe`ngo 1.25.0`n"
  Write-Fixture ".gitignore" "data/`n"
  Write-Fixture "fixture.go" "package fixture`nconst Value = 1`n"
  Write-Fixture "deleted.go" "package fixture`n"
  Invoke-FixtureGit @("init", "--quiet")
  Invoke-FixtureGit @("-c", "core.autocrlf=false", "add", ".")
  Invoke-FixtureGit @("-c", "user.name=Release Test", "-c", "user.email=release-test@example.invalid", "commit", "--quiet", "-m", "fixture")
  Write-Fixture "fixture.go" "package fixture`nconst Value = 2`n"
  Write-Fixture "new.go" "package fixture`nconst Added = true`n"
  Write-Fixture "agent.md" "local instructions not part of the build`n"
  Write-Fixture "data/secret.key" "test-only secret`n"
  Remove-Item -LiteralPath (Join-Path $fixture "deleted.go")
  Push-Location $fixture
  try {
    & (Join-Path $fixture "scripts/stage-release-package.ps1") -Destination $stage -Version "1.0.0" -Commit "fixture-dirty" -BuiltAt "2026-10-12T00:00:00Z" -Platform "test"
  } finally { Pop-Location }
  $manifest = Get-Content -LiteralPath (Join-Path $stage "SOURCE-MANIFEST.txt") -Raw
  foreach ($relative in @("fixture.go", "new.go")) {
    $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $fixture $relative)).Hash.ToLowerInvariant()
    if (-not $manifest.Contains("$hash  $relative`n")) { throw "manifest does not describe actual input: $relative" }
  }
  if ($manifest -match '  (agent\.md|data/|deleted\.go)') { throw "manifest includes local-only or deleted input" }
  $metadata = Get-Content -LiteralPath (Join-Path $stage "VERSION.json") -Raw | ConvertFrom-Json
  $digest = (Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $stage "SOURCE-MANIFEST.txt")).Hash.ToLowerInvariant()
  if ($metadata.sourceMode -ne "working-tree" -or -not $metadata.sourceDirty -or $metadata.sourceSha256 -ne $digest) {
    throw "incorrect provenance metadata"
  }
  $sbom = Get-Content -LiteralPath (Join-Path $stage "sbom.cdx.json") -Raw | ConvertFrom-Json
  if (-not ($sbom.components | Where-Object name -eq "vue")) { throw "embedded Vue runtime is absent from SBOM" }
  Write-Host "Release provenance regression passed."
} finally {
  $env:PATH = $previousPath
  $resolved = [System.IO.Path]::GetFullPath($testRoot)
  $prefix = [System.IO.Path]::GetFullPath($testParent) + [System.IO.Path]::DirectorySeparatorChar
  if (-not $resolved.StartsWith($prefix, [System.StringComparison]::OrdinalIgnoreCase) -or
      [System.IO.Path]::GetFileName($resolved) -notmatch '^provenance-test-[0-9a-f]{32}$') {
    throw "refusing to remove unsafe test fixture path: $resolved"
  }
  if (Test-Path -LiteralPath $resolved) { Remove-Item -LiteralPath $resolved -Recurse -Force }
}
