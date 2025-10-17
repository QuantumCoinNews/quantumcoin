param(
  [Parameter(Mandatory=$true)][string]$Text,
  [Parameter(Mandatory=$true)][string]$Platforms  # "telegram,x"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

# ========== LOG ==========
$logDir = Join-Path $PSScriptRoot 'logs'
New-Item -ItemType Directory -Force -Path $logDir | Out-Null
$script:__logPath = Join-Path $logDir ("pub-" + (Get-Date -Format "yyyyMMdd-HHmmss") + ".log")
function Write-Log([string]$m) {
  $line = "[{0}] {1}" -f (Get-Date -Format "HH:mm:ss"), $m
  $line | Tee-Object -FilePath $script:__logPath -Append
}

# Güvenli hata metni
function Get-Err([object]$e) {
  try {
    if ($null -eq $e) { return "unknown error" }
    if ($e.PSObject.Properties.Match("ErrorDetails").Count -and
        $e.ErrorDetails -and
        ($e.ErrorDetails | Get-Member -Name Message -MemberType * -ErrorAction SilentlyContinue)) {
      if ($e.ErrorDetails.Message) { return $e.ErrorDetails.Message }
    }
    if ($e.PSObject.Properties.Match("Exception").Count -and $e.Exception -and $e.Exception.Message) {
      return $e.Exception.Message
    }
    return ($e | Out-String)
  } catch { return ($e | Out-String) }
}

# ========== .env loader ==========
function Load-DotEnv([string]$Path) {
  if (-not (Test-Path -LiteralPath $Path)) { return }
  Get-Content -LiteralPath $Path | ForEach-Object {
    $l=$_.Trim(); if($l -eq '' -or $l -match '^\s*#'){return}
    $p=$l -split '=',2; if($p.Count -ne 2){return}
    [Environment]::SetEnvironmentVariable($p[0].Trim(), $p[1].Trim().Trim('"',"'"), 'Process')
  }
  Write-Log "ENV loaded"
}
Load-DotEnv (Join-Path $PSScriptRoot ".env")

# Platform seçimi
$useTelegram = $Platforms -match '(^|,)telegram($|,)'
$useX        = $Platforms -match '(^|,)x($|,)'

$overallOk = $true

# ---------- TELEGRAM ----------
if ($useTelegram) {
  $token  = $env:TELEGRAM_BOT_TOKEN
  $chatId = $env:TELEGRAM_CHAT_ID
  $parse  = if ($env:TELEGRAM_PARSE_MODE) { $env:TELEGRAM_PARSE_MODE } else { 'HTML' }
  $preview= if (($env:TELEGRAM_DISABLE_PREVIEW -as [int]) -eq 1) { 'true' } else { 'false' }
  if (-not $token -or -not $chatId) {
    Write-Log "Telegram SKIP (env missing)"; $overallOk = $false
  } else {
    try {
      $resp = Invoke-RestMethod -Method Post `
        -Uri "https://api.telegram.org/bot$token/sendMessage" `
        -Body @{ chat_id = $chatId; text = $Text; parse_mode = $parse; disable_web_page_preview = $preview }
      Write-Log ("Telegram OK -> message_id=" + $resp.result.message_id)
    } catch { Write-Log ("Telegram ERR -> " + (Get-Err $_)); $overallOk = $false }
  }
}

# ---------- X (Twitter) ----------
if ($useX) {
  try {
    $nodeArgs = @('--env-file=.env', '.\scripts\post-x.mjs', '--text', $Text)
    $out = & node @nodeArgs 2>&1
    if ($LASTEXITCODE -eq 0 -and $out) {
      Write-Log ("X OK -> " + ($out -replace "`r"," " -replace "`n"," ").Trim())
    } else {
      Write-Log ("X ERR -> " + ($out -replace "`r"," " -replace "`n"," ").Trim()); $overallOk = $false
    }
  } catch { Write-Log ("X ERR -> " + (Get-Err $_)); $overallOk = $false }
}

Write-Log "Bitti -> done"
if (-not $overallOk) { exit 2 } else { exit 0 }
