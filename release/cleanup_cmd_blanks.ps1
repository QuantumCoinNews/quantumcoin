param(
  [string]$FlagPath = ".\miner_stop.flag",
  [int]$IntervalMs = 600
)

# "QuantumCoin Miner" dışındaki boş cmd.exe pencerelerini kapat
while (-not (Test-Path -LiteralPath $FlagPath)) {
  try {
    $procs = Get-Process cmd -ErrorAction SilentlyContinue | Where-Object {
      $_.MainWindowTitle `
        -and $_.MainWindowTitle -ne 'QuantumCoin Miner' `
        -and ($_.MainWindowTitle -like '*\System32\cmd.exe*' -or $_.MainWindowTitle -eq '')
    }
    foreach ($p in $procs) {
      try { Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue } catch {}
    }
  } catch {}
  Start-Sleep -Milliseconds $IntervalMs
}
