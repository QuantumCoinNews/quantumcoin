# scripts/run-slot.ps1
param(
  [Parameter(Mandatory=$true)]
  [ValidateSet(1,2)]
  [int]$Slot
)

$ErrorActionPreference = "Stop"

$ROOT = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $ROOT

# Date key (local)
$today = (Get-Date).ToString("yyyy-MM-dd")

$stateDir = Join-Path $ROOT "state"
$lockDir  = Join-Path $stateDir "locks"
New-Item -ItemType Directory -Force -Path $lockDir | Out-Null

$lockPath = Join-Path $lockDir ("{0}_slot{1}.json" -f $today, $Slot)

if (Test-Path $lockPath) {
  Write-Host "[run-slot] SKIP: already ran today slot=$Slot ($lockPath)"
  exit 0
}

# Pre-create lock to avoid double-run race
$lockObj = [ordered]@{
  date = $today
  slot = $Slot
  startedAt = (Get-Date).ToString("o")
  status = "started"
}
$lockObj | ConvertTo-Json -Depth 4 | Set-Content -Encoding UTF8 $lockPath

try {
  # Call Node script with slot context (we'll use it for deterministic topic)
  $env:QC_SLOT = "$Slot"
  $env:QC_TODAY = "$today"

  # Optional: disable subtitles to avoid screen-cover text
  if (-not $env:DISABLE_SUBTITLES) { $env:DISABLE_SUBTITLES = "1" }

  # Run make-shorts (it already uploads YouTube and can announce to TG/X if ENABLE_ANNOUNCE=1)
  node --env-file=.env .\scripts\make-shorts.mjs --privacy public

  # Mark success
  $lockObj.status = "done"
  $lockObj.finishedAt = (Get-Date).ToString("o")
  $lockObj | ConvertTo-Json -Depth 4 | Set-Content -Encoding UTF8 $lockPath

  Write-Host "[run-slot] DONE slot=$Slot"
  exit 0
}
catch {
  $lockObj.status = "failed"
  $lockObj.error = $_.Exception.Message
  $lockObj.finishedAt = (Get-Date).ToString("o")
  $lockObj | ConvertTo-Json -Depth 4 | Set-Content -Encoding UTF8 $lockPath
  Write-Error "[run-slot] FAILED slot=$Slot : $($_.Exception.Message)"
  exit 1
}
