$ErrorActionPreference="Stop"
$Release=Split-Path -Parent $MyInvocation.MyCommand.Path
$exe=Join-Path $Release "qc-kontrol-paneli.exe"
"=== QC_KONTROL_PANELI_28G_CALISTIR ==="
"GIT_GITHUB_TOUCHED=NO"
"REAL_TRANSFER_SENT_BY_THIS_SCRIPT=NO"
"TESTNET_STARTED_BY_THIS_SCRIPT=NO"
"MAINNET_CHAIN_TOUCHED=NO"
"PANEL_DILI=TURKCE"
if(!(Test-Path $exe)){ throw "PANEL_EXE_YOK" }
Start-Process $exe
"KONTROL_PANELI_EXE_ACILDI=EVET"