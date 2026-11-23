

function Write-QcInfo {
    param(
        [Parameter(Position = 0, ValueFromRemainingArguments = $true)]
        [object[]] $Object,
        [switch] $NoNewline,
        [object] $Separator = " ",
        [ConsoleColor] $ForegroundColor,
        [ConsoleColor] $BackgroundColor
    )

    $null = $ForegroundColor
    $null = $BackgroundColor


    $null = $NoNewline

    $text = ($Object | ForEach-Object { [string] $_ }) -join [string] $Separator

    Write-Output $text
}
$null = "Stop"

# Bu script assets\cinematics içinden çalışır.
# PSScriptRoot = ...\dev_ui\assets\cinematics
# Base assets klasörü olmalı:
$base = Split-Path -Parent $PSScriptRoot

$files = @(
  "videos\intro\intro_logo.mp4",
  "videos\intro\intro_ship.mp4",
  "videos\intro\intro_cockpit.mp4",
  "videos\intro\intro_moon_overview.mp4",
  "videos\zones\zone_01.mp4",
  "videos\zones\zone_02.mp4",
  "videos\zones\zone_03.mp4",
  "videos\zones\zone_04.mp4",
  "videos\zones\zone_05.mp4",
  "audio\intro_logo.wav",
  "audio\space_ambience.mp3",
  "audio\cockpit_radio.mp3",
  "audio\zone_transition.wav",
  "cinematics\cinematic_manifest.json",
  "cinematics\README_CINEMATICS.md"
)

Write-QcInfo "`n===== Cinematic Asset Check =====" -ForegroundColor Cyan
Write-QcInfo "Base: $base" -ForegroundColor DarkGray

foreach ($relative in $files) {
  $path = Join-Path -Path $base -ChildPath $relative

  if (!(Test-Path $path)) {
    Write-QcInfo "MISSING: $relative" -ForegroundColor Red
    continue
  }

  $item = Get-Item $path

  if ($item.Length -eq 0 -and ($item.Extension -in ".mp4", ".mp3", ".wav")) {
    Write-QcInfo "PLACEHOLDER: $relative is empty" -ForegroundColor Yellow
  } else {
    Write-QcInfo "OK: $relative ($($item.Length) bytes)" -ForegroundColor Green
  }
}
