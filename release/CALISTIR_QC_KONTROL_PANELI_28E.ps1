$ErrorActionPreference="Stop"
$Release=Split-Path -Parent $MyInvocation.MyCommand.Path
$html=Join-Path $Release "qc_kontrol_paneli\index.html"
"=== QC_KONTROL_PANELI_28E_CALISTIR ==="
"GIT_GITHUB_TOUCHED=NO"
"REAL_TRANSFER_SENT_BY_THIS_SCRIPT=NO"
"TESTNET_STARTED_BY_THIS_SCRIPT=NO"
"MAINNET_CHAIN_TOUCHED=NO"
"PANEL_DILI=TURKCE"
if(!(Test-Path $html)){ throw "KONTROL_PANELI_HTML_YOK" }
Start-Process $html
"KONTROL_PANELI_ACILDI=EVET"
