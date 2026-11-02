param(
  [Parameter(Mandatory = $true)]
  [string]$OutputPath,
  [string]$Version = $env:GITHUB_REF_NAME,
  [string]$Commit = "unknown",
  [string]$Platform = "unknown",
  [string]$PackageName = "local-ai-gateway"
)

$ErrorActionPreference = "Stop"
$GoCommand = Get-Command go -ErrorAction SilentlyContinue
if ($GoCommand) {
  $Go = $GoCommand.Source
} else {
  $Go = Join-Path (Split-Path -Parent $PSScriptRoot) ".tools\go\bin\go.exe"
  if (-not (Test-Path -LiteralPath $Go)) {
    throw "Go toolchain was not found in PATH or .tools/go/bin/go.exe"
  }
}
$components = @()
$goVersion = (& $Go env GOVERSION).Trim()
if ($LASTEXITCODE -ne 0) { throw "read Go version failed" }
$moduleLines = @(& $Go list -m -f '{{.Path}}|{{.Version}}|{{.Sum}}' all)
if ($LASTEXITCODE -ne 0) {
  throw "go list failed with exit code $LASTEXITCODE"
}

# The embedded Vue runtime is shipped with the application. Build-only npm
# packages are intentionally not represented as deployed runtime dependencies.
$lockPath = Join-Path (Split-Path -Parent $PSScriptRoot) "package-lock.json"
$vueVersion = (& node -e "process.stdout.write(require(process.argv[1]).packages['node_modules/vue'].version)" $lockPath).Trim()
if ($LASTEXITCODE -ne 0) { throw "read embedded Vue version failed" }
if ([string]::IsNullOrWhiteSpace($vueVersion)) { throw "Vue runtime version missing from lockfile" }
$components += [ordered]@{
  type = "library"
  name = "vue"
  version = $vueVersion
  purl = "pkg:npm/vue@$vueVersion"
  scope = "required"
}
$moduleLines | ForEach-Object {
  $fields = $_ -split '\|', 3
  $path = $fields[0]
  $moduleVersion = if ($fields.Count -gt 1 -and $fields[1]) { $fields[1] } else { "devel" }
  $component = [ordered]@{
    type = "library"
    name = $path
    version = $moduleVersion
    purl = "pkg:golang/$path@$moduleVersion"
  }
  $components += $component
}

$sbom = [ordered]@{
  bomFormat = "CycloneDX"
  specVersion = "1.5"
  serialNumber = "urn:uuid:$([Guid]::NewGuid())"
  version = 1
  metadata = @{
    timestamp = [DateTime]::UtcNow.ToString("o")
    component = @{
      type = "application"
      name = $PackageName
      version = $Version
      properties = @(
        @{ name = "${PackageName}:commit"; value = $Commit }
        @{ name = "${PackageName}:platform"; value = $Platform }
        @{ name = "${PackageName}:go-toolchain"; value = $goVersion }
      )
    }
  }
  components = $components
}

$directory = Split-Path -Parent $OutputPath
if ($directory) {
  New-Item -ItemType Directory -Force $directory | Out-Null
}
$sbom | ConvertTo-Json -Depth 8 | Set-Content -Encoding utf8 $OutputPath
