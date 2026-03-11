# stop_miner.ps1
param()

Set-Location -LiteralPath 'C:\Projects\quantumcoin\release'

# Stop bayrağı
$flag = Join-Path $pwd 'miner_stop.flag'
New-Item -Path $flag -ItemType File -Force | Out-Null

# Sadece "mine" argümanlı quantumcoin.exe süreçlerini kapat
try {
  $procs = Get-CimInstance Win32_Process -Filter "Name='quantumcoin.exe'"
  foreach ($p in $procs) {
    if ($p.CommandLine -match '\s+mine(\s|$)') {
      Stop-Process -Id $p.ProcessId -Force -ErrorAction SilentlyContinue
    }
  }
} catch {}

# Loga not düş
$log = Join-Path $pwd 'miner_out.log'
if (Test-Path $log) {
  Add-Content -Path $log -Value ("Stopped by stop_miner.ps1 : {0:yyyy-MM-dd HH:mm:ss}" -f (Get-Date)) -Encoding utf8
}
