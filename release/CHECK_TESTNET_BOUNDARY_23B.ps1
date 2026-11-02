$ErrorActionPreference="Stop"

$Release=Split-Path -Parent $MyInvocation.MyCommand.Path
$configTestnet=Join-Path $Release "config.testnet.json"
$runApi=Join-Path $Release "RUN_TESTNET_API_18082.ps1"
$runNode=Join-Path $Release "RUN_TESTNET_NODE_18082_13001.ps1"
$disabledPublicRun=Join-Path $Release "RUN_PUBLIC_TESTNET_NODE_DISABLED_UNTIL_APPROVAL.ps1"

"=== QC_TESTNET_BOUNDARY_23B_CHECK ==="
"GIT_GITHUB_TOUCHED=NO"
"REAL_TRANSFER_SENT_BY_THIS_SCRIPT=NO"
"TESTNET_STARTED_BY_THIS_SCRIPT=NO"
"MAINNET_CHAIN_TOUCHED=NO"

if(!(Test-Path $configTestnet)){ throw "MISSING_config.testnet.json" }
if(!(Test-Path $runApi)){ throw "MISSING_RUN_TESTNET_API_18082.ps1" }
if(!(Test-Path $runNode)){ throw "MISSING_RUN_TESTNET_NODE_18082_13001.ps1" }
if(!(Test-Path $disabledPublicRun)){ throw "MISSING_RUN_PUBLIC_TESTNET_NODE_DISABLED_UNTIL_APPROVAL.ps1" }

$c=Get-Content $configTestnet -Raw | ConvertFrom-Json

"CONFIG_NETWORK=$($c.network)"
"CONFIG_NETWORK_MAGIC=$($c.network_magic)"
"CONFIG_CHAIN_FILE=$($c.chain_file)"
"CONFIG_HTTP_PORT=$($c.http_port)"
"CONFIG_P2P_PORT=$($c.p2p_port)"
"CONFIG_TOTAL_SUPPLY=$($c.total_supply)"
"CONFIG_FAUCET_ENABLED=$($c.faucet_enabled)"
"CONFIG_PREMINE_ENABLED=$($c.premine_enabled)"

if($c.network -ne "testnet"){ throw "BOUNDARY_FAIL_NETWORK" }
if($c.network_magic -ne "QC_TESTNET_V1"){ throw "BOUNDARY_FAIL_MAGIC" }
if($c.chain_file -ne "chain_data_testnet.dat"){ throw "BOUNDARY_FAIL_CHAIN_FILE" }
if($c.http_port -ne ":18082"){ throw "BOUNDARY_FAIL_HTTP_PORT" }
if($c.p2p_port -ne ":13001"){ throw "BOUNDARY_FAIL_P2P_PORT" }
if([int]$c.total_supply -ne 25500000){ throw "BOUNDARY_FAIL_TOTAL_SUPPLY" }
if($c.faucet_enabled -ne $false){ throw "BOUNDARY_FAIL_FAUCET" }
if($c.premine_enabled -ne $false){ throw "BOUNDARY_FAIL_PREMINE" }

$apiText=Get-Content $runApi -Raw
$nodeText=Get-Content $runNode -Raw
$disabledText=Get-Content $disabledPublicRun -Raw

foreach($name in @("QC_TESTNET","QC_PUBLIC_TESTNET","QC_NET_MAGIC","QC_NETWORK","QC_CHAIN_MODE","QC_CHAIN_FILE","QC_HTTP_PORT","QC_P2P_PORT")){
  if($apiText -notmatch [regex]::Escape($name)){ throw "BOUNDARY_FAIL_API_SCRIPT_MISSING_$name" }
  if($nodeText -notmatch [regex]::Escape($name)){ throw "BOUNDARY_FAIL_NODE_SCRIPT_MISSING_$name" }
}

if($apiText -notmatch 'QC_PUBLIC_TESTNET="0"'){ throw "BOUNDARY_FAIL_API_NOT_LOCAL_DEV_MODE" }
if($nodeText -notmatch 'QC_PUBLIC_TESTNET="0"'){ throw "BOUNDARY_FAIL_NODE_NOT_LOCAL_DEV_MODE" }
if($disabledText -notmatch "PUBLIC_TESTNET_DISABLED_UNTIL_APPROVAL"){ throw "BOUNDARY_FAIL_PUBLIC_DISABLED_SCRIPT" }

"LOCAL_DEV_TESTNET_MODE=YES"
"PUBLIC_TESTNET_MODE=DISABLED_UNTIL_APPROVAL"
"MUTATING_ENDPOINTS_PUBLIC_EXPOSURE=NOT_APPROVED"
"QC_TESTNET_BOUNDARY_23B_CHECK_RESULT=PASS"
