# QC_33C7_PSSCRIPTANALYZER_LEGACY_SUPPRESSIONS
[Diagnostics.CodeAnalysis.SuppressMessageAttribute('PSUseApprovedVerbs', '', Justification='QC 33C7: legacy/test/run script naming or control-flow retained to avoid behavior change while zeroing analyzer warnings.')]
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

$report = Join-Path -Path $reportDir -ChildPath "QC_06A_LEGACY_NOISE_AUDIT_$stamp.txt"

function W($x) {
  $x | Tee-Object -FilePath $report -Append
}

function Rel($full) {
  return $full.Substring($Repo.Length).TrimStart("\")
}

function Is-LocalReleaseRuntimePath([string]$rel) {
  $p = $rel.Replace("/", "\").ToLowerInvariant()
  return $p -eq "release\wallet-gui.exe" -or
         $p -eq "release\quantumcoin.exe" -or
         $p -eq "release\run_miner.cmd" -or
         $p -eq "release\_qc_api_mempool_miner.ps1"
}

function Classify-LegacyNoise([string]$rel) {
  $p = $rel.Replace("/", "\")
  $pl = $p.ToLowerInvariant()
  $name = [System.IO.Path]::GetFileName($pl)
  $ext = [System.IO.Path]::GetExtension($pl)

  if (Is-LocalReleaseRuntimePath $p) { return "PROTECTED_LOCAL_RELEASE_RUNTIME" }

  if ($pl -like ".git\*" -or $pl -like "*\.git\*") { return "SKIP_GIT_INTERNAL" }
  if ($pl -like "_local_audit_reports\*" -or $pl -like "_local_audit_backups\*") { return "LOCAL_AUDIT_OUTPUT" }

  if ($pl -eq ".wallet" -or $pl -eq ".env" -or $name -eq "wallet_priv.hex" -or $name -eq "miner_priv.hex") {
    return "PROTECTED_LOCAL_SECRET_OR_WALLET"
  }

  if ($name -eq "chain_data.dat" -or $name -eq "qc_blockchain.db" -or $name -eq "qc_blockchain.db.lock") {
    return "PROTECTED_RUNTIME_STATE"
  }

  if ($name -like "main.go.before_*") { return "SAFE_LEGACY_BACKUP_FILE" }
  if ($name -like "*.before_*") { return "SAFE_LEGACY_BACKUP_FILE" }
  if ($name -like "*_before_*") { return "SAFE_LEGACY_BACKUP_FILE" }
  if ($name -like "*.bak") { return "SAFE_LEGACY_BACKUP_FILE" }

  if ($name -eq "build_errors.txt" -or $name -like "*_errors.txt" -or $name -like "*error*.txt") {
    return "SAFE_OLD_ERROR_REPORT"
  }

  if ($ext -eq ".log") { return "SAFE_OLD_LOG" }

  $oldRootText = @(
    "_flat.txt",
    "_goenv.txt",
    "_gomod.txt",
    "_tree.txt",
    "cmd-tree.txt",
    "project_structure.txt",
    "project-tree.txt",
    "release-tree.txt",
    "walletminer-tree.txt",
    "lint_report.txt",
    "README.md.rtf"
  )

  if ($oldRootText -contains $name) { return "SAFE_OLD_REPORT_TEXT" }

  if ($name -eq "bonus_store.json" -or $name -eq "mined_balance.json" -or $name -eq "wallet_balance.cache.json") {
    return "SAFE_OLD_LOCAL_CACHE"
  }

  $oldFolders = @(
    "__qcdiag\",
    "_checkpoints\",
    "_legacy_runtime_",
    "_root_exe_quarantine\",
    "_wm_review\",
    "old_working_release\",
    "release_backup_",
    "release_clean_test\",
    "release_clean_test_old_"
  )

  foreach ($prefix in $oldFolders) {
    if ($pl.StartsWith($prefix)) { return "SAFE_OLD_FOLDER_CONTENT" }
  }

  if ($pl -like "telegram_game\telegram_game\dev_api_before_*") { return "SAFE_LEGACY_BACKUP_FILE" }
  if ($pl -like "telegram_game\telegram_game\dev_ui\*before_*") { return "SAFE_LEGACY_BACKUP_FILE" }
  if ($pl -like "telegram_game\telegram_game\dev_ui\*broken*") { return "SAFE_LEGACY_BACKUP_FILE" }
  if ($pl -like "telegram_game\telegram_game\dev_ui\*baseline*") { return "SAFE_LEGACY_BACKUP_FILE" }

  if ($pl -like "autopublisher\tmp\*" -or $pl -like "autopublisher\cache\*" -or $pl -like "autopublisher\logs\*") {
    return "SAFE_GENERATED_AUTOPUBLISHER_OUTPUT"
  }

  return "NOT_A_6A_CANDIDATE"
}

Set-Location -Path $Repo

W "=== QC 06A LEGACY / BUILD / ERROR / BACKUP AUDIT ==="
W "Repo: $Repo"
W "Stamp: $stamp"
W ""

W "=== Previous gates ==="

& (Join-Path $PSScriptRoot 'qc_repo_safety_check.ps1') @auditArgs | Tee-Object -FilePath $report -Append
& (Join-Path $PSScriptRoot 'qc_active_source_standard_check.ps1') @auditArgs | Tee-Object -FilePath $report -Append
& (Join-Path $PSScriptRoot 'qc_main_entrypoint_check.ps1') @auditArgs | Tee-Object -FilePath $report -Append
& (Join-Path $PSScriptRoot 'qc_go_build_targets_check.ps1') @auditArgs | Tee-Object -FilePath $report -Append

W ""
W "=== Release gate ==="
W $releaseEvidence

W ""
W "=== 6A candidate classification summary ==="

$rows = New-Object System.Collections.Generic.List[object]

Get-ChildItem -LiteralPath $Repo -Recurse -File -Force -ErrorAction SilentlyContinue |
  Where-Object { $_.FullName -notlike "*\.git\*" } |
  ForEach-Object {
    $rel = Rel $_.FullName
    $cat = Classify-LegacyNoise $rel
    if ($cat -ne "SKIP_GIT_INTERNAL" -and $cat -ne "NOT_A_6A_CANDIDATE" -and $cat -ne "LOCAL_AUDIT_OUTPUT") {
      $rows.Add([pscustomobject]@{
        Category = $cat
        Path = $rel
        Size = $_.Length
      })
    }
  }

$rows |
  Group-Object Category |
  Sort-Object Name |
  ForEach-Object {
    $total = ($_.Group | Measure-Object Size -Sum).Sum
    W ("{0}`tCount={1}`tBytes={2}" -f $_.Name, $_.Count, $total)
  }

W ""
W "=== Safe 6A candidates sample ==="

$safeCats = @(
  "SAFE_LEGACY_BACKUP_FILE",
  "SAFE_OLD_ERROR_REPORT",
  "SAFE_OLD_LOG",
  "SAFE_OLD_REPORT_TEXT",
  "SAFE_OLD_LOCAL_CACHE",
  "SAFE_OLD_FOLDER_CONTENT",
  "SAFE_GENERATED_AUTOPUBLISHER_OUTPUT"
)

$safeRows = @($rows | Where-Object { $safeCats -contains $_.Category } | Sort-Object Category,Path)

if ($safeRows.Count -eq 0) {
  W "No safe legacy/noise candidates found."
} else {
  $safeRows | Select-Object -First 250 | ForEach-Object {
    W ("{0}`t{1}`t{2}" -f $_.Category, $_.Size, $_.Path)
  }
  if ($safeRows.Count -gt 250) {
    W "... more safe candidates not shown in sample ..."
  }
}

W ""
W "=== Protected files detected but not move candidates ==="

$protected = @($rows | Where-Object { $_.Category -like "PROTECTED_*" } | Sort-Object Category,Path)

if ($protected.Count -eq 0) {
  W "No protected files detected in candidate scan."
} else {
  $protected | ForEach-Object {
    W ("{0}`t{1}`t{2}" -f $_.Category, $_.Size, $_.Path)
  }
}

W ""
W "=== Git tracked safe-candidate check ==="

git -C $Repo rev-parse --is-inside-work-tree *> $null
if ($LASTEXITCODE -eq 0) {
  $tracked = @(git -C $Repo ls-files)
  $trackedSafe = @($tracked | Where-Object {
    $c = Classify-LegacyNoise $_
    $safeCats -contains $c
  })

  if ($trackedSafe.Count -gt 0) {
    W "Tracked safe-candidate files exist. They must be reviewed before moving:"
    $trackedSafe | Sort-Object | ForEach-Object { W $_ }
  } else {
    W "No tracked safe-candidate files OK."
  }
} else {
  W "Git tracked check skipped."
}

W ""
W "QC_06A_LEGACY_NOISE_AUDIT_PASS"
W "QC_06A_LEGACY_NOISE_AUDIT_DONE"
W "Report: $report"
