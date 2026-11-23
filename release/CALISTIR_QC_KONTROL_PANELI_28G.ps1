$ErrorActionPreference = 'Stop'
Push-Location (Split-Path -Parent $PSScriptRoot)
try {
  & go run ./release/qc_kontrol_paneli
  if ($LASTEXITCODE -ne 0) { throw 'Canli panel baslatilamadi.' }
} finally {
  Pop-Location
}
