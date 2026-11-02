$ErrorActionPreference="Continue"

"=== QC_TESTNET_NODE_18082_13001_LOCAL_STOP ==="
"GIT_GITHUB_TOUCHED=NO"
"REAL_TRANSFER_SENT_BY_THIS_SCRIPT=NO"
"MAINNET_CHAIN_TOUCHED=NO"

$pidsToStop=@()

foreach($port in @(18082,13001)){
  $conns=@(Get-NetTCPConnection -LocalPort $port -ErrorAction SilentlyContinue | Where-Object { $_.State -eq "Listen" })
  foreach($c in $conns){
    if($pidsToStop -notcontains $c.OwningProcess){
      $pidsToStop += $c.OwningProcess
    }
  }
}

if($pidsToStop.Count -eq 0){
  "NO_TESTNET_NODE_LISTENING_ON_18082_OR_13001"
}else{
  foreach($pidItem in $pidsToStop){
    $p=Get-Process -Id $pidItem -ErrorAction SilentlyContinue
    "STOP_CANDIDATE_PID=$pidItem PROCESS=$($p.ProcessName) PATH=$($p.Path)"
    if($p -and $p.ProcessName -eq "quantumcoin"){
      Stop-Process -Id $pidItem -Force -ErrorAction Stop
      "STOPPED_TESTNET_NODE_PID=$pidItem"
    }else{
      "SKIPPED_NON_QUANTUMCOIN_PID=$pidItem"
    }
  }
}

Start-Sleep -Seconds 2

foreach($port in @(18082,13001)){
  $left=@(Get-NetTCPConnection -LocalPort $port -ErrorAction SilentlyContinue | Where-Object { $_.State -eq "Listen" })
  "PORT_$port`_LISTEN_AFTER_STOP=$($left.Count)"
}

"QC_TESTNET_NODE_18082_13001_LOCAL_STOP_DONE"
