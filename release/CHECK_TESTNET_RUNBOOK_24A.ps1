$ErrorActionPreference = 'Stop'
if (-not (Test-Path -LiteralPath (Join-Path $PSScriptRoot 'TESTNET_RUNBOOK_24A.md') -PathType Leaf)) {
  throw 'Testnet runbook is missing.'
}
& (Join-Path $PSScriptRoot 'CHECK_TESTNET_BOUNDARY_23B.ps1')
& (Join-Path $PSScriptRoot 'CHECK_TESTNET_BACKUP_ROLLBACK_25A.ps1')
'QC_TESTNET_RUNBOOK_24A_CHECK_RESULT=PASS'
