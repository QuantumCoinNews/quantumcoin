$ErrorActionPreference="Stop"
$Release=Split-Path -Parent $MyInvocation.MyCommand.Path
$backupRoot=Join-Path $Release "_testnet_backups"
$stamp=Get-Date -Format "yyyyMMdd_HHmmss"
$dest=Join-Path $backupRoot "testnet_backup_$stamp"
"=== QC_TESTNET_BACKUP_25A ==="
"GIT_GITHUB_TOUCHED=NO"
"REAL_TRANSFER_SENT_BY_THIS_SCRIPT=NO"
"TESTNET_STARTED_BY_THIS_SCRIPT=NO"
"MAINNET_CHAIN_TOUCHED=NO"
"RESTORE_PERFORMED_BY_THIS_SCRIPT=NO"
if(!(Test-Path $backupRoot)){ New-Item -ItemType Directory -Path $backupRoot | Out-Null }
New-Item -ItemType Directory -Path $dest | Out-Null
$files=@(
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
$manifest=@()
foreach($name in $files){
  $src=Join-Path $Release $name
  if(Test-Path $src){
    Copy-Item $src (Join-Path $dest $name) -Force
    $h=(Get-FileHash $src -Algorithm SHA256).Hash
    $len=(Get-Item $src).Length
    $manifest += "$name|$len|$h"
    "BACKED_UP=$name LENGTH=$len HASH=$h"
  }else{
    $manifest += "$name|MISSING|"
    "SKIPPED_MISSING=$name"
  }
}
Set-Content -Path (Join-Path $dest "manifest.sha256.txt") -Value $manifest -Encoding UTF8
"BACKUP_FOLDER=$dest"
"QC_TESTNET_BACKUP_25A_RESULT=PASS"
