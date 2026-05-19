param(
  [string]$ApiBase = "http://127.0.0.1:8082",
  [string]$Address = "",
  [string]$ReleaseDir = "",
  [int]$DelayMs = 5000,
  [int]$DifficultyBits = 12
)

if ([string]::IsNullOrWhiteSpace($ReleaseDir)) {
  $ReleaseDir = Split-Path -Parent $MyInvocation.MyCommand.Path
}

Set-Location $ReleaseDir

$env:QC_NODE_DIR = $ReleaseDir
$env:QC_HTTP_PORT = ":8082"
$env:QC_API_BASE = $ApiBase
$env:QC_DIFFICULTY_BITS = "$DifficultyBits"
$env:QC_MINING_DELAY_MS = "$DelayMs"

if ([string]::IsNullOrWhiteSpace($Address)) {
  $minerAddrPath = Join-Path $ReleaseDir "miner_address.txt"
  $walletAddrPath = Join-Path $ReleaseDir "wallet_address.txt"

  if (Test-Path $minerAddrPath) {
    $Address = (Get-Content $minerAddrPath -Raw).Trim()
  } elseif (Test-Path $walletAddrPath) {
    $Address = (Get-Content $walletAddrPath -Raw).Trim()
  }
}

$Address = ($Address -replace '"','').Trim()
$stopPath = Join-Path $ReleaseDir "miner_stop.flag"
$logPath = Join-Path $ReleaseDir "miner_out.log"
$exePath = Join-Path $ReleaseDir "quantumcoin.exe"

function SafeLog([string]$msg, [ConsoleColor]$Color = [ConsoleColor]::Gray) {
  $line = "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') $msg"
  Write-Host $line -ForegroundColor $Color
  try {
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($line + [Environment]::NewLine)
    $fs = [System.IO.File]::Open($logPath, [System.IO.FileMode]::Append, [System.IO.FileAccess]::Write, [System.IO.FileShare]::ReadWrite)
    try { $fs.Write($bytes, 0, $bytes.Length) } finally { $fs.Dispose() }
  } catch {}
}

function EnsureApi() {
  try {
    Invoke-WebRequest -Uri "$ApiBase/api/telemetry" -UseBasicParsing -TimeoutSec 3 | Out-Null
    return $true
  } catch {
    SafeLog "API not reachable; starting API..."
    try {
      $env:QC_DIFFICULTY_BITS = "$DifficultyBits"
      $env:QC_MINING_DELAY_MS = "$DelayMs"
      Start-Process -FilePath $exePath -ArgumentList "api" -WorkingDirectory $ReleaseDir -WindowStyle Hidden
      Start-Sleep -Seconds 4
      Invoke-WebRequest -Uri "$ApiBase/api/telemetry" -UseBasicParsing -TimeoutSec 5 | Out-Null
      SafeLog "API started."
      return $true
    } catch {
      SafeLog ("API start failed: " + $_.Exception.Message)
      return $false
    }
  }
}

if (Test-Path $stopPath) {
  Remove-Item $stopPath -Force -ErrorAction SilentlyContinue
}

SafeLog "QuantumCoin stable API-mempool miner started"
SafeLog "Runtime=$ReleaseDir"
SafeLog "API=$ApiBase"
SafeLog "Address=$Address"
SafeLog "DifficultyBits=$DifficultyBits"
SafeLog "DelayMs=$DelayMs"

while (!(Test-Path $stopPath)) {
  if ([string]::IsNullOrWhiteSpace($Address)) {
    SafeLog "ERROR missing miner address"
    Start-Sleep -Seconds 5
    continue
  }

  if (!(EnsureApi)) {
    Start-Sleep -Seconds 5
    continue
  }

  try {
    $body = @{ address = $Address } | ConvertTo-Json -Compress
    $resp = Invoke-WebRequest -Uri "$ApiBase/api/mine" -Method POST -Body $body -ContentType "application/json" -UseBasicParsing -TimeoutSec 120
    SafeLog ("BLOCK " + $resp.Content) Green
  } catch {
    SafeLog ("ERROR mine failed: " + $_.Exception.Message)
    Start-Sleep -Seconds 3
  }

  Start-Sleep -Milliseconds $DelayMs
}

SafeLog "QuantumCoin stable API-mempool miner stopped"

