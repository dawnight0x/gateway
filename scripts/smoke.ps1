param(
  [ValidateRange(1, 65535)]
  [int]$Port = 28787
)

$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
Set-Location -LiteralPath $Root

$Go = Join-Path $Root ".tools\go\bin\go.exe"
if (-not (Test-Path -LiteralPath $Go)) {
  $Go = (Get-Command go -ErrorAction Stop).Source
}
$RunID = "$PID-$([Guid]::NewGuid().ToString('N'))"
$SmokeDir = Join-Path ([System.IO.Path]::GetTempPath()) "local-ai-gateway-smoke-$RunID"
$Exe = Join-Path $SmokeDir "gateway-smoke.exe"
$DB = Join-Path $SmokeDir "gateway.db"
$Secret = Join-Path $SmokeDir "secret.key"
$Out = Join-Path $SmokeDir "stdout.log"
$Err = Join-Path $SmokeDir "stderr.log"
$Config = Join-Path $SmokeDir "missing-config.yaml"
$proc = $null

$environmentNames = @(
  "GATEWAY_CONFIG",
  "GATEWAY_TRAY",
  "GATEWAY_HOST",
  "GATEWAY_PUBLIC_HOST",
  "GATEWAY_PORT",
  "GATEWAY_DB",
  "GATEWAY_SECRET_PATH",
  "GATEWAY_ADMIN_TOKEN_FILE",
  "GATEWAY_ADMIN_TOKEN",
  "GATEWAY_PROXY_TOKEN",
  "GATEWAY_OPEN_BROWSER_ON_DUPLICATE"
)
$previousEnvironment = @{}
foreach ($name in $environmentNames) {
  $item = Get-Item -LiteralPath "Env:$name" -ErrorAction SilentlyContinue
  $previousEnvironment[$name] = if ($item) { $item.Value } else { $null }
}

try {
  if (Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue) {
    throw "Smoke-test port $Port is already in use; no process was stopped. Choose another port with -Port."
  }

  New-Item -ItemType Directory -Force -Path $SmokeDir | Out-Null

  Write-Host "== test =="
  & $Go test -buildvcs=false ./...
  if ($LASTEXITCODE -ne 0) {
    throw "go test failed with exit code $LASTEXITCODE"
  }

  Write-Host "== build =="
  & $Go build -buildvcs=false -ldflags "-H=windowsgui" -o $Exe .\cmd\gateway
  if ($LASTEXITCODE -ne 0) {
    throw "go build failed with exit code $LASTEXITCODE"
  }

  $env:GATEWAY_CONFIG = $Config
  $env:GATEWAY_TRAY = "false"
  $env:GATEWAY_HOST = "127.0.0.1"
  $env:GATEWAY_PUBLIC_HOST = "localhost"
  $env:GATEWAY_PORT = "$Port"
  $env:GATEWAY_DB = $DB
  $env:GATEWAY_SECRET_PATH = $Secret
  $env:GATEWAY_ADMIN_TOKEN_FILE = Join-Path $SmokeDir "admin.token"
  $env:GATEWAY_ADMIN_TOKEN = "smoke-admin-token"
  $env:GATEWAY_PROXY_TOKEN = "smoke-proxy-token"
  $env:GATEWAY_OPEN_BROWSER_ON_DUPLICATE = "false"

  Write-Host "== start gateway =="
  $proc = Start-Process -FilePath $Exe -WorkingDirectory $Root -WindowStyle Hidden -PassThru -RedirectStandardOutput $Out -RedirectStandardError $Err

  $healthy = $false
  for ($i = 0; $i -lt 30; $i++) {
    try {
      $health = Invoke-RestMethod -Uri "http://127.0.0.1:$Port/health" -TimeoutSec 1
      if ($health.status -eq "ok") {
        $healthy = $true
        break
      }
    } catch {
      Start-Sleep -Milliseconds 200
    }
  }
  if (-not $healthy) {
    throw "gateway did not become healthy; stderr: $(Get-Content -Raw $Err -ErrorAction SilentlyContinue)"
  }

  Write-Host "== duplicate launch should exit quickly =="
  $dupeOut = Join-Path $SmokeDir "dupe.stdout.log"
  $dupeErr = Join-Path $SmokeDir "dupe.stderr.log"
  $dupe = Start-Process -FilePath $Exe -WorkingDirectory $Root -WindowStyle Hidden -PassThru -RedirectStandardOutput $dupeOut -RedirectStandardError $dupeErr
  if (-not $dupe.WaitForExit(5000)) {
    Stop-Process -Id $dupe.Id -Force -ErrorAction SilentlyContinue
    throw "duplicate launch did not exit"
  }

  Write-Host "== endpoints =="
  Invoke-RestMethod -Uri "http://127.0.0.1:$Port/status" -Headers @{ "Authorization" = "Bearer smoke-proxy-token" } -TimeoutSec 3 | ConvertTo-Json -Depth 5
  try {
    Invoke-RestMethod -Uri "http://127.0.0.1:$Port/admin/api/dashboard" -TimeoutSec 3 | Out-Null
    throw "admin API accepted missing token"
  } catch {
    if ($_.Exception.Response.StatusCode.value__ -ne 401) {
      throw
    }
  }
  Invoke-RestMethod -Uri "http://127.0.0.1:$Port/admin/api/dashboard" -Headers @{ "X-Admin-Token" = "smoke-admin-token" } -TimeoutSec 3 | ConvertTo-Json -Depth 5
  $admin = Invoke-WebRequest -UseBasicParsing -Uri "http://127.0.0.1:$Port/admin/" -TimeoutSec 3
  $css = Invoke-WebRequest -UseBasicParsing -Uri "http://127.0.0.1:$Port/admin/assets/styles.css" -TimeoutSec 3
  $js = Invoke-WebRequest -UseBasicParsing -Uri "http://127.0.0.1:$Port/admin/assets/app.js" -TimeoutSec 3
  if ($admin.Headers["X-Content-Type-Options"] -ne "nosniff") {
    throw "admin security headers are missing"
  }
  Write-Host "admin=$($admin.StatusCode) css=$($css.Content.Length) js=$($js.Content.Length)"

  if (-not (Test-Path -LiteralPath (Join-Path $SmokeDir "gateway.log"))) {
    throw "gateway.log was not created"
  }

  Write-Host "SMOKE OK port=$Port"
} finally {
  if ($proc -and -not $proc.HasExited) {
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
    $proc.WaitForExit()
  }
  foreach ($name in $environmentNames) {
    if ($null -eq $previousEnvironment[$name]) {
      Remove-Item -LiteralPath "Env:$name" -ErrorAction SilentlyContinue
    } else {
      Set-Item -LiteralPath "Env:$name" -Value $previousEnvironment[$name]
    }
  }
  $tempRoot = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath()).TrimEnd('\') + '\'
  $resolvedSmokeDir = [System.IO.Path]::GetFullPath($SmokeDir)
  if (-not $resolvedSmokeDir.StartsWith($tempRoot, [System.StringComparison]::OrdinalIgnoreCase) -or
      -not ([System.IO.Path]::GetFileName($resolvedSmokeDir)).StartsWith("local-ai-gateway-smoke-")) {
    throw "refusing to clean unexpected smoke directory: $resolvedSmokeDir"
  }
  if (Test-Path -LiteralPath $resolvedSmokeDir) {
    Remove-Item -LiteralPath $resolvedSmokeDir -Recurse -Force
  }
}
