# run_miner.ps1
param()

Set-Location -LiteralPath 'C:\Projects\quantumcoin\release'

# === Env ===
$env:QC_NODE_DIR         = (Get-Location).Path
$env:QC_COMMUNITY_ADDRESS= '1iZ9PBE1F3gJ5VbBLpqFX1r9uEj5BnHBV'
$env:QC_DEV_FUND_ADDRESS = '1iZ9PBE1F3gJ5VbBLpqFX1r9uEj5BnHBV'
$env:QC_PREMINE_ADDRESS  = '1iZ9PBE1F3gJ5VbBLpqFX1r9uEj5BnHBV'
$env:QC_MINED_PATH       = Join-Path $pwd 'mined_balance.json'
$env:QC_LANG             = 'en'
$env:LANG                = 'en_US.UTF-8'
$env:LANGUAGE            = 'en'
$env:LC_ALL              = 'C'
$env:ADDR                = '1iZ9PBE1F3gJ5VbBLpqFX1r9uEj5BnHBV'

$log  = Join-Path $pwd 'miner_out.log'
$flag = Join-Path $pwd 'miner_stop.flag'

# Eski stop bayrağını temizle
Remove-Item -LiteralPath $flag -ErrorAction SilentlyContinue

# Log rotasyonu (~10MB)
if (Test-Path $log) {
  $fi = Get-Item $log -ErrorAction SilentlyContinue
  if ($fi.Length -gt 10MB) {
    $bak = "$log.1"
    Remove-Item -LiteralPath $bak -ErrorAction SilentlyContinue
    Rename-Item -LiteralPath $log -NewName (Split-Path $bak -Leaf)
  }
}

# Logu UTF-8 BOM ile başlat (yoksa)
if (-not (Test-Path $log)) {
  Set-Content -Path $log -Value '' -Encoding utf8BOM -NoNewline
}

# Başlık
Add-Content -Path $log -Value "===============================" -Encoding utf8
Add-Content -Path $log -Value "Mining to: $($env:ADDR)"         -Encoding utf8
Add-Content -Path $log -Value "Folder    : $pwd"                -Encoding utf8
Add-Content -Path $log -Value ("Started   : {0:yyyy-MM-dd HH:mm:ss}" -f (Get-Date)) -Encoding utf8
Add-Content -Path $log -Value "===============================" -Encoding utf8

# Döngü (crash sonrası yeniden başlatır)
while (-not (Test-Path $flag)) {
  try {
    & "$pwd\quantumcoin.exe" mine $env:ADDR 2>&1 |
      Tee-Object -File $log -Append -Encoding utf8
  } catch {
    Add-Content -Path $log -Value ("ERROR: " + $_) -Encoding utf8
  }

  if (Test-Path $flag) { break }
  Start-Sleep -Seconds 1
}

Add-Content -Path $log -Value ("Stopped   : {0:yyyy-MM-dd HH:mm:ss}" -f (Get-Date)) -Encoding utf8
