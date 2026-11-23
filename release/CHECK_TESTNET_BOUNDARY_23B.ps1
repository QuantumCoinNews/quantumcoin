$ErrorActionPreference = 'Stop'
$observer = & (Join-Path $PSScriptRoot 'RUN_TESTNET_API_18082.ps1') -PlanOnly | ConvertFrom-Json
$node = & (Join-Path $PSScriptRoot 'RUN_TESTNET_NODE_18082_13001.ps1') -PlanOnly | ConvertFrom-Json
if ($observer.mode -ne 'api' -or $node.mode -ne 'run' -or $observer.config -ne $node.config -or
    $node.chain -ne 'chain_data_testnet.dat' -or $node.httpPort -ne ':18082' -or $node.p2pListen -ne '127.0.0.1:13001') {
  throw 'Testnet launch plans diverged.'
}
'QC_TESTNET_BOUNDARY_23B_CHECK_RESULT=PASS'
