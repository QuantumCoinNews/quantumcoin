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

$report = Join-Path -Path $reportDir -ChildPath "QC_04_MAIN_ENTRYPOINTS_$stamp.txt"

function W($x) {
  $x | Tee-Object -FilePath $report -Append
}

function Classify-MainGo {
  param([string]$Rel)

  $p = $Rel.Replace("/", "\").ToLowerInvariant()

  if ($p -eq "cmd\wallet-crypto\main.go") { return "BROWSER_WALLET_WASM" }
  if ($p -eq "main.go") { return "ACTIVE_ROOT_NODE" }

  if ($p -eq "cmd\miner\main.go") { return "ACTIVE_MINER_CLI" }

  if ($p -eq "cmd\quantumcoin-gui\main.go") { return "ACTIVE_GUI" }

  if ($p -like "cmd\qc_deterministic_wallet\main.go") { return "RETIRED_DEMO" }
  if ($p -like "cmd\qc_recovery_init\main.go") { return "ACTIVE_WALLET_TOOL" }
  if ($p -like "cmd\qc_seed_canonical_check\main.go") { return "RETIRED_DEMO" }
  if ($p -like "cmd\qc_seed_manager\main.go") { return "ACTIVE_WALLET_TOOL" }
  if ($p -like "cmd\qc_wallet_create\main.go") { return "ACTIVE_WALLET_TOOL" }
  if ($p -like "cmd\qc_wallet_restore\main.go") { return "ACTIVE_WALLET_TOOL" }

  if ($p -in @("cmd\gui-smoke\main.go", "cmd\policy-export\main.go")) { return "TEST_OR_HELPER" }
  if ($p -eq "cmd\recovery\main.go") { return "ACTIVE_WALLET_TOOL" }
  if ($p -like "cmd\apitest\*") { return "TEST_OR_HELPER" }

  if ($p -like "_wm_review\*") { return "ARCHIVE_REVIEW" }

  return "UNKNOWN_MAIN_ENTRYPOINT"
}

Set-Location -Path $Repo

W "=== QC 04 MAIN ENTRYPOINTS CHECK ==="
W "Repo: $Repo"
W "Stamp: $stamp"
W ""

W "=== Previous gates ==="

& (Join-Path $PSScriptRoot 'qc_repo_safety_check.ps1') @auditArgs | Tee-Object -FilePath $report -Append
& (Join-Path $PSScriptRoot 'qc_active_source_standard_check.ps1') @auditArgs | Tee-Object -FilePath $report -Append

W ""
W "=== Release gate ==="
W $releaseEvidence
W ""

W "=== All main.go files ==="

$mainFiles = Get-ChildItem -LiteralPath $Repo -Recurse -File -Force -Filter "main.go" |
  Where-Object { $_.FullName -notlike "*\.git\*" } |
  ForEach-Object {
    $rel = $_.FullName.Substring($Repo.Length).TrimStart("\")
    $hash = (Get-FileHash $_.FullName -Algorithm SHA256).Hash
    $class = Classify-MainGo $rel
    [pscustomobject]@{
      Class = $class
      Path = $rel
      SHA256 = $hash
    }
  } |
  Sort-Object Class,Path

foreach ($m in $mainFiles) {
  W ("{0}`t{1}`t{2}" -f $m.Class, $m.Path, $m.SHA256)
}

W ""
W "=== Entrypoint class summary ==="

$mainFiles |
  Group-Object Class |
  Sort-Object Name |
  ForEach-Object {
    W ("{0}`t{1}" -f $_.Name, $_.Count)
  }

W ""
W "=== Required entrypoint check ==="

$required = @(
  "main.go",
  "cmd\miner\main.go",
  "cmd\quantumcoin-gui\main.go",
  "cmd\qc_wallet_create\main.go",
  "cmd\qc_wallet_restore\main.go"
)

$missing = @()

foreach ($r in $required) {
  if (!(Test-Path (Join-Path -Path $Repo -ChildPath $r))) {
    $missing += $r
  }
}

if ($missing.Count -gt 0) {
  W "Missing required main.go entrypoints:"
  $missing | ForEach-Object { W $_ }
  throw "Required main.go entrypoints missing"
}

W "Required main.go entrypoints OK."

W ""
W "=== Unknown main.go check ==="

$unknown = @($mainFiles | Where-Object { $_.Class -eq "UNKNOWN_MAIN_ENTRYPOINT" })

if ($unknown.Count -gt 0) {
  W "Unknown main.go entrypoints:"
  $unknown | ForEach-Object { W $_.Path }
  throw "Unknown main.go entrypoint found"
}

W "No unknown main.go entrypoints OK."

W ""
W "=== Active vs archive rule ==="

$archive = @($mainFiles | Where-Object { $_.Class -eq "ARCHIVE_REVIEW" })
if ($archive.Count -gt 0) {
  W "Archive/review main.go files detected but classified:"
  $archive | ForEach-Object { W $_.Path }
}

W "Archive/review main.go files are not active source OK."

W ""
W "QC_04_MAIN_ENTRYPOINTS_PASS"
W "QC_04_MAIN_ENTRYPOINTS_DONE"
W "Report: $report"
