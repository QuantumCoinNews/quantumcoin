$ErrorActionPreference="Stop"
$Release=Split-Path -Parent $MyInvocation.MyCommand.Path
$panel=Join-Path $Release "qc_kontrol_paneli"
$html=Join-Path $panel "index.html"
$json=Join-Path $panel "kontrol_paneli_veri.json"
$exe=Join-Path $Release "qc-kontrol-paneli.exe"
"=== QC_KONTROL_PANELI_28G_CHECK ==="
"GIT_GITHUB_TOUCHED=NO"
"REAL_TRANSFER_SENT_BY_THIS_SCRIPT=NO"
"TESTNET_STARTED_BY_THIS_SCRIPT=NO"
"MAINNET_CHAIN_TOUCHED=NO"
"PANEL_DILI=TURKCE"
if(!(Test-Path $html)){ throw "HTML_YOK" }
if(!(Test-Path $json)){ throw "JSON_YOK" }
if(!(Test-Path $exe)){ throw "EXE_YOK" }
$raw=Get-Content $html -Raw
$txt=[System.Net.WebUtility]::HtmlDecode($raw)
foreach($m in @("QuantumCoin Kontrol Paneli","AI","Blokzincir","Güvenlik","İzleme / Operasyon","Telegram Game","Cüzdan + Madenci","Gelecek Modüller","Türkçe yerel kontrol ekranı","Okuma modu")){
  if($txt -notmatch [regex]::Escape($m)){ throw "PANEL_MARKER_YOK_$m" }
  "MARKER_OK=$m"
}
$j=Get-Content $json -Raw | ConvertFrom-Json
"VERI_SURUM=$($j.surum)"
"VERI_DIL=$($j.dil)"
if($j.dil -ne "Türkçe"){ throw "DIL_TURKCE_DEGIL" }
$h=(Get-FileHash $exe -Algorithm SHA256).Hash
"EXE_OK=$exe"
"EXE_HASH=$h"
"QC_KONTROL_PANELI_28G_CHECK_RESULT=PASS"