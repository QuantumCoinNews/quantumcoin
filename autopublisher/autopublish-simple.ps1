# ==== LOG & ORTAM ====
$ErrorActionPreference = 'Stop'

# Log klasörü
$LogDir = Join-Path $PSScriptRoot "logs"
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null
$Log = Join-Path $LogDir ("runlog-{0}.log" -f (Get-Date -Format 'yyyyMMdd-HHmmss'))

Start-Transcript -Path $Log -Append | Out-Null
"[$(Get-Date -Format 'HH:mm:ss')] ENV loaded under user: $([System.Security.Principal.WindowsIdentity]::GetCurrent().Name)"

# SYSTEM için PATH takviyesi (Varsayılan kurulum yollarını ekledim; sende farklıysa düzelt)
$pathsToAdd = @(
  "C:\Program Files\nodejs",
  "C:\ffmpeg\bin",
  "C:\Program Files\Git\bin"
)
foreach($p in $pathsToAdd){
  if((Test-Path $p) -and ($env:Path -split ';') -notcontains $p){
    $env:Path = "$p;$env:Path"
  }
}

# Sürüm bilgisi (sorun ayıklama için)
try { "node:  $(node -v)" } catch { "node: NOT FOUND" }
try { "ffmpeg: $((ffmpeg -version | Select-String 'ffmpeg version').Line)" } catch { "ffmpeg: NOT FOUND" }

# Çalışma klasörü güvence
Set-Location $PSScriptRoot
"PWD:   $((Get-Location).Path)"
"Paths: $env:Path"
"==== SCRIPT START ===="


$ErrorActionPreference = "Stop"
$base = "C:\Projects\quantumcoin\autopublisher"
Set-Location $base

# Log + tek-instance lock
if(-not (Test-Path ".\logs")){ New-Item -ItemType Directory -Path ".\logs" | Out-Null }
$ts = Get-Date -Format "yyyy-MM-dd_HHmmss"
Start-Transcript -Path ".\logs\$ts.log" | Out-Null
$lock = ".\autopublisher.lock"
if(Test-Path $lock){ Write-Host "Another run is in progress. Exiting."; Stop-Transcript | Out-Null; exit 0 }
New-Item -ItemType File -Path $lock | Out-Null

# Logo/yol güvence
if(!(Test-Path ".\assets\logo.png")){
  $src="C:\Projects\quantumcoin\assets\quantum coin logo.png"
  if(Test-Path $src){ Copy-Item $src ".\assets\logo.png" -Force }
}
if(!(Test-Path ".\assets\bg.mp4") -and !(Test-Path ".\assets\bg.jpg")){
  # sağlam 9:16 koyu zemin (19 sn)
  ffmpeg -y -f lavfi -i color=c=0x0B1220:size=1080x1920:rate=30 -t 19 ".\assets\bg.mp4"
}

function Get-Topic {
  node --env-file=.env .\scripts\gen-content.mjs | Out-Null
  $cands = @(".\out\last.json",".\last.json") + (Get-ChildItem . -Recurse -Filter last.json -EA SilentlyContinue | Sort-Object LastWriteTime -Desc | Select -First 3 | % FullName)
  foreach($p in $cands){ if(Test-Path $p){ try{ $o = Get-Content $p -Raw | ConvertFrom-Json; if($o.text){ return ($o.text.ToString().Trim()) } }catch{} } }
  return "QuantumCoin daily update: AI-augmented chain & real-time mining — https://github.com/QuantumCoinNews/quantumcoin"
}

try{
  # [1] Konu üret
  $msg = Get-Topic

  # [2] Telegram + X (X’e küçük zaman etiketi ekle ki duplicate olmasın)
  & .\publish-all.ps1 -Text $msg -Platform telegram
  $xMsg = "$msg — $(Get-Date -Format 'HHmmss')"
  & .\publish-all.ps1 -Text $xMsg -Platform x

  # [3] Kısa video (logo ortada; make-shorts.mjs’de merkez overlay patch’i zaten var)
  node --env-file=.env .\scripts\make-shorts.mjs --privacy public

  # (Opsiyonel) YouTube linkini duyur (ENABLE_ANNOUNCE=1 ise)
  $announce = ((Get-Content .\.env | ? {$_ -match '^ENABLE_ANNOUNCE='}) -replace '.*=','')
  if($announce -eq '1' -and (Test-Path ".\out\last_youtube.json")){
    node --env-file=.env .\scripts\announce.mjs
  }

  Write-Host "DONE."
}
finally{
  if(Test-Path $lock){ Remove-Item $lock -Force }
  Stop-Transcript | Out-Null
}
"==== SCRIPT END ===="
Stop-Transcript | Out-Null

