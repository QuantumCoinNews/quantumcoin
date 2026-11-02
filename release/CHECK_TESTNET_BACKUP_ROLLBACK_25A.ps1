$ErrorActionPreference="Stop"
$Release=Split-Path -Parent $MyInvocation.MyCommand.Path
"=== QC_TESTNET_BACKUP_ROLLBACK_25A_CHECK ==="
"GIT_GITHUB_TOUCHED=NO"
"REAL_TRANSFER_SENT_BY_THIS_SCRIPT=NO"
"TESTNET_STARTED_BY_THIS_SCRIPT=NO"
"MAINNET_CHAIN_TOUCHED=NO"
$required=@(
  "TESTNET_BACKUP_ROLLBACK_25A.md",
  "BACKUP_TESTNET_STATE_25A.ps1",
  "RESTORE_TESTNET_STATE_25A_REQUIRE_EXPLICIT_PATH.ps1",
  "CHECK_TESTNET_BACKUP_ROLLBACK_25A.ps1"
)
foreach($name in $required){
  $p=Join-Path $Release $name
  if(!(Test-Path $p)){ throw "MISSING_$name" }
  "FOUND=$name"
}
$doc=Get-Content (Join-Path $Release "TESTNET_BACKUP_ROLLBACK_25A.md") -Raw
foreach($m in @("LOCAL DEV TESTNET","chain_data_testnet.dat","chain_data.dat","Restore requires explicit backup folder path","26 first local testnet dry-run")){
  if($doc -notmatch [regex]::Escape($m)){ throw "DOC_MARKER_MISSING_$m" }
  "MARKER_OK=$m"
}
"QC_TESTNET_BACKUP_ROLLBACK_25A_CHECK_RESULT=PASS"
