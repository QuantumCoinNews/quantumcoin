param(
  [Parameter(Position=0)]
  [ValidateSet('1','2')]
  [string]$Slot = '1'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# Script hangi klasördeyse root orası (her PC’de çalışır)
$root = $PSScriptRoot
Set-Location $root


# --- FIX: Always run from repo root (stable paths) ---
$REPO = $root
$TMP  = Join-Path $REPO "tmp"
$OUT  = Join-Path $REPO "src\content\out"
$LAST = Join-Path $OUT  "last.json"

New-Item -ItemType Directory -Force -Path $TMP | Out-Null
New-Item -ItemType Directory -Force -Path $OUT | Out-Null
# -----------------------------------------------------

# =====================================================
# LOGGING (hem ekrana hem dosyaya) — 0-byte asla olmasın
# =====================================================
$logDir = Join-Path $root 'logs'
New-Item -ItemType Directory -Force -Path $logDir | Out-Null
$log = Join-Path $logDir ("sched-" + (Get-Date -Format "yyyyMMdd-HHmmss") + "-slot$Slot.log")

function Write-Log([string]$m){
  $line = "[{0}] {1}" -f (Get-Date -Format "HH:mm:ss"), $m
  $line | Out-File -FilePath $log -Append -Encoding UTF8
  Write-Host $line
}

Write-Log "START slot=$Slot user=$env:USERNAME computer=$env:COMPUTERNAME pwd=$(Get-Location)"
Write-Log "log=$log"

# =====================================================
# LOCK (slot bazlı) — scheduler aynı anda 2 kere çalışmasın
# =====================================================
$tmpDir = Join-Path $root 'tmp'
New-Item -ItemType Directory -Force -Path $tmpDir | Out-Null

function Enter-Lock([string]$slot){
  $lockPath = Join-Path $tmpDir ("slot{0}.lock" -f $slot)

  if (Test-Path $lockPath){
    $ageMin = [int]((Get-Date) - (Get-Item $lockPath).LastWriteTime).TotalMinutes

    # lock içinden pid okumaya çalış
    $lockPid = $null
    try {
      $raw = (Get-Content $lockPath -Raw -ErrorAction Stop).Trim()
      if ($raw -match 'pid=(\d+)'){ $lockPid = [int]$Matches[1] }
    } catch {}

    $alive = $false
    if ($lockPid){
      try { $null = Get-Process -Id $lockPid -ErrorAction Stop; $alive = $true } catch { $alive = $false }
    }

    if ($alive -and $ageMin -lt 180){
      Write-Log "lock exists ($lockPath) age=${ageMin}m pid=$lockPid -> skipping"
      return $null
    }

    Write-Log "lock stale ($lockPath) age=${ageMin}m pid=$lockPid alive=$alive -> removing"
    Remove-Item $lockPath -Force -ErrorAction SilentlyContinue
  }

  "pid=$PID`nstart=$(Get-Date -Format o)" | Out-File -FilePath $lockPath -Encoding UTF8 -Force
  Write-Log "lock acquired ($lockPath) pid=$PID"
  return $lockPath
}

function Exit-Lock([string]$lockPath){
  if ($lockPath -and (Test-Path $lockPath)){
    Remove-Item $lockPath -Force -ErrorAction SilentlyContinue
    Write-Log "lock released"
  }
}

# Eski isimlerle çağrı varsa bozulmasın
Set-Alias Acquire-Lock Enter-Lock -Scope Script -Force
Set-Alias Release-Lock Exit-Lock  -Scope Script -Force

# =====================================================
# .env load (sağlam parser) — Split(…';'…) gibi kırılmaz
# =====================================================
function Import-DotEnv {
  param([string]$Path = ".\.env")
  if (-not (Test-Path $Path)) { throw ".env not found in $root" }

  foreach ($raw in Get-Content $Path){
    $line = ($raw ?? "").Trim()
    if ($line.Length -eq 0) { continue }
    if ($line.StartsWith("#")) { continue }

    # "KEY=VALUE" (ilk '=' üzerinden)
    $eq = $line.IndexOf("=")
    if ($eq -lt 1) { continue }

    $k = $line.Substring(0, $eq).Trim()
    $v = $line.Substring($eq + 1).Trim()

    # tırnakları kırp
    if ($v.Length -ge 2 -and (
        ($v.StartsWith('"') -and $v.EndsWith('"')) -or
        ($v.StartsWith("'") -and $v.EndsWith("'"))
    )){
      $v = $v.Substring(1, $v.Length - 2)
    }

    if ($k.Length -gt 0){
      [Environment]::SetEnvironmentVariable($k, $v, "Process")
    }
  }
}
Set-Alias Load-DotEnv Import-DotEnv -Scope Script -Force

# =====================================================
# make-shorts output parse (title + youtube link)
# =====================================================
function ConvertFrom-MakeShortsOutput {
  param([string[]]$Lines)

  $title = $null
  $yt    = $null

  foreach ($l in $Lines){
    $s = ($l ?? "").ToString()

    if (-not $title -and $s -match '^\[make-shorts\]\s*Title:\s*(.+)$'){
      $title = $Matches[1].Trim()
      continue
    }
    if (-not $yt -and $s -match 'YouTube:\s*(https?://\S+)'){
      $yt = $Matches[1].Trim()
      continue
    }
    if (-not $yt -and $s -match 'youtube\s*=\s*(https?://\S+)'){
      $yt = $Matches[1].Trim()
      continue
    }
  }

  return [pscustomobject]@{
    title = $title
    youtube = $yt
  }
}
Set-Alias Parse-MakeShortsOutput ConvertFrom-MakeShortsOutput -Scope Script -Force

# =====================================================
# last.json safe read (JSON değilse parse etmeye çalışma)
# =====================================================
function Get-LastJsonSafely {
  param([string]$Path)

  if (-not (Test-Path $Path)) { return $null }
  $raw = (Get-Content $Path -Raw -ErrorAction SilentlyContinue)
  if (-not $raw) { return $null }

  $t = $raw.TrimStart()
  if (-not ($t.StartsWith("{") -or $t.StartsWith("["))) { return $null }

  try { return ($raw | ConvertFrom-Json) } catch { return $null }
}
Set-Alias Try-ReadLastJson Get-LastJsonSafely -Scope Script -Force

# =====================================================
# yardımcı: platform listesi
# =====================================================
function Get-Platforms {
  $p = $env:QC_PLATFORMS
  if ($env:QC_FORCE_ALL -eq '1' -or [string]::IsNullOrWhiteSpace($p)){
    $p = 'youtube,telegram,x'
  }
  return ($p -split ',' | ForEach-Object { $_.Trim().ToLowerInvariant() } | Where-Object { $_ })
}

# =====================================================
# POOL ROTATOR (bgs + audio) — her çalışmada bir sonraki sesi/görseli seç
#   - audio: assets\pool\audio\ (voice_01..voice_30)
#   - bgs:   assets\pool\bgs\   (bg_01..bg_07)
#   - state: assets\pool\state_slot{Slot}.json   { "next": 1 }
# =====================================================
$poolDir      = Join-Path $root 'assets\pool'
$poolAudioDir = Join-Path $poolDir 'audio'
$poolBgsDir   = Join-Path $poolDir 'bgs'
$poolState    = Join-Path $poolDir ("state_slot{0}.json" -f $Slot)

function Get-Num([string]$name){
  $m = [regex]::Match($name, '\d+')
  if($m.Success){ return [int]$m.Value }
  return 0
}

function Get-PoolList([string]$dir, [string[]]$exts){
  if(!(Test-Path $dir)){ return @() }
  Get-ChildItem -Path $dir -File -ErrorAction SilentlyContinue |
    Where-Object { $exts -contains $_.Extension.ToLower() } |
    Sort-Object @{Expression={ Get-Num $_.Name }}, Name |
    Select-Object -ExpandProperty FullName
}

function Get-PoolIndex([string]$statePath, [int]$fallback=1){
  try {
    if(Test-Path $statePath){
      $j = Get-Content $statePath -Raw | ConvertFrom-Json
      $val = $j.next
      if($null -ne $val){
        $i = $val -as [int]
        if($i -ge 1){ return [int]$i }
      }
    }
  } catch {}
  return $fallback
}

function Set-PoolIndex([string]$statePath, [int]$next){
  @{ next = $next } | ConvertTo-Json -Compress | Out-File -FilePath $statePath -Encoding UTF8 -Force
}

function New-Dir([string]$p){ New-Item -ItemType Directory -Force -Path $p | Out-Null }

function Set-PoolAssets([string]$slot){
 New-Dir (Join-Path $root 'tmp')
New-Dir $poolDir


  # state yoksa otomatik oluştur
  if(!(Test-Path $poolState)){
    Set-PoolIndex $poolState 1
  }

  $audios = Get-PoolList $poolAudioDir @('.mp3','.wav')
  $bgs    = Get-PoolList $poolBgsDir   @('.mp4','.png','.jpg','.jpeg')
    # BG döngüsü daima 7 olsun (fazla dosya varsa ilk 7’yi kullan)
  if ($bgs.Count -gt 7) { $bgs = $bgs | Select-Object -First 7 }

  if($audios.Count -lt 1){ Write-Log "POOL: no audio files in $poolAudioDir -> skipping"; return $null }
  if($bgs.Count    -lt 1){ Write-Log "POOL: no bg files in $poolBgsDir -> skipping"; return $null }

  $idx = Get-PoolIndex $poolState 1

  # idx=1..30; bg sayısı 7 ise otomatik 1..7 döner
  $audioPath = $audios[ ($idx-1) % $audios.Count ]
  $bgPath    = $bgs[    ($idx-7) % $bgs.Count    ]

  $wavOut = Join-Path $root 'tmp\tts.wav'      # make-shorts bunu kullanıyor
  $bgOut  = Join-Path $root 'assets\bg.mp4'    # make-shorts arka planı buradan alıyor

  Write-Log "POOL: idx=$idx audio=$(Split-Path $audioPath -Leaf)"
  if($audioPath.ToLower().EndsWith('.wav')){
    Copy-Item $audioPath $wavOut -Force
  } else {
    & ffmpeg -y -i $audioPath -ar 22050 -ac 1 -c:a pcm_s16le $wavOut | Out-Null
  }

  Write-Log "POOL: idx=$idx bg=$(Split-Path $bgPath -Leaf)"
  if($bgPath.ToLower().EndsWith('.mp4')){
    Copy-Item $bgPath $bgOut -Force
  } else {
    & ffmpeg -y -loop 1 -t 16 -i $bgPath `
      -vf "scale=1080:1920:force_original_aspect_ratio=increase,crop=1080:1920,setsar=1,zoompan=z='min(zoom+0.0008,1.08)':d=1:s=1080x1920:fps=30" `
      -r 30 -an -c:v libx264 -pix_fmt yuv420p $bgOut | Out-Null
  }

  $next = ([int]$idx) + 1
  if($next -gt 30){ $next = 1 }

  return @{ idx=$idx; next=$next; state=$poolState }
}

# =====================================================
# MAIN
# =====================================================
$lockPath = $null
try {
  $lockPath = Enter-Lock $Slot
  if (-not $lockPath) { exit 0 }

  # Node PATH (Task Scheduler PATH boş olabiliyor)
  $nodeDir = 'C:\Program Files\nodejs'
  if (Test-Path $nodeDir){ $env:Path = "$nodeDir;$env:Path" }

  $nodeV = (& node -v 2>&1 | Out-String).Trim()
  Write-Log "node=$nodeV"

  Import-DotEnv ".\.env"
  Write-Log "ENV loaded from .env"

  # .env içinden 1 gelse bile burada kesin kapat (double post olmasın)
  $env:ENABLE_ANNOUNCE = "0"

  $platforms = Get-Platforms
  Write-Log ("platforms=" + ($platforms -join ','))

  # --- publish zamanı: slot saati geçmişse yarına at (YouTube publishAt geçmişte kalmasın)
  $slotHour = if ($Slot -eq '1') { 18 } else { 21 }
  $now = Get-Date
  $target = Get-Date -Hour $slotHour -Minute 0 -Second 0
  if ($now -ge $target) { $target = $target.AddDays(1) }

  $env:QC_TODAY = $target.ToString('yyyy-MM-dd')
  $env:QC_SLOT  = $Slot
  if (-not $env:QC_TZ_OFFSET) { $env:QC_TZ_OFFSET = '+03:00' }

  $env:DISABLE_SUBTITLES = '1'
  $env:YOUTUBE_SCHEDULE  = '1'

  Write-Log "vars: QC_TODAY=$env:QC_TODAY QC_SLOT=$env:QC_SLOT QC_TZ_OFFSET=$env:QC_TZ_OFFSET DISABLE_SUBTITLES=$env:DISABLE_SUBTITLES YOUTUBE_SCHEDULE=$env:YOUTUBE_SCHEDULE"

# =====================================================
# 1) YouTube (make-shorts)
# =====================================================
$msLines = @()
$meta = [pscustomobject]@{ title=$null; youtube=$null }
$pool = $null

if ($platforms -contains 'youtube') {

  # pool -> tmp\tts.wav + assets\bg.mp4 hazırlanır
  try {
    $pool = Set-PoolAssets $Slot
  } catch {
    $pool = $null
    Write-Log "WARN: Set-PoolAssets failed: $($_.Exception.Message)"
  }

  if ($pool) {
    Write-Log "POOL ready (idx=$($pool.idx) -> next=$($pool.next))"

    # make-shorts pool ile aynı index/metayı kullanabilsin
    $env:POOL_MODE = "1"
    $env:POOL_IDX  = "$($pool.idx)"
    # Not: USE_POOL_TTS dışarıdan set ediliyor olabilir; pool varken garantiye alalım
    if (-not $env:USE_POOL_TTS) { $env:USE_POOL_TTS = "1" }

  } else {
    # pool yoksa önceki koşudan kalan env karışmasın
    Remove-Item Env:POOL_MODE -ErrorAction SilentlyContinue
    Remove-Item Env:POOL_IDX  -ErrorAction SilentlyContinue
  }

  Write-Log "make-shorts start"
  $msLines = @(& node .\scripts\make-shorts.mjs 2>&1)
  $msExit  = $LASTEXITCODE

  foreach ($l in $msLines) { Write-Log $l }

  if ($msExit -ne 0) {
    $joined = ($msLines -join "`n")
    if ($joined -match 'exceeded the number of videos they may upload') {
      Write-Log "WARN: YouTube upload limit hit today. Continuing with Telegram/X."
    } else {
      throw "make-shorts FAILED exit=$msExit"
    }
  }
  else {
    Write-Log "make-shorts OK"

    # pool index sadece başarılı render/upload sonrası ilerlesin
    if ($pool) {
      Set-PoolIndex $pool.state $pool.next
      Write-Log "POOL: advanced -> next=$($pool.next)"
    }
  }

  $meta = ConvertFrom-MakeShortsOutput -Lines $msLines
  if ($meta.title)  { Write-Log "parsed title=$($meta.title)" }
  if ($meta.youtube){ Write-Log "parsed youtube=$($meta.youtube)" }
}


   # =====================================================
  # 2) last.json BUILD (SINGLE SOURCE OF TRUTH) + fresh TG/X
  #   - NO hardcoded paths
  #   - Telegram always shows latest title+link
  #   - X includes rolling history bullets and stays <= 280
  # =====================================================

  $lastJsonPath = $LAST
  New-Item -ItemType Directory -Force -Path (Split-Path $lastJsonPath) | Out-Null

  # read previous last.json (for X history)
  $prev = Get-LastJsonSafely $lastJsonPath

  # extract videoId
  $videoId = $null
  if ($meta.youtube -and $meta.youtube -match 'v=([A-Za-z0-9_\-]+)'){ $videoId = $Matches[1] }

  # Telegram: ALWAYS fresh
  if ($meta.youtube) {
    $tgText = "New YouTube Short: $($meta.title)`n$($meta.youtube)`n@QuantumCoinHQ"
  } else {
    $tgText = "QuantumCoin daily update (slot $Slot) — @QuantumCoinHQ"
  }

  # X base: ALWAYS fresh
  if ($meta.youtube) {
    $xBase = "$($meta.title) — $($meta.youtube) #QuantumCoin #crypto #shorts"
  } else {
    $xBase = "QuantumCoin daily update (slot $Slot) — YouTube @QuantumCoinHQ #crypto #shorts"
  }

  # Build history bullets from previous x field
  $hist = @()
  if ($prev -and $prev.x) {
    $hist = @(
      ($prev.x -split '•' | ForEach-Object { $_.Trim() } |
        Where-Object { $_ -match '^\d{4}-\d{2}-\d{2}\s+S\d+' } |
        ForEach-Object { "• $_" })
    )
  }

  # current bullet
  $day = $env:QC_TODAY
  $slotStr = "$Slot"
  $newBullet = if ($videoId) { "• $day S$slotStr $videoId" } else { "• $day S$slotStr" }

  if ($newBullet -and -not ($hist -contains $newBullet)) { $hist += $newBullet }

  # keep only latest N bullets to avoid length issues
  $maxBullets = 8
  if ($hist.Count -gt $maxBullets) {
    $hist = $hist[-$maxBullets..-1]
  }

  # compose X with history
  $xText = $xBase
  if ($hist.Count -gt 0) { $xText = ($xText + " " + ($hist -join " ")).Trim() }

  # enforce 280 chars by dropping oldest bullets first
  while ($xText.Length -gt 280 -and $hist.Count -gt 0) {
    $hist = $hist[1..($hist.Count-1)]  # drop oldest
    $xText = $xBase
    if ($hist.Count -gt 0) { $xText = ($xText + " " + ($hist -join " ")).Trim() }
  }

  # still too long? hard-trim end (rare)
  if ($xText.Length -gt 280) {
    $xText = $xText.Substring(0, 280)
  }

  # write last.json once
  $outObj = [ordered]@{
    title       = $meta.title
    youtube     = $meta.youtube
    telegram    = $tgText
    x           = $xText
    slot        = "$Slot"
    day         = $env:QC_TODAY
    tz          = $env:QC_TZ_OFFSET
    generatedAt = (Get-Date).ToString("o")
  }

  ($outObj | ConvertTo-Json -Depth 8) | Set-Content -Path $lastJsonPath -Encoding UTF8
  Write-Log "last.json written OK ($lastJsonPath)"

# ---- end ----

  # =====================================================
  # 3) Telegram + X publish
  # =====================================================
  if ($platforms -contains 'telegram'){
    Write-Log "telegram announce start"
    $tgOut = @(& pwsh -NoProfile -ExecutionPolicy Bypass -File .\publish-all.ps1 -Text $tgText -Platforms telegram 2>&1)
    $tgExit = $LASTEXITCODE
    foreach($l in $tgOut){ Write-Log $l }
    Write-Log "telegram exit=$tgExit"
    Start-Sleep -Seconds 2
  }

  if ($platforms -contains 'x'){
    Write-Log "x announce start"
    $xOut = @(& pwsh -NoProfile -ExecutionPolicy Bypass -File .\publish-all.ps1 -Text $xText -Platforms x 2>&1)
    $xExit = $LASTEXITCODE
    foreach($l in $xOut){ Write-Log $l }
    Write-Log "x exit=$xExit"

    # duplicate ise 1 kez retry (sonuna HHmmss ekle)
    if ($xExit -ne 0 -and (($xOut -join "`n") -match 'duplicate')){
      Start-Sleep -Seconds 2

      $tag = (Get-Date -Format "HHmmss")
      $xRetry = ($xText + " " + $tag).Trim()
      if ($xRetry.Length -gt 280) { $xRetry = $xRetry.Substring(0, 280) }

      Write-Log "x retry start"
      $xrOut = @(& pwsh -NoProfile -ExecutionPolicy Bypass -File .\publish-all.ps1 -Text $xRetry -Platforms x 2>&1)
      $xrExit = $LASTEXITCODE
      foreach($l in $xrOut){ Write-Log $l }
      Write-Log "x retry exit=$xrExit"
    }
  }

  Write-Log "DONE OK"
  exit 0
}
catch {
  Write-Log ("FATAL: " + $_.Exception.Message)
  if ($_.ScriptStackTrace){ Write-Log ("STACK: " + ($_.ScriptStackTrace -replace "`r?`n"," | ")) }
  exit 2
}
finally {
  Exit-Lock $lockPath
}
