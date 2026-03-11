param(
  [string]$Voice = "en-US-GuyNeural"
)

$root = Split-Path -Parent $PSScriptRoot
$topicsPath = Join-Path $root "src\content\pool\topics_pool.json"
$outDir = Join-Path $root "assets\pool\voices"
New-Item -ItemType Directory -Force -Path $outDir | Out-Null

$topics = Get-Content $topicsPath -Raw | ConvertFrom-Json

$i = 1
foreach ($t in $topics) {
  $num = "{0:D3}" -f $i
  $out = Join-Path $outDir ("voice_{0}.mp3" -f $num)

  # 15-20 saniye için normal hız, istersen rate ile oynarız.
  $text = $t.script.Replace("`r"," ").Replace("`n"," ").Trim()

  Write-Host "[$num] -> $out"
  edge-tts --voice $Voice --text $text --write-media $out --format "audio-24khz-48kbitrate-mono-mp3"
  $i++
}
