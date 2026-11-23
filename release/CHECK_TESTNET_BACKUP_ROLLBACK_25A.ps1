$ErrorActionPreference = 'Stop'
foreach ($name in @('BACKUP_TESTNET_STATE_25A.ps1', 'RESTORE_TESTNET_STATE_25A_REQUIRE_EXPLICIT_PATH.ps1')) {
  $result = & pwsh -NoProfile -File (Join-Path $PSScriptRoot $name) 2>&1
  if ($LASTEXITCODE -eq 0 -or ($result -join ' ') -notmatch 'retired') {
    throw 'Legacy copy or in-place restore path is still active.'
  }
}
'QC_TESTNET_BACKUP_ROLLBACK_25A_CHECK_RESULT=PASS'
