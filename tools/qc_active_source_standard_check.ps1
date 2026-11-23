param(
  [switch]$SourceOnly,
  [string]$ReleaseDirectory,
  [string]$ReleaseCommit,
  [string]$TrustedSigners
)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'qc_release_audit.ps1')
$auditArgs = @{
  SourceOnly = $SourceOnly
  ReleaseDirectory = $ReleaseDirectory
  ReleaseCommit = $ReleaseCommit
  TrustedSigners = $TrustedSigners
}
$releaseEvidence = Get-QCReleaseAuditEvidence @auditArgs

$Repo = Split-Path -Parent $PSScriptRoot
$stamp = Get-Date -Format "yyyyMMdd_HHmmss"
$reportDir = Join-Path -Path $Repo -ChildPath "_local_audit_reports"
New-Item -ItemType Directory -Force -Path $reportDir | Out-Null

$report = Join-Path -Path $reportDir -ChildPath "QC_02_ACTIVE_SOURCE_STANDARD_$stamp.txt"

function W($x) {
  $x | Tee-Object -FilePath $report -Append
}

function Rel($full) {
  return $full.Substring($Repo.Length).TrimStart("\")
}

function Category($rel) {
  $p = $rel.Replace("/", "\")
  $pl = $p.ToLowerInvariant()
  $name = [System.IO.Path]::GetFileName($pl)
  $ext = [System.IO.Path]::GetExtension($pl)

  if ($pl -like ".git\*" -or $pl -like "*\.git\*") { return "SKIP_GIT_INTERNAL" }

  # Example environment templates are allowed active source.
  # Real .env files remain local-only sensitive.
  if ($name -eq ".env.example" -or $pl.EndsWith("\.env.example")) { return "ACTIVE_SOURCE" }

  $localSensitiveNames = @(
    ".env",
    ".wallet",
    "wallet_priv.hex",
    "miner_priv.hex",
    "private_key.hex",
    "credentials.json",
    "token.json",
    "secrets.json",
    "private.json"
  )

  if ($localSensitiveNames -contains $name) { return "LOCAL_ONLY_SENSITIVE" }
  if ($name -like ".env*") { return "LOCAL_ONLY_SENSITIVE" }
  if ($name -like "*wallet*priv*") { return "LOCAL_ONLY_SENSITIVE" }
  if ($name -like "*miner*priv*") { return "LOCAL_ONLY_SENSITIVE" }
  if ($name -like "*private*key*") { return "LOCAL_ONLY_SENSITIVE" }
  if ($name -like "*credential*.json") { return "LOCAL_ONLY_SENSITIVE" }
  if ($name -like "*secret*.json") { return "LOCAL_ONLY_SENSITIVE" }
  if ($name -like "*token.json") { return "LOCAL_ONLY_SENSITIVE" }

  $runtimeExt = @(".exe",".dll",".pdb",".db",".sqlite",".sqlite3",".dat",".log",".zip",".7z",".rar",".tar",".gz")
  if ($runtimeExt -contains $ext) { return "LOCAL_RUNTIME_OR_BINARY" }

  if ($pl -like "node_modules\*" -or $pl -like "*\node_modules\*") { return "DEPENDENCY_GENERATED" }
  if ($pl -like "webwallet\dist\*") { return "LOCAL_RUNTIME_OR_BINARY" }
  if ($pl -like "bin\*" -or $pl -like "build\*" -or $pl -like "dist\*" -or $pl -like "*\__pycache__\*") { return "LOCAL_RUNTIME_OR_BINARY" }
  if ($pl -like "_local_audit_reports\*" -or $pl -like "_local_audit_backups\*") { return "LOCAL_AUDIT_OUTPUT" }

  if ($name -like "*.bak") { return "ARCHIVE_BACKUP" }
  if ($name -like "*.before_*") { return "ARCHIVE_BACKUP" }
  if ($name -like "main.go.before_*") { return "ARCHIVE_BACKUP" }
  if ($name -like "*_before_*") { return "ARCHIVE_BACKUP" }
  if ($name -like "*broken*") { return "ARCHIVE_BACKUP" }
  if ($name -like "*baseline*") { return "ARCHIVE_BACKUP" }

  $archivePrefixes = @(
    "_checkpoints\",
    "_legacy_runtime_",
    "_root_exe_quarantine\",
    "_wm_review\",
    "old_working_release\",
    "release_backup_",
    "release_clean_test\",
    "release_clean_test_old_",
    "quantumcoin_runtime_backup\",
    "quantumcoin_main_pr_tmp\"
  )

  foreach ($prefix in $archivePrefixes) {
    if ($pl.StartsWith($prefix)) { return "ARCHIVE_BACKUP" }
  }

  if ($name -like "*.disabled" -or $pl -like "src\*" -or
      $pl -like "gui\*" -or $pl -like "ui\*" -or $pl -like "qc-desktop\*") {
    return "RETIRED_HISTORICAL_SOURCE"
  }
  if ($name -like "*_test.go") { return "VERIFICATION_SOURCE" }
  if ($pl -like "i18n\*" -or $pl -like "tests\*") { return "UNWIRED_SUPPORT_OR_DIAGNOSTIC" }

  if ($pl -like "autopublisher\tmp\*" -or $pl -like "autopublisher\output\*" -or $pl -like "autopublisher\outputs\*" -or $pl -like "autopublisher\media\*" -or $pl -like "autopublisher\videos\*" -or $pl -like "autopublisher\images\*" -or $pl -like "autopublisher\cache\*" -or $pl -like "autopublisher\logs\*") {
    return "LOCAL_RUNTIME_OR_BINARY"
  }

  $activeRootFiles = @(
    ".editorconfig",
    ".gitattributes",
    ".gitignore",
    ".golangci.yml",
    "config.json.example",
    "cspell.json",
    "go.mod",
    "go.sum",
    "http_security.go",
    "main.go",
    "package.json",
    "package-lock.json",
    "readme.md",
    "staticcheck.conf"
  )

  if ($activeRootFiles -contains $pl) { return "ACTIVE_SOURCE" }

  $activePrefixes = @(
    "ai\",
    "api\",
    "assets\",
    "autopublisher\src\",
    "blockchain\",
    "cmd\",
    "config\",
    "consolefx\",
    "docs\",
    "game\",
    "internal\",
    "miner\",
    "nft\",
    "p2p\",
    "security\",
    "telegram_game\telegram_game\",
    "telegram_game\mini_regression_v1.py",
    "telegram_game\requirements.txt",
    "testnet\",
    "token\",
    "tools\",
    "utils\",
    "wallet\",
    "webui\",
    "scripts\",
    "webwallet\"
  )

  foreach ($prefix in $activePrefixes) {
    if ($pl.StartsWith($prefix)) { return "ACTIVE_SOURCE" }
  }

  return "REVIEW_UNCLASSIFIED"
}

Set-Location -Path $Repo

W "=== QC 02 ACTIVE SOURCE STANDARD CHECK ==="
W "Repo: $Repo"
W "Stamp: $stamp"
W ""

W "=== Release gate ==="
W $releaseEvidence
W ""

W "=== Required active files/dirs ==="

$requiredFiles = @(
  "main.go",
  "http_security.go",
  "go.mod",
  "go.sum",
  ".gitignore",
  "tools\qc_repo_safety_check.ps1",
  "tools\qc_active_source_standard_check.ps1",
  "docs\QC_ACTIVE_SOURCE_STANDARD.md"
)

$requiredDirs = @(
  "ai",
  "api",
  "blockchain",
  "cmd",
  "config",
  "internal",
  "miner",
  "nft",
  "p2p",
  "security",
  "telegram_game\telegram_game",
  "testnet",
  "webui",
  "token",
  "tools",
  "wallet"
)

$missing = @()

foreach ($f in $requiredFiles) {
  if (!(Test-Path (Join-Path -Path $Repo -ChildPath $f))) { $missing += $f }
}

foreach ($d in $requiredDirs) {
  if (!(Test-Path (Join-Path -Path $Repo -ChildPath $d))) { $missing += $d }
}

if ($missing.Count -gt 0) {
  W "Missing required active entries:"
  $missing | ForEach-Object { W $_ }
  throw "Required active source entries missing"
}

W "Required active files/dirs OK."
W ""

W "=== File classification summary ==="

$rows = New-Object System.Collections.Generic.List[object]

Get-ChildItem -LiteralPath $Repo -Recurse -File -Force -ErrorAction SilentlyContinue | ForEach-Object {
  $r = Rel $_.FullName
  $c = Category $r
  if ($c -ne "SKIP_GIT_INTERNAL") {
    $rows.Add([pscustomobject]@{ Category = $c; Path = $r })
  }
}

$rows |
  Group-Object Category |
  Sort-Object Name |
  ForEach-Object {
    W ("{0}`t{1}" -f $_.Name, $_.Count)
  }

W ""
W "=== Review unclassified sample ==="

$review = @($rows | Where-Object { $_.Category -eq "REVIEW_UNCLASSIFIED" } | Select-Object -First 80)

if ($review.Count -eq 0) {
  W "No REVIEW_UNCLASSIFIED files OK."
} else {
  $review | ForEach-Object { W $_.Path }
}

W ""

W "=== Git tracked risky file check ==="

git -C $Repo rev-parse --is-inside-work-tree *> $null
if ($LASTEXITCODE -eq 0) {
  $tracked = @(git -C $Repo ls-files)
  $badTracked = @($tracked | Where-Object {
    $cat = Category $_
    $cat -eq "LOCAL_ONLY_SENSITIVE" -or
    $cat -eq "LOCAL_RUNTIME_OR_BINARY" -or
    $cat -eq "DEPENDENCY_GENERATED"
  })

  if ($badTracked.Count -gt 0) {
    W "Tracked risky files:"
    $badTracked | Sort-Object | ForEach-Object { W $_ }
    throw "Tracked risky files found"
  }

  W "No tracked risky files OK."
} else {
  W "Git tracked risky file check skipped."
}

W ""
W "QC_02_ACTIVE_SOURCE_STANDARD_PASS"
W "QC_02_ACTIVE_SOURCE_STANDARD_DONE"
W "Report: $report"
