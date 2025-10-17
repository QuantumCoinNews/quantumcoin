Param(
  [string]$Message = "chore: safe update",
  [string]$Remote  = "origin",
  [string]$Branch  = "main",
  [switch]$NoPush
)
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Ensure-Git { if (-not (Get-Command git -ErrorAction SilentlyContinue)) { throw "git yok" } }
function Get-RepoRoot { (git rev-parse --show-toplevel) 2>$null }
function Ensure-GitIgnore([string]$Path) {
  $need = @(
    "# === secrets/logs/cache (auto) ===",
    ".env","*.env",".env.*","*.key","*.pem","*.p12",
    "*secret*","*secrets*","credentials.*","id_rsa*","id_ecdsa*","ssh_private_key*",
    "appsettings.*.Local.json","appsettings.Development.json","config.local.json",
    "wallet_priv.hex","wallet_address.txt","miner_address.txt",
    "wallet_balance.cache.json","bonus_store.json",
    "mined_balance.json","miner_out.log","api_out.log","*.log",
    ".vscode/",".idea/","*.code-workspace"
  )
  if (-not (Test-Path $Path)) { New-Item -Type File -Path $Path | Out-Null }
  $cur = Get-Content -Raw $Path
  $toAdd = @($need | Where-Object { $_ -and ($cur -notmatch [regex]::Escape($_)) })
  if ($toAdd.Count -gt 0) {
    Add-Content -Path $Path -Value "" -Encoding UTF8
    foreach ($line in $toAdd) { Add-Content -Path $Path -Value $line -Encoding UTF8 }
  }
}
function GlobToRegex([string]$g){ ([regex]::Escape($g)).Replace("\*\*",".*").Replace("\*","[^/\\]*").Replace("\?",".") }
function Untrack-IgnoredIfTracked {
  if (-not (Test-Path ".gitignore")) { return }
  $ignore = @(Get-Content -Raw ".gitignore" | Select-String -AllMatches "^[^\r\n#].+" | ForEach-Object { $_.Matches.Value.Trim() })
  if (-not $ignore -or $ignore.Count -eq 0) { return }
  $tracked = @((& git ls-files -z) -split "`0" | Where-Object { $_ })
  if (-not $tracked -or $tracked.Count -eq 0) { return }
  $regexes = @($ignore | ForEach-Object { [regex](GlobToRegex $_) })
  $toRm = @()
  foreach ($f in $tracked) {
    foreach ($rx in $regexes) { if ($rx.IsMatch($f)) { $toRm += $f; break } }
  }
  $toRm = @($toRm | Select-Object -Unique)
  if ($toRm.Count -gt 0) {
    git rm --cached -- $toRm | Out-Null
    git commit -m "chore: stop tracking ignored secrets/logs" | Out-Null
  }
}
function Get-StagedFiles { @((& git diff --cached --name-only -z) -split "`0" | Where-Object { $_ }) }
function Test-Secrets([string[]]$Files){
  if (-not $Files -or $Files.Count -eq 0) { return @() }
  $patterns = @(
    'sk-[A-Za-z0-9]{32,}','\bghp_[A-Za-z0-9]{36}\b','\bAIza[0-9A-Za-z\-_]{35}\b',
    '\bAKIA[0-9A-Z]{16}\b','(?i)aws(.{0,20})?(secret|access)[^A-Za-z0-9]?([A-Za-z0-9/+=]{40})',
    'xox[baprs]-[A-Za-z0-9-]{10,48}','\b[0-9]{7,12}:[A-Za-z0-9_-]{35}\b',
    'https://discord\.com/api/webhooks/[0-9]+/[A-Za-z0-9_-]+',
    '-----BEGIN (RSA|EC|OPENSSH|PRIVATE) KEY-----','\b[0-9a-fA-F]{64}\b'
  ) | ForEach-Object { New-Object regex($_,'Compiled') }
  $hits = @()
  foreach ($f in $Files) {
    try {
      $fi = Get-Item $f; if ($fi.Length -gt 10MB) { continue }
      $bytes = [IO.File]::ReadAllBytes($f); if ($bytes -contains 0) { continue }
      $text = [Text.Encoding]::UTF8.GetString($bytes)
      foreach ($rx in $patterns) {
        $m = $rx.Matches($text); if ($m.Count -gt 0) { $hits += [pscustomobject]@{File=$f;Pattern=$rx;Count=$m.Count} }
      }
    } catch {}
  }
  return $hits
}

Ensure-Git
$root = Get-RepoRoot
if (-not $root) { throw "Repo kökünde çalıştırın." }
Set-Location $root

Ensure-GitIgnore -Path ".gitignore"
Untrack-IgnoredIfTracked

git add -A

$staged = Get-StagedFiles
$hits = @(Test-Secrets -Files $staged)
if ($hits.Count -gt 0) {
  $hits | Format-Table -AutoSize | Out-Host
  Write-Host "❌ Gizli anahtar şüphesi. Commit/push iptal." -ForegroundColor Red
  exit 2
}

if (-not $Message) { $Message = "update" }
git commit -m $Message

if (-not $NoPush) {
  git push $Remote $Branch
  Write-Host "✅ Push tamam." -ForegroundColor Green
} else {
  Write-Host "ℹ️ Push atlandı." -ForegroundColor Yellow
}
