$ErrorActionPreference="Stop"

$Release=Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $Release

"=== QC_TESTNET_API_18082_LOCAL_START ==="
"GIT_GITHUB_TOUCHED=NO"
"REAL_TRANSFER_SENT_BY_THIS_SCRIPT=NO"
"MAINNET_CHAIN_TOUCHED=NO"
"TESTNET_CHAIN_FILE=chain_data_testnet.dat"

$exe=Join-Path $Release "quantumcoin.exe"
if(!(Test-Path $exe)){
  throw "quantumcoin.exe missing in release folder"
}

$mainnetChain=Join-Path $Release "chain_data.dat"
$testnetChain=Join-Path $Release "chain_data_testnet.dat"

"MAINNET_CHAIN_EXISTS=$((Test-Path $mainnetChain))"
"TESTNET_CHAIN_EXISTS_BEFORE=$((Test-Path $testnetChain))"

$existing=Get-NetTCPConnection -LocalPort 18082 -ErrorAction SilentlyContinue | Where-Object { $_.State -eq "Listen" }
if($existing){
  $pids=($existing | Select-Object -ExpandProperty OwningProcess -Unique) -join ","
  throw "Port 18082 already in use. PID(s): $pids"
}

# QC_TESTNET_ENV_BOUNDARY_V1_BEGIN

# Parent process, terminal, release miner or previous node sessions may leave
# network and difficulty variables behind. Remove every conflicting signal
# before applying the canonical local-testnet profile.
$inheritedBoundaryVariables = @(
"QC_TESTNET",
"QC_TESTNET_PUBLIC_API",
"QC_API_PUBLIC_TESTNET",
"QC_PUBLIC_TESTNET",
"QC_MAINNET",
"QC_NET_MAGIC",
"QC_NETWORK",
"QC_NETWORK_MODE",
"QC_CHAIN_MODE",
"QC_MODE",
"QC_CHAIN_FILE",
"QC_BONUS_FILE",
"QC_WALLET_FILE",
"QC_HTTP_PORT",
"HTTP_PORT",
"QC_API_LISTEN",
"QC_P2P_PORT",
"QC_DEFAULT_DIFFICULTY_BITS",
"QC_DIFFICULTY_BITS",
"QC_TESTNET_FAUCET_ENABLED",
"QC_TESTNET_PREMINE_ENABLED"
)

foreach ($variableName in $inheritedBoundaryVariables) {
Remove-Item `
-LiteralPath ("Env:{0}" -f $variableName) `
-ErrorAction SilentlyContinue
}

# Canonical isolated local-testnet profile.
$env:QC_TESTNET = "1"
$env:QC_TESTNET_PUBLIC_API = "0"
$env:QC_API_PUBLIC_TESTNET = "0"
$env:QC_PUBLIC_TESTNET = "0"

$env:QC_NET_MAGIC = "QC_TESTNET_V1"
$env:QC_NETWORK = "testnet"
$env:QC_NETWORK_MODE = "testnet"
$env:QC_CHAIN_MODE = "testnet"
$env:QC_MODE = "testnet"

$env:QC_CHAIN_FILE = "chain_data_testnet.dat"
$env:QC_BONUS_FILE = "bonus_store_testnet.json"
$env:QC_WALLET_FILE = "wallet_data_testnet.json"

$env:QC_HTTP_PORT = ":18082"
$env:QC_API_LISTEN = "127.0.0.1:18082"
$env:QC_P2P_PORT = ":13001"
$env:QC_DEFAULT_DIFFICULTY_BITS = "8"

$env:QC_TESTNET_FAUCET_ENABLED = "0"
$env:QC_TESTNET_PREMINE_ENABLED = "0"

# QC_DIFFICULTY_BITS intentionally remains absent. Testnet does not permit
# the temporary fixed-difficulty override used by local release mining.
# HTTP_PORT intentionally remains absent; QC_HTTP_PORT is authoritative.
# QC_MAINNET intentionally remains absent.

# QC_TESTNET_ENV_BOUNDARY_V1_END
"QC_TESTNET=$env:QC_TESTNET"
"QC_NET_MAGIC=$env:QC_NET_MAGIC"
"QC_NETWORK=$env:QC_NETWORK"
"QC_CHAIN_MODE=$env:QC_CHAIN_MODE"
"QC_CHAIN_FILE=$env:QC_CHAIN_FILE"
"QC_HTTP_PORT=$env:QC_HTTP_PORT"
"QC_P2P_PORT=$env:QC_P2P_PORT"

$proc=Start-Process -FilePath $exe `
  -ArgumentList "api" `
  -WorkingDirectory $Release `
  -WindowStyle Hidden `
  -PassThru

"STARTED_TESTNET_API_PID=$($proc.Id)"
Start-Sleep -Seconds 6

try{
  $r=Invoke-WebRequest -Uri "http://127.0.0.1:18082/api/network/label" -UseBasicParsing -TimeoutSec 20 -ErrorAction Stop
  "TESTNET_NETWORK_LABEL=$($r.Content)"
}catch{
  "TESTNET_NETWORK_LABEL_ERROR=$($_.Exception.Message)"
}

try{
  $h=Invoke-WebRequest -Uri "http://127.0.0.1:18082/api/health" -UseBasicParsing -TimeoutSec 20 -ErrorAction Stop
  "TESTNET_HEALTH=$($h.Content)"
}catch{
  "TESTNET_HEALTH_ERROR=$($_.Exception.Message)"
}

"TESTNET_CHAIN_EXISTS_AFTER=$((Test-Path $testnetChain))"
"QC_TESTNET_API_18082_LOCAL_START_DONE"
