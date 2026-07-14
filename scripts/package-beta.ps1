param(
  [string]$OutputRoot = "E:\gateway-beta",
  [string]$Version = "v2-beta4"
)

$ErrorActionPreference = "Stop"
$Root = [System.IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot))
$OutputRoot = [System.IO.Path]::GetFullPath($OutputRoot).TrimEnd('\')
$outputLeaf = [System.IO.Path]::GetFileName($OutputRoot)
if ([string]::IsNullOrWhiteSpace($outputLeaf) -or $OutputRoot -eq [System.IO.Path]::GetPathRoot($OutputRoot)) {
  throw "refusing to package into unsafe output root: $OutputRoot"
}

$Go = Join-Path $Root ".tools\go\bin\go.exe"
if (-not (Test-Path -LiteralPath $Go)) {
  $Go = (Get-Command go -ErrorAction Stop).Source
}
$Tar = (Get-Command tar -ErrorAction Stop).Source
$BuiltAt = [DateTime]::UtcNow.ToString("o")
$Commit = "unknown"
try {
  $candidateCommit = (& git -C $Root rev-parse --verify HEAD 2>$null).Trim()
  if ($LASTEXITCODE -eq 0 -and $candidateCommit) {
    $Commit = $candidateCommit
	$worktreeChanges = @(& git -C $Root status --porcelain=v1 --untracked-files=normal)
	if ($worktreeChanges.Count -gt 0) {
		$Commit += "-dirty"
	}
  }
} catch {}
$SourceManifestLines = @()
try {
	$sourcePaths = @(& git -C $Root ls-files --cached --others --exclude-standard) | Sort-Object
	foreach ($relative in $sourcePaths) {
		$sourcePath = Join-Path $Root $relative
		if (Test-Path -LiteralPath $sourcePath -PathType Leaf) {
			$hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $sourcePath).Hash.ToLowerInvariant()
			$SourceManifestLines += "$hash  $($relative.Replace('\', '/'))"
		}
	}
} catch {}
$sourceManifestText = ($SourceManifestLines -join "`n") + "`n"
$sourceDigestBytes = [System.Security.Cryptography.SHA256]::HashData([System.Text.Encoding]::UTF8.GetBytes($sourceManifestText))
$SourceDigest = [Convert]::ToHexString($sourceDigestBytes).ToLowerInvariant()
$LdVersion = $Version.Replace("'", "")
$LdCommit = $Commit.Replace("'", "")
$LdBuiltAt = $BuiltAt.Replace("'", "")
$CommonLdFlags = "-s -w -X local-ai-gateway/internal/buildinfo.Version=$LdVersion -X local-ai-gateway/internal/buildinfo.Commit=$LdCommit -X local-ai-gateway/internal/buildinfo.BuiltAt=$LdBuiltAt"

New-Item -ItemType Directory -Force -Path $OutputRoot | Out-Null
$StagingRoot = Join-Path $OutputRoot (".staging-" + [Guid]::NewGuid().ToString("N"))
$WinStage = Join-Path $StagingRoot "Win-amd64"
$LinuxStage = Join-Path $StagingRoot "linux-amd64"
New-Item -ItemType Directory -Force -Path $WinStage,$LinuxStage | Out-Null

function Assert-SafePackagePath([string]$Path, [string]$ExpectedLeaf) {
  $full = [System.IO.Path]::GetFullPath($Path)
  $prefix = $OutputRoot + [System.IO.Path]::DirectorySeparatorChar
  if (-not $full.StartsWith($prefix, [System.StringComparison]::OrdinalIgnoreCase) -or [System.IO.Path]::GetFileName($full) -ne $ExpectedLeaf) {
    throw "refusing to modify unexpected package path: $full"
  }
}

function Invoke-GoBuild([string]$GOOS, [string]$GOARCH, [string]$Package, [string]$Output, [string]$LdFlags) {
  $previousGOOS = $env:GOOS
  $previousGOARCH = $env:GOARCH
  $previousCGO = $env:CGO_ENABLED
  try {
    $env:GOOS = $GOOS
    $env:GOARCH = $GOARCH
    $env:CGO_ENABLED = "0"
    & $Go build -trimpath -buildvcs=false -ldflags $LdFlags -o $Output $Package
    if ($LASTEXITCODE -ne 0) {
      throw "Go build failed for $GOOS/$GOARCH $Package"
    }
  } finally {
    $env:GOOS = $previousGOOS
    $env:GOARCH = $previousGOARCH
    $env:CGO_ENABLED = $previousCGO
  }
}

function Copy-PackageReadme([string]$Destination) {
  Copy-Item -LiteralPath (Join-Path $Root "docs\README-distribution.md") -Destination (Join-Path $Destination "README.md")
	Copy-Item -LiteralPath (Join-Path $Root "config.example.yaml") -Destination (Join-Path $Destination "config.example.yaml")
	$DocsDestination = Join-Path $Destination "docs"
	New-Item -ItemType Directory -Force -Path $DocsDestination | Out-Null
	foreach ($doc in @("linux.md", "operations.md", "protocol-compatibility.md")) {
		Copy-Item -LiteralPath (Join-Path $Root "docs\$doc") -Destination (Join-Path $DocsDestination $doc)
	}
}

