param(
  [int]$TargetSec = 180,   # 2-3 dk hedef (bilgi amaçlı)
  [int]$MinWords  = 450    # ~2-3 dk için kelime adedi (hızı -1)
)

$ErrorActionPreference = "Stop"
$base = Split-Path -Parent $PSCommandPath
Set-Location $base
if(-not (Test-Path ".\tmp")){ New-Item -ItemType Directory -Path ".\tmp" | Out-Null }

function Get-LastText {
  # içerik üret
  node --env-file=.env .\scripts\gen-content.mjs | Out-Null

  # muhtemel json yollarını sırayla dene
  $cands = @("$base\out\last.json","$base\last.json") + (
    Get-ChildItem . -Recurse -Filter last.json -ErrorAction SilentlyContinue |
    Sort-Object LastWriteTime -Desc | Select-Object -Expand FullName
  )

  foreach($p in $cands | Get-Unique){
    if(Test-Path $p){
      try{
        $obj = Get-Content $p -Raw | ConvertFrom-Json
        if($obj -and $obj.PSObject.Properties['text']){
          $txt = [string]$obj.text
          if(-not [string]::IsNullOrWhiteSpace($txt)){ return $txt }
        }
      } catch { }
    }
  }

  # txt fallback
  foreach($p in @('.\out\last.txt','.\last.txt')){
    if(Test-Path $p){
      $txt = (Get-Content $p -Raw).Trim()
      if($txt){ return $txt }
    }
  }
  return ''
}

# 1) Uzun metni blok blok topla
$blocks=@(); $wc=0; $maxIter=20
for($i=0; $i -lt $maxIter -and $wc -lt $MinWords; $i++){
  $t = (Get-LastText).Trim()
  if([string]::IsNullOrWhiteSpace($t)){ continue }
  $blocks += $t.TrimEnd('.','!','?')
  $wc += ($t -split '\s+').Count
}

if($blocks.Count -eq 0){
  $blocks = @("QuantumCoin long-form update: roadmap, AI-augmented chain design, miner progress, and ecosystem outlook")
}

$full = (($blocks -join '. ') + '.').Replace('  ',' ')
Set-Content .\tmp\script.txt $full -Encoding UTF8

# 2) Windows TTS ile WAV (TR varsa seç)
Add-Type -AssemblyName System.Speech
$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer
try{
  $tr = $synth.GetInstalledVoices() | Where-Object {$_.VoiceInfo.Culture.Name -like 'tr-*'} | Select-Object -First 1
  if($tr){ $synth.SelectVoice($tr.VoiceInfo.Name) }
}catch{}
$synth.Volume = 100
$synth.Rate   = -1          # daha yavaş/uzun: -2, daha hızlı: 0..2
$synth.SetOutputToWaveFile(".\tmp\tts.wav")
$synth.Speak([string]$full)
$synth.Dispose()

# 3) Render + upload (online TTS'e dokunma)
$env:OFFLINE_SHORTS = "1"
node --env-file=.env .\scripts\make-shorts.mjs --privacy public
Remove-Item Env:OFFLINE_SHORTS -ErrorAction SilentlyContinue
