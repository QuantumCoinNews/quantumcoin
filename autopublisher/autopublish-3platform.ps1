param([switch]$Announce=$true)
$ErrorActionPreference='Stop'
Set-Location "C:\Projects\quantumcoin\autopublisher"

# Log + lock
if(-not (Test-Path ".\logs")){ New-Item -ItemType Directory -Path ".\logs" | Out-Null }
$ts=Get-Date -Format "yyyy-MM-dd_HHmmss"
Start-Transcript -Path ".\logs\$ts.log" | Out-Null
$lock = ".\autopublisher.lock"
if(Test-Path $lock){ Write-Host "Another run is in progress. Exiting."; Stop-Transcript | Out-Null; exit 0 }
New-Item -ItemType File -Path $lock | Out-Null

# Logo yoksa kökten kopyala
if(!(Test-Path ".\assets\logo.png")){
  $src="C:\Projects\quantumcoin\assets\quantum coin logo.png"
  if(Test-Path $src){ Copy-Item $src ".\assets\logo.png" -Force }
}

function Find-LastJson {
  if (Test-Path ".\out\last.json") { return ".\out\last.json" }
  elseif (Test-Path ".\last.json") { return ".\last.json" }
  else { return "" }
}

function New-Topic {
  node --env-file=.env .\scripts\gen-content.mjs | Out-Null
  $lj = Find-LastJson
  if(-not $lj){ return "" }
  try { return ((Get-Content $lj -Raw | ConvertFrom-Json).text).Trim() } catch { return "" }
}

try {
  # [1] Telegram + X için konu üret
  Write-Host "[1/4] Generating topic for Telegram/X..."
  $msg = New-Topic
  if([string]::IsNullOrWhiteSpace($msg)){
    $msg = "QuantumCoin update: AI-augmented chain, real-time mining and Wallet+Miner. Follow: https://github.com/QuantumCoinNews/quantumcoin"
  }

  # [2] Telegram + X paylaş — publish-all tek string bekliyorsa teker teker çağır
  Write-Host "[2/4] Posting to Telegram & X...
& .\publish-all.ps1 -Text $msg -Platform telegram
$xMsg = "$msg — $(Get-Date -Format 'HHmmss')"
& .\publish-all.ps1 -Text $xMsg -Platform x
[3/4] Generating NEW topic for YouTube... Generating NEW topic for YouTube..."
  $yt = ""
  for($i=0; $i -lt 3 -and ([string]::IsNullOrWhiteSpace($yt) -or $yt -eq $msg); $i++){
    Start-Sleep -Milliseconds 300
    $yt = New-Topic
  }
  if([string]::IsNullOrWhiteSpace($yt)){ $yt = $msg }  # son çare

  # Görsel kontrol: logo/bg
  if(!(Test-Path ".\assets\logo.png")){ Write-Warning "assets\\logo.png yok — logo eklenmeyecek." }
  if(!(Test-Path ".\assets\bg.mp4") -and !(Test-Path ".\assets\bg.jpg")){ Write-Warning "assets\\bg.mp4/bg.jpg yok — düz arka plan olabilir." }

  # [4] Shorts üret + YouTube yükle (logo/bg varsa otomatik)
  Write-Host "[4/4] Making Shorts & uploading to YouTube..."
  node --env-file=.env .\scripts\make-shorts.mjs --privacy public

  # (opsiyonel) YT linkini Telegram/X'e duyur
  $announceLine = (Get-Content .\.env | ? {$_ -match '^ENABLE_ANNOUNCE='}) -replace '.*=',''
  if($announceLine -eq '1' -and (Test-Path ".\out\last_youtube.json")){
    Write-Host "[Announce] Sharing YouTube link to Telegram & X..."
    node --env-file=.env .\scripts\announce.mjs
  }

  Write-Host "DONE."
}
finally{
  if(Test-Path $lock){ Remove-Item $lock -Force }
  Stop-Transcript | Out-Null
}


