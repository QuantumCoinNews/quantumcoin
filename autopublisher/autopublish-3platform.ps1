param([bool]$Announce = $true)

$ErrorActionPreference = 'Stop'

# En stabil: script hangi klasörden çalışırsa çalışsın kendi dizinine gelir
Set-Location $PSScriptRoot

# Log + lock
if(-not (Test-Path ".\logs")){ New-Item -ItemType Directory -Path ".\logs" | Out-Null }
if(-not (Test-Path ".\out")){ New-Item -ItemType Directory -Path ".\out" | Out-Null }

$ts = Get-Date -Format "yyyy-MM-dd_HHmmss"
Start-Transcript -Path ".\logs\$ts.log" | Out-Null

$lock = ".\autopublisher.lock"

# Stale-lock kurtarma: 4 saatten eski lock varsa temizle
if(Test-Path $lock){
  $ageHours = ((Get-Date) - (Get-Item $lock).LastWriteTime).TotalHours
  if($ageHours -gt 4){
    Write-Warning "Stale lock detected ($([math]::Round($ageHours,1))h). Removing lock..."
    Remove-Item $lock -Force
  }
}

if(Test-Path $lock){
  Write-Host "Another run is in progress. Exiting."
  Stop-Transcript | Out-Null
  exit 0
}

# Lock oluştur (timestamp yaz)
Set-Content -Path $lock -Value ("started=" + (Get-Date).ToString("o")) -Encoding ascii

# Logo yoksa kökten kopyala
if(!(Test-Path ".\assets\logo.png")){
  $src = "C:\Projects\quantumcoin\assets\quantum coin logo.png"
  if(Test-Path $src){
    if(-not (Test-Path ".\assets")){ New-Item -ItemType Directory -Path ".\assets" | Out-Null }
    Copy-Item $src ".\assets\logo.png" -Force
  }
}

function Find-LastJson {
  if (Test-Path ".\src\content\out\last.json") { return ".\src\content\out\last.json" }
  elseif (Test-Path ".\out\last.json") { return ".\out\last.json" }
  elseif (Test-Path ".\last.json") { return ".\last.json" }
  else { return "" }
}

function Get-TrimmedXText([string]$s, [int]$max = 260) {
  if([string]::IsNullOrWhiteSpace($s)){ return $s }
  if($s.Length -le $max){ return $s }
  return $s.Substring(0, $max - 1) + "…"
}

function New-Content {
  try {
    node --env-file=.env .\scripts\gen-content.mjs | Out-Null
    if($LASTEXITCODE -ne 0){
      throw "gen-content failed (exit=$LASTEXITCODE). AI required."
    }
  } catch {
    # Scheduler retry yapabilsin diye hata yükselt
    Write-Error "gen-content error (AI required): $($_.Exception.Message)"
    throw
  }

  $lj = Find-LastJson
  if([string]::IsNullOrWhiteSpace($lj)){ return $null }

  try { return (Get-Content $lj -Raw | ConvertFrom-Json) }
  catch { return $null }
}


try {
  # [1] Telegram + X için içerik üret
  Write-Host "[1/4] Generating content for Telegram/X (AI required)..."
  $c = New-Content

  $tgMsg = ""
  $xMsg  = ""

  if($null -ne $c){
    $tgMsg = $c.telegram
    $xMsg  = $c.x
  }

  if([string]::IsNullOrWhiteSpace($tgMsg)){
    $tgMsg = "QuantumCoin update: AI-augmented chain, real-time mining and Wallet+Miner.`n`nhttps://github.com/QuantumCoinNews/quantumcoin`n#QuantumCoin #QC #Web3"
  }

  if([string]::IsNullOrWhiteSpace($xMsg)){
    $xMsg = "QuantumCoin update: AI-augmented chain, real-time mining and Wallet+Miner. #QuantumCoin #QC #Web3"
  }

  # [2] Telegram + X paylaş
  Write-Host "[2/4] Posting to Telegram & X..."
  & .\publish-all.ps1 -Text $tgMsg -Platform telegram

  # X'e küçük zaman damgası ekle (max 260 koru)
  $xMsg = Get-TrimmedXText "$xMsg $(Get-Date -Format 'HHmmss')" 260
  & .\publish-all.ps1 -Text $xMsg -Platform x

  # [3] YouTube için ayrı içerik üret (AI required) — (make-shorts kendi işini yapıyorsa bile bunu loglamak iyi)
  Write-Host "[3/4] Generating NEW content for YouTube (AI required)..."
  $ytc = $null
  try { $ytc = New-Content } catch { $ytc = $null }

  if($null -ne $ytc){
    $ytOut = @{
      dt = (Get-Date).ToString("o")
      title = $ytc.title
      category = $ytc.category
      text = $ytc.telegram
      x = $ytc.x
      source = $ytc.source
    } | ConvertTo-Json -Depth 6

    Set-Content -Path ".\out\yt_topic.json" -Value $ytOut -Encoding utf8
  }

  # Görsel kontrol: logo/bg
  if(!(Test-Path ".\assets\logo.png")){ Write-Warning "assets\logo.png yok — logo eklenmeyecek." }
  if(!(Test-Path ".\assets\bg.mp4") -and !(Test-Path ".\assets\bg.jpg")){ Write-Warning "assets\bg.mp4/bg.jpg yok — düz arka plan olabilir." }

  # [4] Shorts üret + YouTube yükle
  Write-Host "[4/4] Making Shorts & uploading to YouTube..."
  node --env-file=.env .\scripts\make-shorts.mjs --privacy public

  # (opsiyonel) YT linkini Telegram/X'e duyur
  $announceLine = ""
  if(Test-Path ".\.env"){
    $announceLine = (Get-Content .\.env | Where-Object { $_ -match '^ENABLE_ANNOUNCE=' } | Select-Object -First 1) -replace '.*=',''
  }

  if($Announce -and $announceLine -eq '1' -and (Test-Path ".\out\last_youtube.json")){
    Write-Host "[Announce] Sharing YouTube link to Telegram & X..."
    node --env-file=.env .\scripts\announce.mjs
  }

  Write-Host "DONE."
}
finally{
  if(Test-Path $lock){ Remove-Item $lock -Force }
  Stop-Transcript | Out-Null
}
