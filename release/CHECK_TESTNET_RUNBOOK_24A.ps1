$ErrorActionPreference="Stop"
$Release=Split-Path -Parent $MyInvocation.MyCommand.Path
"=== QC_TESTNET_RUNBOOK_24A_CHECK ==="
"GIT_GITHUB_TOUCHED=NO"
"REAL_TRANSFER_SENT_BY_THIS_SCRIPT=NO"
"TESTNET_STARTED_BY_THIS_SCRIPT=NO"
"MAINNET_CHAIN_TOUCHED=NO"
$required=@(
  "config.testnet.json",
  "RUN_TESTNET_API_18082.ps1",
  "STOP_TESTNET_API_18082.ps1",
  "RUN_TESTNET_NODE_18082_13001.ps1",
  "STOP_TESTNET_NODE_18082_13001.ps1",
  "TESTNET_DEV_MODE_BOUNDARY_23B.md",
  "CHECK_TESTNET_BOUNDARY_23B.ps1",
  "RUN_PUBLIC_TESTNET_NODE_DISABLED_UNTIL_APPROVAL.ps1",
  "TESTNET_RUNBOOK_24A.md"
)
foreach($name in $required){
  $p=Join-Path $Release $name
  if(!(Test-Path $p)){ throw "RUNBOOK_MISSING_$name" }
  "FOUND=$name"
}
$runbook=Get-Content (Join-Path $Release "TESTNET_RUNBOOK_24A.md") -Raw
$markers=@(
  "LOCAL DEV TESTNET",
  "QC_TESTNET_V1",
  "18082",
  "13001",
  "chain_data_testnet.dat",
  "chain_data.dat",
  "Public testnet is disabled",
  "25 backup rollback plan",
  "26 first local testnet dry-run"
)
foreach($m in $markers){
  if($runbook -notmatch [regex]::Escape($m)){ throw "RUNBOOK_MARKER_MISSING_$m" }
  "MARKER_OK=$m"
}
"QC_TESTNET_RUNBOOK_24A_CHECK_RESULT=PASS"
