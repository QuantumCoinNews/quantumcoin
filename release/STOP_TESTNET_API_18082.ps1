$ErrorActionPreference="Continue"

"=== QC_TESTNET_API_18082_LOCAL_STOP ==="
"GIT_GITHUB_TOUCHED=NO"
"REAL_TRANSFER_SENT_BY_THIS_SCRIPT=NO"
"MAINNET_CHAIN_TOUCHED=NO"

$conns=@(Get-NetTCPConnection -LocalPort 18082 -ErrorAction SilentlyContinue | Where-Object { $_.State -eq "Listen" })
if(!$conns -or $conns.Count -eq 0){
  "NO_TESTNET_API_LISTENING_ON_18082"
}else{
  foreach($pidItem in ($conns | Select-Object -ExpandProperty OwningProcess -Unique)){
    $p=Get-Process -Id $pidItem -ErrorAction SilentlyContinue
    "STOP_CANDIDATE_PID=$pidItem PROCESS=$($p.ProcessName) PATH=$($p.Path)"
    if($p -and $p.ProcessName -eq "quantumcoin"){
      Stop-Process -Id $pidItem -Force -ErrorAction Stop
      "STOPPED_TESTNET_API_PID=$pidItem"
    }else{
      "SKIPPED_NON_QUANTUMCOIN_PID=$pidItem"
    }
  }
}

"QC_TESTNET_API_18082_LOCAL_STOP_DONE"
