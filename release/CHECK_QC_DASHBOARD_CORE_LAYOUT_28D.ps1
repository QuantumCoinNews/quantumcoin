$ErrorActionPreference="Stop"
$Release=Split-Path -Parent $MyInvocation.MyCommand.Path
$layout=Join-Path $Release "QC_DASHBOARD_CORE_LAYOUT_28D.md"
$manifest=Join-Path $Release "QC_DASHBOARD_CORE_MODULES_28D.json"
"=== QC_DASHBOARD_STEP_28D_SECURITY_MONITORING_CHECK ==="
"GIT_GITHUB_TOUCHED=NO"
"REAL_TRANSFER_SENT_BY_THIS_SCRIPT=NO"
"TESTNET_STARTED_BY_THIS_SCRIPT=NO"
"MAINNET_CHAIN_TOUCHED=NO"
if(!(Test-Path $layout)){ throw "LAYOUT_MISSING" }
if(!(Test-Path $manifest)){ throw "MANIFEST_MISSING" }
$j=Get-Content $manifest -Raw | ConvertFrom-Json
"VERSION=$($j.version)"
"MODULE_COUNT=$($j.modules.Count)"
if($j.modules.Count -ne 7){ throw "EXPECTED_7_MODULES" }
$required=@("ai","blockchain","security","monitoring_ops","telegram_game","wallet_miner","future_modules")
foreach($id in $required){
  $hit=$j.modules | Where-Object { $_.id -eq $id }
  if(!$hit){ throw "MISSING_MODULE_$id" }
  "MODULE_OK=$id"
}
$txt=Get-Content $layout -Raw
foreach($m in @("Security center","Monitoring folder purpose","Monitoring is not the dashboard itself","Yellow warnings","Read-only first","Token Factory later")){
  if($txt -notmatch [regex]::Escape($m)){ throw "LAYOUT_MARKER_MISSING_$m" }
  "LAYOUT_MARKER_OK=$m"
}
"QC_DASHBOARD_STEP_28D_SECURITY_MONITORING_CHECK_RESULT=PASS"
