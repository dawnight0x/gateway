param(
  [string]$OutputRoot = "",
  [string]$Version = "",
  [string]$PackageName = "",
  [switch]$AllowDirty
)

$ErrorActionPreference = "Stop"
$Root = [System.IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot))
$OutputRoot = if ([string]::IsNullOrWhiteSpace($OutputRoot)) {
  Join-Path $Root "..\gateway-beta"
} else {
  $OutputRoot
}
$ReleaseMetadataPath = Join-Path $Root "release.json"
try {
  $ReleaseMetadata = Get-Content -Raw -LiteralPath $ReleaseMetadataPath | ConvertFrom-Json -ErrorAction Stop
} catch {
  throw "read release metadata $ReleaseMetadataPath`: $($_.Exception.Message)"
}
if ([string]::IsNullOrWhiteSpace($Version)) {
  $Version = [string]$ReleaseMetadata.version
}
if ([string]::IsNullOrWhiteSpace($PackageName)) {
  $PackageName = [string]$ReleaseMetadata.packageName
}
$OutputRoot = [System.IO.Path]::GetFullPath($OutputRoot).TrimEnd('\')
$outputLeaf = [System.IO.Path]::GetFileName($OutputRoot)
if ([string]::IsNullOrWhiteSpace($outputLeaf) -or $OutputRoot -eq [System.IO.Path]::GetPathRoot($OutputRoot)) {
  throw "refusing to package into unsafe output root: $OutputRoot"
}
$versionMatch = [regex]::Match($Version, '^(?<major>0|[1-9][0-9]*)\.(?<minor>0|[1-9][0-9]*)\.(?<patch>0|[1-9][0-9]*)(?<prerelease>-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$')
if (-not $versionMatch.Success) {
  throw "version must use major.minor.patch with an optional prerelease suffix: $Version"
}
if ($PackageName -notmatch '^[A-Za-z0-9][A-Za-z0-9._-]*$') {
  throw "package name contains unsupported characters: $PackageName"
}
$WindowsMajor = [int]$versionMatch.Groups['major'].Value
$WindowsMinor = [int]$versionMatch.Groups['minor'].Value
$WindowsPatch = [int]$versionMatch.Groups['patch'].Value
foreach ($part in @($WindowsMajor, $WindowsMinor, $WindowsPatch)) {
  if ($part -gt [uint16]::MaxValue) {
    throw "Windows version components must not exceed $([uint16]::MaxValue): $Version"
  }
}

$Go = Join-Path $Root ".tools\go\bin\go.exe"
if (-not (Test-Path -LiteralPath $Go)) {
  $Go = (Get-Command go -ErrorAction Stop).Source
}
$Tar = (Get-Command tar -ErrorAction Stop).Source
$BuiltAt = [DateTime]::UtcNow.ToString("o")
$Commit = "unknown"
$worktreeChanges = @(& git -C $Root status --porcelain=v1 --untracked-files=normal 2>$null)
if ($LASTEXITCODE -ne 0) {
  throw "git status failed with exit code $LASTEXITCODE"
}
if ($worktreeChanges.Count -gt 0 -and -not $AllowDirty) {
  throw "working tree contains uncommitted changes; commit them before packaging or pass -AllowDirty for a development build"
}
try {
  $candidateCommit = (& git -C $Root rev-parse --verify HEAD 2>$null).Trim()
  if ($LASTEXITCODE -eq 0 -and $candidateCommit) {
    $Commit = $candidateCommit
    if ($worktreeChanges.Count -gt 0) {
      $Commit += "-dirty"
    }
  }
} catch {}
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

try {
  Push-Location $Root
  try {
    npm ci
    if ($LASTEXITCODE -ne 0) {
      throw "admin dependency installation failed"
    }
    npm run test:admin
    if ($LASTEXITCODE -ne 0) {
      throw "admin tests failed"
    }
    npm run build:admin
    if ($LASTEXITCODE -ne 0) {
      throw "admin build failed"
    }
    & $Go test ./...
    if ($LASTEXITCODE -ne 0) {
      throw "Go tests failed"
    }

    & (Join-Path $Root "scripts\generate-windows-resources.ps1") `
      -Major $WindowsMajor `
      -Minor $WindowsMinor `
      -Patch $WindowsPatch `
      -Build 0 `
      -ProductVersion $Version

    Invoke-GoBuild "windows" "amd64" ".\cmd\gateway" (Join-Path $WinStage "gateway.exe") "$CommonLdFlags -H=windowsgui"
    Invoke-GoBuild "windows" "amd64" ".\cmd\gateway-backup" (Join-Path $WinStage "gateway-backup.exe") $CommonLdFlags
    Invoke-GoBuild "linux" "amd64" ".\cmd\gateway" (Join-Path $LinuxStage "gateway") $CommonLdFlags
    Invoke-GoBuild "linux" "amd64" ".\cmd\gateway-backup" (Join-Path $LinuxStage "gateway-backup") $CommonLdFlags

    & (Join-Path $Root "scripts\stage-release-package.ps1") -Destination $WinStage -Version $Version -Commit $Commit -BuiltAt $BuiltAt -Platform "windows/amd64" -PackageName $PackageName
    & (Join-Path $Root "scripts\stage-release-package.ps1") -Destination $LinuxStage -Version $Version -Commit $Commit -BuiltAt $BuiltAt -Platform "linux/amd64" -PackageName $PackageName
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

  $WinArchive = Join-Path $OutputRoot ("$PackageName-$Version-win-x64.zip")
  $LinuxArchive = Join-Path $OutputRoot ("$PackageName-$Version-linux-amd64.tar.gz")
  foreach ($archive in @($WinArchive,$LinuxArchive)) {
    if (Test-Path -LiteralPath $archive) {
      Remove-Item -LiteralPath $archive -Force
    }
  }
  Compress-Archive -LiteralPath $WinTarget -DestinationPath $WinArchive -CompressionLevel Optimal
  & $Tar -czf $LinuxArchive -C $OutputRoot "linux-amd64"
  if ($LASTEXITCODE -ne 0) {
    throw "Linux tar archive creation failed"
  }

  $TopChecksums = Join-Path $OutputRoot ("$PackageName-$Version-SHA256SUMS.txt")
  $archiveFiles = @($WinArchive,$LinuxArchive)
  $archiveLines = foreach ($archive in $archiveFiles) {
    $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $archive).Hash.ToLowerInvariant()
    "$hash  $([System.IO.Path]::GetFileName($archive))"
  }
  $archiveText = if ($archiveLines.Count -gt 0) { ($archiveLines -join "`n") + "`n" } else { "" }
  [System.IO.File]::WriteAllText($TopChecksums, $archiveText, [System.Text.Encoding]::ASCII)

  Get-ChildItem -LiteralPath $OutputRoot -File | Where-Object { $_.Name -in @(
    [System.IO.Path]::GetFileName($WinArchive),
    [System.IO.Path]::GetFileName($LinuxArchive),
    [System.IO.Path]::GetFileName($TopChecksums)
  ) } | Select-Object Name,Length,LastWriteTime
} finally {
  if (Test-Path -LiteralPath $StagingRoot) {
    Assert-SafePackagePath $StagingRoot ([System.IO.Path]::GetFileName($StagingRoot))
    Remove-Item -LiteralPath $StagingRoot -Recurse -Force
  }
}