function Write-PackageMetadata([string]$Destination, [string]$Platform) {
	$metadata = [ordered]@{
		name = "local-ai-gateway"
		version = $Version
		commit = $Commit
		builtAt = $BuiltAt
		platform = $Platform
		defaultPort = 18787
		sourceSha256 = $SourceDigest
	}
	$metadata | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $Destination "VERSION.json") -Encoding utf8
	$SourceManifestLines | Set-Content -LiteralPath (Join-Path $Destination "SOURCE-MANIFEST.txt") -Encoding ascii
	$lines = Get-ChildItem -LiteralPath $Destination -File -Recurse | Where-Object { $_.Name -ne "SHA256SUMS" } | Sort-Object FullName | ForEach-Object {
		$hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $_.FullName).Hash.ToLowerInvariant()
		$relative = [System.IO.Path]::GetRelativePath($Destination, $_.FullName).Replace('\', '/')
		"$hash  $relative"
	}
	$lines | Set-Content -LiteralPath (Join-Path $Destination "SHA256SUMS") -Encoding ascii
}

try {
  Push-Location $Root
  try {
    npm run build:admin
    if ($LASTEXITCODE -ne 0) {
      throw "admin build failed"
    }
    & $Go test ./...
    if ($LASTEXITCODE -ne 0) {
      throw "Go tests failed"
    }

		Invoke-GoBuild "windows" "amd64" ".\cmd\gateway" (Join-Path $WinStage "gateway.exe") "$CommonLdFlags -H=windowsgui"
		Invoke-GoBuild "windows" "amd64" ".\cmd\gateway-backup" (Join-Path $WinStage "gateway-backup.exe") $CommonLdFlags
		Invoke-GoBuild "linux" "amd64" ".\cmd\gateway" (Join-Path $LinuxStage "gateway") $CommonLdFlags
		Invoke-GoBuild "linux" "amd64" ".\cmd\gateway-backup" (Join-Path $LinuxStage "gateway-backup") $CommonLdFlags

		Copy-PackageReadme $WinStage
		Copy-PackageReadme $LinuxStage
		$previousRefName = $env:GITHUB_REF_NAME
		try {
			$env:GITHUB_REF_NAME = $Version
			& (Join-Path $Root "scripts\generate-sbom.ps1") -OutputPath (Join-Path $WinStage "sbom.cdx.json")
			Copy-Item -LiteralPath (Join-Path $WinStage "sbom.cdx.json") -Destination (Join-Path $LinuxStage "sbom.cdx.json")
		} finally {
			$env:GITHUB_REF_NAME = $previousRefName
		}
		Write-PackageMetadata $WinStage "windows/amd64"
		Write-PackageMetadata $LinuxStage "linux/amd64"
  } finally {
    Pop-Location
  }

  $WinTarget = Join-Path $OutputRoot "Win-amd64"
  $LinuxTarget = Join-Path $OutputRoot "linux-amd64"
  Assert-SafePackagePath $WinTarget "Win-amd64"
  Assert-SafePackagePath $LinuxTarget "linux-amd64"
  if (Test-Path -LiteralPath $WinTarget) {
    Remove-Item -LiteralPath $WinTarget -Recurse -Force
  }
  if (Test-Path -LiteralPath $LinuxTarget) {
    Remove-Item -LiteralPath $LinuxTarget -Recurse -Force
  }
  Move-Item -LiteralPath $WinStage -Destination $WinTarget
  Move-Item -LiteralPath $LinuxStage -Destination $LinuxTarget

  $WinArchive = Join-Path $OutputRoot ("Local-AI-Gateway-$Version-win-x64.zip")
  $LinuxArchive = Join-Path $OutputRoot ("Local-AI-Gateway-$Version-linux-amd64.tar.gz")
  $LinuxLegacyArchive = Join-Path $OutputRoot "gateway-linux-amd64.tar.gz"
  foreach ($archive in @($WinArchive,$LinuxArchive,$LinuxLegacyArchive)) {
    if (Test-Path -LiteralPath $archive) {
      Remove-Item -LiteralPath $archive -Force
    }
  }
  Compress-Archive -LiteralPath $WinTarget -DestinationPath $WinArchive -CompressionLevel Optimal
  & $Tar -czf $LinuxArchive -C $OutputRoot "linux-amd64"
  if ($LASTEXITCODE -ne 0) {
    throw "Linux tar archive creation failed"
  }
  Copy-Item -LiteralPath $LinuxArchive -Destination $LinuxLegacyArchive

  $TopChecksums = Join-Path $OutputRoot ("SHA256SUMS-$Version.txt")
  $archiveFiles = @($WinArchive,$LinuxArchive,$LinuxLegacyArchive)
  $archiveLines = foreach ($archive in $archiveFiles) {
    $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $archive).Hash.ToLowerInvariant()
    "$hash  $([System.IO.Path]::GetFileName($archive))"
  }
  $archiveLines | Set-Content -LiteralPath $TopChecksums -Encoding ascii
  $linuxLegacyHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $LinuxLegacyArchive).Hash.ToLowerInvariant()
  "$linuxLegacyHash  gateway-linux-amd64.tar.gz" | Set-Content -LiteralPath (Join-Path $OutputRoot "gateway-linux-amd64.tar.gz.sha256") -Encoding ascii

  Get-ChildItem -LiteralPath $OutputRoot -File | Where-Object { $_.Name -in @(
    [System.IO.Path]::GetFileName($WinArchive),
    [System.IO.Path]::GetFileName($LinuxArchive),
    [System.IO.Path]::GetFileName($LinuxLegacyArchive),
    [System.IO.Path]::GetFileName($TopChecksums),
    "gateway-linux-amd64.tar.gz.sha256"
  ) } | Select-Object Name,Length,LastWriteTime
} finally {
  if (Test-Path -LiteralPath $StagingRoot) {
    Assert-SafePackagePath $StagingRoot ([System.IO.Path]::GetFileName($StagingRoot))
    Remove-Item -LiteralPath $StagingRoot -Recurse -Force
  }
}
