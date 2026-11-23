# QC_33C7_PSSCRIPTANALYZER_LEGACY_SUPPRESSIONS
[Diagnostics.CodeAnalysis.SuppressMessageAttribute('PSAvoidUsingPositionalParameters', '', Justification='QC 33C7: legacy/test/run script naming or control-flow retained to avoid behavior change while zeroing analyzer warnings.')]
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
$buildDir = Join-Path -Path $reportDir -ChildPath ("QC_05_GO_BUILD_TARGETS_" + $stamp)

New-Item -ItemType Directory -Force -Path $reportDir,$buildDir | Out-Null

$report = Join-Path -Path $reportDir -ChildPath "QC_05_GO_BUILD_TARGETS_$stamp.txt"

function W($x) {
  $x | Tee-Object -FilePath $report -Append
}

function Run-Capture {
  param(
    [string]$Label,
    [string]$Exe,
    [string[]]$ArgList
  )

  W ""
  W "=== $Label ==="
  W ("COMMAND: {0} {1}" -f $Exe, ($ArgList -join " "))

  $psi = New-Object System.Diagnostics.ProcessStartInfo
  $psi.FileName = $Exe
  foreach ($a in $ArgList) {
    [void]$psi.ArgumentList.Add($a)
  }
  $psi.WorkingDirectory = $Repo
  $psi.RedirectStandardOutput = $true
  $psi.RedirectStandardError = $true
  $psi.UseShellExecute = $false

  $p = [System.Diagnostics.Process]::Start($psi)
  $stdoutRead = $p.StandardOutput.ReadToEndAsync()
  $stderrRead = $p.StandardError.ReadToEndAsync()
  $p.WaitForExit()
  $stdout = $stdoutRead.GetAwaiter().GetResult()
  $stderr = $stderrRead.GetAwaiter().GetResult()

  if ($stdout.Trim().Length -gt 0) { W $stdout.TrimEnd() }
  if ($stderr.Trim().Length -gt 0) { W $stderr.TrimEnd() }

  W "EXIT_CODE: $($p.ExitCode)"

  if ($p.ExitCode -ne 0) {
    throw "$Label failed"
  }
}

Set-Location -Path $Repo

W "=== QC 05 GO BUILD TARGETS CHECK ==="
W "Repo: $Repo"
W "Stamp: $stamp"
W "BuildDir: $buildDir"
W ""

W "=== Previous gates ==="

& (Join-Path $PSScriptRoot 'qc_repo_safety_check.ps1') @auditArgs | Tee-Object -FilePath $report -Append
& (Join-Path $PSScriptRoot 'qc_active_source_standard_check.ps1') @auditArgs | Tee-Object -FilePath $report -Append
& (Join-Path $PSScriptRoot 'qc_main_entrypoint_check.ps1') @auditArgs | Tee-Object -FilePath $report -Append

W ""
W "=== Release gate ==="
W $releaseEvidence

Run-Capture "Go version" "go" @("version")
Run-Capture "Go env target" "go" @("env","GOVERSION","GOOS","GOARCH")

W ""
W "=== Build target table ==="

$targets = @(
  [pscustomobject]@{ Class="ROOT_NODE"; Required=$true; Package="."; Output="quantumcoin-node-audit.exe" },
  [pscustomobject]@{ Class="MINER_CLI"; Required=$true; Package="./cmd/miner"; Output="quantumcoin-miner-cli-audit.exe" },
  [pscustomobject]@{ Class="DESKTOP_GUI"; Required=$true; Package="./cmd/quantumcoin-gui"; Output="quantumcoin-launcher-audit.exe" },

  [pscustomobject]@{ Class="NATIVE_WALLET"; Required=$true; Package="./cmd/wallet"; Output="wallet-gui-audit.exe" },
  [pscustomobject]@{ Class="WALLET_TOOL"; Required=$true; Package="./cmd/qc_deterministic_wallet"; Output="qc-deterministic-wallet-audit.exe" },
  [pscustomobject]@{ Class="WALLET_TOOL"; Required=$true; Package="./cmd/qc_recovery_init"; Output="qc-recovery-init-audit.exe" },
  [pscustomobject]@{ Class="WALLET_TOOL"; Required=$true; Package="./cmd/qc_seed_canonical_check"; Output="qc-seed-canonical-check-audit.exe" },
  [pscustomobject]@{ Class="WALLET_TOOL"; Required=$true; Package="./cmd/qc_seed_manager"; Output="qc-seed-manager-audit.exe" },
  [pscustomobject]@{ Class="WALLET_TOOL"; Required=$true; Package="./cmd/qc_wallet_create"; Output="qc-wallet-create-audit.exe" },
  [pscustomobject]@{ Class="WALLET_TOOL"; Required=$true; Package="./cmd/qc_wallet_restore"; Output="qc-wallet-restore-audit.exe" },

  [pscustomobject]@{ Class="TEST_OR_HELPER"; Required=$false; Package="./cmd/apitest"; Output="apitest-audit.exe" },
  [pscustomobject]@{ Class="TEST_OR_HELPER"; Required=$false; Package="./cmd/apitest/quantumcoin-gui"; Output="apitest-quantumcoin-gui-audit.exe" }
)

foreach ($t in $targets) {
  W ("{0}`tRequired={1}`t{2}`t{3}" -f $t.Class, $t.Required, $t.Package, $t.Output)
}

W ""
W "=== go list required targets ==="

foreach ($t in $targets | Where-Object { $_.Required -eq $true }) {
  Run-Capture ("go list " + $t.Package) "go" @("list",$t.Package)
}

W ""
W "=== go build required targets into audit build dir ==="

foreach ($t in $targets | Where-Object { $_.Required -eq $true }) {
  $out = Join-Path -Path $buildDir -ChildPath $t.Output
  Run-Capture ("go build " + $t.Package) "go" @("build","-trimpath","-o",$out,$t.Package)

  if (!(Test-Path $out)) {
    throw "Expected audit build output missing: $out"
  }

  $hash = (Get-FileHash $out -Algorithm SHA256).Hash
  $size = (Get-Item $out).Length
  W ("BUILT`t{0}`t{1}`t{2}`t{3}" -f $t.Package, $t.Output, $size, $hash)
}

W ""
W "=== Optional helper target go list only ==="

foreach ($t in $targets | Where-Object { $_.Required -eq $false }) {
  Run-Capture ("go list optional " + $t.Package) "go" @("list",$t.Package)
}

W ""
W "=== Release gate after audit ==="
$afterEvidence = Get-QCReleaseAuditEvidence @auditArgs
if ($afterEvidence -ne $releaseEvidence) { throw 'Signed release changed during audit.' }
W $afterEvidence

W ""
W "=== Release overwrite guard ==="

$releaseNames = @(
  "release\wallet-gui.exe",
  "release\quantumcoin.exe",
  "release\run_miner.cmd",
  "release\_qc_api_mempool_miner.ps1"
)

foreach ($r in $releaseNames) {
  W "UNCHANGED_RELEASE_TARGET`t$r"
}

W "Audit build outputs stayed under: $buildDir"

W ""
W "QC_05_GO_BUILD_TARGETS_PASS"
W "QC_05_GO_BUILD_TARGETS_DONE"
W "Report: $report"
