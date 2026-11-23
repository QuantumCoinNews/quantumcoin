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
$repo = Split-Path -Parent $PSScriptRoot
$stamp = Get-Date -Format "yyyyMMdd_HHmmss"
$reportDir = Join-Path -Path $repo -ChildPath "_local_audit_reports"
New-Item -ItemType Directory -Force -Path $reportDir | Out-Null
$report = Join-Path -Path $reportDir -ChildPath "QC_01_REPO_SAFETY_CHECK_$stamp.txt"
function W($x){ $x | Tee-Object -FilePath $report -Append }
Set-Location -Path $repo
W "=== QC 01 REPO SAFETY CHECK ==="
W "Repo: $repo"
W "Stamp: $stamp"
W ""
W "=== Release gate ==="
W $releaseEvidence
W ""
$ignoredExamples = @(".env", "nested/.env.local", "nested/.wallet", "release/wallet_priv.hex",
  "release/wallet_recovery_words.txt", "nested/credentials.json", "nested/token.json",
  "release/chain_data_testnet.dat", "nested/example.db-wal", "release/node.exe",
  "release/_testnet_backups/fixture/manifest.txt", "_local_audit_reports/fixture.txt", "node_modules/fixture/index.js")
$visibleExamples = @(".env.example", "autopublisher/.env.example", "release/config.testnet.json",
  "release/release-manifest.example.json", "run_miner.cmd")
foreach ($path in $ignoredExamples) {
  git check-ignore --quiet --no-index -- $path
  if ($LASTEXITCODE -ne 0) { throw "Runtime path is not ignored: $path" }
}
foreach ($path in $visibleExamples) {
  git check-ignore --quiet --no-index -- $path
  if ($LASTEXITCODE -ne 1) { throw "Reviewed source path is ignored or unreadable: $path" }
}
W "Git ignore decisions verified for runtime and reviewed source paths."
W ""
$riskNames = @(".env",".wallet","wallet_priv.hex","miner_priv.hex","private_key.hex","credentials.json","token.json","chain_data.dat","chain_data_testnet.dat","qc_blockchain.db")
$found = @(Get-ChildItem -LiteralPath $repo -Recurse -Force -ErrorAction SilentlyContinue -File |
  Where-Object { $_.Name -in $riskNames } |
  ForEach-Object { $_.FullName.Substring($repo.Length).TrimStart("\") })
W "Local sensitive/runtime filename findings: $($found.Count)"
if($found.Count -gt 0){ $found | Sort-Object | ForEach-Object { W "IGNORED/LOCAL: $_" } }
git -C $repo rev-parse --is-inside-work-tree *> $null
if($LASTEXITCODE -eq 0){
  $tracked = @(git -C $repo ls-files)
  $badTracked = @($tracked | Where-Object {
    $x = $_.ToLowerInvariant()
    $x -eq ".env" -or $x -eq ".wallet" -or $x -like "*wallet*priv*" -or $x -like "*wallet_recovery_words.txt" -or $x -like "*wallet_recovery_manifest.json" -or $x -like "*miner*priv*" -or $x -like "*private*key*" -or $x -like "*credentials.json" -or $x -like "*token.json" -or $x -like "*chain_data*.dat" -or $x -like "*.db" -or $x -like "*.sqlite" -or $x -like "*.zip"
  })
  if($badTracked.Count -gt 0){ W "Tracked risky files:"; $badTracked | ForEach-Object { W $_ }; throw "Tracked risky files found" }
  W "No tracked risky files OK."
} else {
  W "Git tracked check skipped."
}
W ""
W "QC_01_REPO_SAFETY_CHECK_PASS"
W "QC_01_AUDIT_PACKAGE_SECURITY_DONE"
W "Report: $report"
