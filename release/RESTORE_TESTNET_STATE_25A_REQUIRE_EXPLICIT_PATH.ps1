param(
  [Parameter(Mandatory=$true)]
  [string]$BackupFolder
)
$ErrorActionPreference="Stop"
$Release=Split-Path -Parent $MyInvocation.MyCommand.Path
"=== QC_TESTNET_RESTORE_25A_REQUIRE_EXPLICIT_PATH ==="
"GIT_GITHUB_TOUCHED=NO"
"REAL_TRANSFER_SENT_BY_THIS_SCRIPT=NO"
"TESTNET_STARTED_BY_THIS_SCRIPT=NO"
"MAINNET_CHAIN_TOUCHED=NO"
if(!(Test-Path $BackupFolder)){ throw "BACKUP_FOLDER_NOT_FOUND" }
if(!(Test-Path (Join-Path $BackupFolder "manifest.sha256.txt"))){ throw "MANIFEST_NOT_FOUND" }
$allowed=@(
  "config.testnet.json",
  "chain_data_testnet.dat",
  "bonus_store_testnet.json",
  "wallet_data_testnet.json",
  "RUN_TESTNET_API_18082.ps1",
  "STOP_TESTNET_API_18082.ps1",
  "RUN_TESTNET_NODE_18082_13001.ps1",
  "STOP_TESTNET_NODE_18082_13001.ps1",
  "TESTNET_DEV_MODE_BOUNDARY_23B.md",
  "TESTNET_RUNBOOK_24A.md"
)
foreach($name in $allowed){
  $src=Join-Path $BackupFolder $name
  if(Test-Path $src){
    Copy-Item $src (Join-Path $Release $name) -Force
    "RESTORED_TESTNET_FILE=$name"
  }
}
if(Test-Path (Join-Path $BackupFolder "chain_data.dat")){ throw "SAFETY_FAIL_BACKUP_CONTAINS_MAINNET_CHAIN" }
"QC_TESTNET_RESTORE_25A_RESULT=PASS"
