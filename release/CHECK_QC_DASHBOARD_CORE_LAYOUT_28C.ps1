$ErrorActionPreference="Stop"
$Release=Split-Path -Parent $MyInvocation.MyCommand.Path
$layout=Join-Path $Release "QC_DASHBOARD_CORE_LAYOUT_28C.md"
$manifest=Join-Path $Release "QC_DASHBOARD_CORE_MODULES_28C.json"
"=== QC_DASHBOARD_STEP_28C_CORE_LAYOUT_CHECK ==="
"GIT_GITHUB_TOUCHED=NO"
"REAL_TRANSFER_SENT_BY_THIS_SCRIPT=NO"
"TESTNET_STARTED_BY_THIS_SCRIPT=NO"
"MAINNET_CHAIN_TOUCHED=NO"
if(!(Test-Path $layout)){ throw "LAYOUT_MISSING" }
if(!(Test-Path $manifest)){ throw "MANIFEST_MISSING" }
$j=Get-Content $manifest -Raw | ConvertFrom-Json
"VERSION=$($j.version)"
"MODULE_COUNT=$($j.modules.Count)"
if($j.modules.Count -ne 5){ throw "EXPECTED_5_MODULES" }
$required=@("ai","blockchain","telegram_game","wallet_miner","system_ops_future")
foreach($id in $required){
  $hit=$j.modules | Where-Object { $_.id -eq $id }
  if(!$hit){ throw "MISSING_MODULE_$id" }
  "MODULE_OK=$id"
}
$txt=Get-Content $layout -Raw
foreach($m in @("AI","Blockchain","Telegram Game","Wallet + Miner","System / Ops / Future Modules","Token Factory should not be a main dashboard center yet","Read-only first")){
  if($txt -notmatch [regex]::Escape($m)){ throw "LAYOUT_MARKER_MISSING_$m" }
  "LAYOUT_MARKER_OK=$m"
}
"QC_DASHBOARD_STEP_28C_CORE_LAYOUT_CHECK_RESULT=PASS"
