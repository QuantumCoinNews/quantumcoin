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
$buildDir = Join-Path -Path $reportDir -ChildPath ("QC_08_BLOCKCHAIN_PACKAGE_AUDIT_" + $stamp)

New-Item -ItemType Directory -Force -Path $reportDir,$buildDir | Out-Null

$report = Join-Path -Path $reportDir -ChildPath "QC_08_BLOCKCHAIN_PACKAGE_AUDIT_$stamp.txt"

function W($x) {
  $x | Tee-Object -FilePath $report -Append
}

function Run-Capture {
  param(
    [string]$Label,
    [string]$Exe,
    [string[]]$ArgList,
    [bool]$AllowFail = $false
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

  if ($p.ExitCode -ne 0 -and -not $AllowFail) {
    throw "$Label failed"
  }
}

function Rel($full) {
  return $full.Substring($Repo.Length).TrimStart("\")
}

function IsIgnoredAuditOrArchivePath([string]$rel) {
  $p = $rel.Replace("/", "\").ToLowerInvariant()

  return $p -like ".git\*" -or
         $p -like "_legacy_noise_archive\*" -or
         $p -like "_local_audit_reports\*" -or
         $p -like "_local_audit_backups\*" -or
         $p -like "_checkpoints\*" -or
         $p -like "_wm_review\*" -or
         $p -like "_legacy_runtime_*" -or
         $p -like "_root_exe_quarantine\*" -or
         $p -like "old_working_release\*" -or
         $p -like "release_backup_*" -or
         $p -like "release_clean_test*" -or
         $p -like "__qcdiag\*" -or
         $p -like "vendor\*" -or
         $p -like "*\node_modules\*"
}

Set-Location -Path $Repo

W "=== QC 08 BLOCKCHAIN PACKAGE AUDIT ==="
W "Repo: $Repo"
W "Stamp: $stamp"
W "BuildDir: $buildDir"
W ""

W "=== Release gate ==="
W $releaseEvidence

W ""
W "=== Canonical blockchain folder check ==="

if (!(Test-Path "blockchain")) {
  throw "Canonical blockchain folder missing: blockchain/"
}

$bcFiles = @(Get-ChildItem -LiteralPath "blockchain" -Recurse -File -Force -Filter "*.go")
W ("CANONICAL_BLOCKCHAIN_GO_FILES`tCount={0}" -f $bcFiles.Count)

foreach ($f in $bcFiles | Sort-Object FullName) {
  $rel = Rel $f.FullName
  $hash = (Get-FileHash $f.FullName -Algorithm SHA256).Hash
  W ("CANONICAL_FILE`t{0}`tSize={1}`tSHA256={2}" -f $rel, $f.Length, $hash)
}

W ""
W "=== package blockchain declarations ==="

$allGo = Get-ChildItem -LiteralPath $Repo -Recurse -File -Force -Filter "*.go" |
  Where-Object { $_.FullName -notlike "*\.git\*" }

$pkgBlockchain = New-Object System.Collections.Generic.List[object]

foreach ($f in $allGo) {
  $rel = Rel $f.FullName
  $firstPkg = Select-String -Path $f.FullName -Pattern "^\s*package\s+blockchain\s*$" -ErrorAction SilentlyContinue | Select-Object -First 1
  if ($firstPkg) {
    $ignored = IsIgnoredAuditOrArchivePath $rel
    $pkgBlockchain.Add([pscustomobject]@{
      Path = $rel
      Line = $firstPkg.LineNumber
      IgnoredOrArchive = $ignored
    })
  }
}

if ($pkgBlockchain.Count -eq 0) {
  throw "No package blockchain declarations found"
}

foreach ($row in $pkgBlockchain | Sort-Object Path) {
  W ("PACKAGE_BLOCKCHAIN`tIgnoredOrArchive={0}`tLine={1}`t{2}" -f $row.IgnoredOrArchive, $row.Line, $row.Path)
}

$activePkgBlockchain = @($pkgBlockchain | Where-Object { $_.IgnoredOrArchive -eq $false })

W ""
W "=== active package blockchain declaration summary ==="
W ("ACTIVE_PACKAGE_BLOCKCHAIN_COUNT`t{0}" -f $activePkgBlockchain.Count)

foreach ($row in $activePkgBlockchain | Sort-Object Path) {
  W ("ACTIVE_PACKAGE_BLOCKCHAIN`t{0}" -f $row.Path)
}

$badActive = @($activePkgBlockchain | Where-Object { $_.Path.Replace("/", "\").ToLowerInvariant() -notlike "blockchain\*" })

if ($badActive.Count -gt 0) {
  W "Unexpected active package blockchain files found outside blockchain/:"
  $badActive | ForEach-Object { W $_.Path }
  throw "Duplicate active blockchain package detected"
}

W "Only canonical active blockchain package detected OK."

W ""
W "=== Import usage of quantumcoin/blockchain ==="

$imports = New-Object System.Collections.Generic.List[object]

foreach ($f in $allGo) {
  $rel = Rel $f.FullName
  if (IsIgnoredAuditOrArchivePath $rel) { continue }

  Select-String -Path $f.FullName -Pattern '"quantumcoin/blockchain"' -SimpleMatch -ErrorAction SilentlyContinue |
    ForEach-Object {
      $imports.Add([pscustomobject]@{
        Path = $rel
        Line = $_.LineNumber
        Text = $_.Line.Trim()
      })
    }
}

if ($imports.Count -eq 0) {
  W "NO_ACTIVE_IMPORTS_OF_QUANTUMCOIN_BLOCKCHAIN_FOUND"
} else {
  foreach ($i in $imports | Sort-Object Path,Line) {
    W ("IMPORT_BLOCKCHAIN`t{0}`tLine={1}`t{2}" -f $i.Path, $i.Line, $i.Text)
  }
}

W ""
W "=== Suspicious blockchain-like folders ==="

$dirs = @(Get-ChildItem -LiteralPath $Repo -Recurse -Directory -Force -ErrorAction SilentlyContinue |
  Where-Object { $_.FullName -notlike "*\.git\*" -and $_.Name -match "blockchain|chain|ledger" } |
  ForEach-Object {
    $rel = Rel $_.FullName
    if (IsIgnoredAuditOrArchivePath $rel) {
      [pscustomobject]@{Path=$rel; IgnoredOrArchive=$true}
    } else {
      [pscustomobject]@{Path=$rel; IgnoredOrArchive=$false}
    }
  } |
  Sort-Object Path)

if ($dirs.Count -eq 0) {
  W "No blockchain-like directories found."
} else {
  foreach ($d in $dirs) {
    W ("CHAIN_DIR`tIgnoredOrArchive={0}`t{1}" -f $d.IgnoredOrArchive, $d.Path)
  }
}

$unexpectedDirs = @($dirs | Where-Object {
  $_.IgnoredOrArchive -eq $false -and
  $_.Path.Replace("/", "\").ToLowerInvariant() -ne "blockchain"
})

if ($unexpectedDirs.Count -gt 0) {
  W "Non-archive blockchain-like folders exist. Review required:"
  $unexpectedDirs | ForEach-Object { W $_.Path }
} else {
  W "No unexpected active blockchain-like folders OK."
}

W ""
W "=== Go list blockchain packages ==="

Run-Capture "go list ./blockchain" "go" @("list","./blockchain")
Run-Capture "go list all packages" "go" @("list","./...")

W ""
W "=== Go test canonical blockchain package ==="

Run-Capture "go test ./blockchain" "go" @("test","./blockchain")

W ""
W "=== Root node audit build after blockchain package audit ==="

$out = Join-Path -Path $buildDir -ChildPath "quantumcoin-node-blockchain-audit.exe"
Run-Capture "go build . blockchain audit" "go" @("build","-trimpath","-o",$out,".")

if (!(Test-Path $out)) {
  throw "Audit build output missing"
}

$bh = (Get-FileHash $out -Algorithm SHA256).Hash
$bs = (Get-Item $out).Length
W ("BUILT_BLOCKCHAIN_AUDIT`tSize={0}`tSHA256={1}`t{2}" -f $bs, $bh, $out)

W ""
W "=== Release gate after audit ==="
$afterEvidence = Get-QCReleaseAuditEvidence @auditArgs
if ($afterEvidence -ne $releaseEvidence) { throw 'Signed release changed during audit.' }
W $afterEvidence

W ""
W "=== Blockchain package conclusion ==="

W "CANONICAL_BLOCKCHAIN_PACKAGE_OK"
W "NO_DUPLICATE_ACTIVE_BLOCKCHAIN_PACKAGE"
W "GO_TEST_BLOCKCHAIN_OK"
W "ROOT_BUILD_WITH_BLOCKCHAIN_OK"
W "Audit build output stayed under: $buildDir"

W ""
W "QC_08_BLOCKCHAIN_PACKAGE_AUDIT_PASS"
W "QC_08_BLOCKCHAIN_PACKAGE_AUDIT_DONE"
W "Report: $report"
