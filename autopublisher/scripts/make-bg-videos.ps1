$root = Split-Path -Parent $PSScriptRoot
$inDir = Join-Path $root "assets\pool\bgs"
$outDir = Join-Path $root "assets\pool\bgs"
New-Item -ItemType Directory -Force -Path $inDir | Out-Null

for ($i=1; $i -le 7; $i++){
  $num = "{0:D2}" -f $i
  $png = Join-Path $inDir ("bg_{0}.png" -f $num)
  $mp4 = Join-Path $outDir ("bg_{0}.mp4" -f $num)

  if (Test-Path $png){
    ffmpeg -y -loop 1 -t 19 -i $png -vf "scale=1080:1920:force_original_aspect_ratio=increase,crop=1080:1920,setsar=1" -r 30 -pix_fmt yuv420p $mp4
  }
}
