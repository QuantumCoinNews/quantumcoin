$ErrorActionPreference = 'Stop'
Push-Location (Split-Path -Parent $PSScriptRoot)
try {
  & go test ./release/qc_kontrol_paneli
  if ($LASTEXITCODE -ne 0) { throw 'Live panel checks failed.' }
  'QC_KONTROL_PANELI_28G_CHECK_RESULT=PASS'
} finally {
  Pop-Location
}
