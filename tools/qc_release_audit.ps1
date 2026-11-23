# Shared release gate for all source audits. The manifest verifier owns policy.
function Get-QCReleaseAuditEvidence {
  param(
    [switch]$SourceOnly,
    [string]$ReleaseDirectory,
    [string]$ReleaseCommit,
    [string]$TrustedSigners
  )

  if ($SourceOnly) {
    if ($ReleaseDirectory -or $ReleaseCommit -or $TrustedSigners) {
      throw 'Choose source-only audit or signed release verification.'
    }
    return 'SOURCE_ONLY: release provenance was not evaluated.'
  }
  if (-not ($ReleaseDirectory -and $ReleaseCommit -and $TrustedSigners)) {
    throw 'Supply -SourceOnly or all of -ReleaseDirectory, -ReleaseCommit and -TrustedSigners.'
  }
  if (-not ([IO.Path]::IsPathFullyQualified($ReleaseDirectory) -and [IO.Path]::IsPathFullyQualified($TrustedSigners))) {
    throw 'ReleaseDirectory and TrustedSigners must be absolute paths.'
  }

  $repo = Split-Path -Parent $PSScriptRoot
  $headCommit = & git -C $repo rev-parse --verify HEAD
  if ($LASTEXITCODE -ne 0 -or $headCommit -cne $ReleaseCommit) {
    throw 'The audit checkout must match the authorized release commit.'
  }
  $changes = & git -C $repo status --porcelain --untracked-files=normal
  if ($LASTEXITCODE -ne 0 -or $changes) {
    throw 'Release audits require a clean source checkout.'
  }

  $verifier = Join-Path $PSScriptRoot 'qc_release.py'
  $proof = & python $verifier verify --directory $ReleaseDirectory --commit $ReleaseCommit --trusted-signers $TrustedSigners
  if ($LASTEXITCODE -ne 0) {
    throw 'Signed release verification failed.'
  }
  return $proof
}
