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
$moduleLines = @(& $Go list -m -f '{{.Path}}|{{.Version}}|{{.Sum}}' all)
if ($LASTEXITCODE -ne 0) {
  throw "go list failed with exit code $LASTEXITCODE"
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
