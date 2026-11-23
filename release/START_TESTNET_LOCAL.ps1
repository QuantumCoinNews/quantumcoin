param(
  [Parameter(Mandatory)]
  [ValidateSet('api', 'run')]
  [string]$Mode,
  [string]$DataDir,
  [string]$ReleaseDirectory,
  [string]$ReleaseCommit,
  [string]$TrustedSigners,
  [switch]$PlanOnly
)

$ErrorActionPreference = 'Stop'
$repo = Split-Path -Parent $PSScriptRoot
$config = Join-Path $PSScriptRoot 'config.testnet.json'
$workspaceTool = Join-Path $repo 'scripts/network_workspaces.py'
$workspaceArgs = @($workspaceTool, 'plan', '--network', 'testnet')
if ($DataDir) { $workspaceArgs += @('--directory', $DataDir) }
if ($ReleaseDirectory) { $workspaceArgs += @('--package', $ReleaseDirectory) }
$workspaceJSON = & python @workspaceArgs
if ($LASTEXITCODE -ne 0) { throw 'Testnet workspace preflight failed; original data preserved.' }
$workspace = $workspaceJSON | ConvertFrom-Json
$data = $workspace.dataDir
$profile = Get-Content -LiteralPath $config -Raw | ConvertFrom-Json
if ($profile.genesis_profile -cne 'network-v4') { throw 'Fresh testnet requires network-v4.' }
if ($profile.chain_file -cne 'chain_data_testnet.dat') { throw 'Expected the canonical testnet chain filename.' }
foreach ($name in @('chain_file', 'bonus_file', 'wallet_file')) {
  if ($profile.$name -cnotmatch '^[a-z0-9_]+_testnet\.(dat|json)$') {
    throw 'Testnet storage paths must be testnet filenames inside the data directory.'
  }
}
if ($profile.p2p_port -cnotmatch '^:[0-9]{1,5}$') { throw 'Testnet P2P port is invalid.' }
$p2pListen = '127.0.0.1' + $profile.p2p_port
if ($PlanOnly) {
  [ordered]@{ mode=$Mode; config=$config; dataDir=$data; chain=$profile.chain_file; genesisProfile=$profile.genesis_profile; balanceMigration=$false; prepared=$workspace.prepared; httpPort=$profile.http_port; p2pListen=$p2pListen } | ConvertTo-Json -Compress
  return
}
if (-not ($ReleaseDirectory -and $ReleaseCommit -and $TrustedSigners)) {
  throw 'Supply a verified release directory, approved commit and independent trusted signers.'
}
if ([string]::IsNullOrWhiteSpace($env:QC_API_TOKEN)) {
  throw 'Set QC_API_TOKEN in the process environment before starting the local testnet.'
}
$package = (Resolve-Path -LiteralPath $ReleaseDirectory).ProviderPath
if ($data -eq $package -or $data.StartsWith($package+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase)) {
  throw 'Testnet runtime must be outside the signed package.'
}
& python (Join-Path $repo 'tools/qc_release.py') verify --directory $package --commit $ReleaseCommit --trusted-signers $TrustedSigners
if ($LASTEXITCODE -ne 0) { throw 'Signed release verification failed.' }
& python $workspaceTool prepare --network testnet --directory $data --package $package | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Fresh testnet workspace preparation failed; node not started.' }

$start = [Diagnostics.ProcessStartInfo]::new()
$start.FileName = Join-Path $package 'quantumcoin.exe'
$start.WorkingDirectory = $data
$start.UseShellExecute = $false
foreach ($argument in @('--config', $config, '--data-dir', $data, $Mode)) { $start.ArgumentList.Add($argument) }
if ($Mode -eq 'run') { $start.ArgumentList.Add($p2pListen) }
foreach ($key in @($start.Environment.Keys)) {
  if (($key -like 'QC_*' -and $key -ne 'QC_API_TOKEN') -or $key -eq 'HTTP_PORT') {
    [void]$start.Environment.Remove($key)
  }
}
$start.Environment['QC_TESTNET'] = '1'
$start.Environment['QC_NETWORK'] = 'testnet-local'
$start.Environment['QC_NET_MAGIC'] = 'QC_TESTNET_V1'
$child = [Diagnostics.Process]::Start($start)
try {
  $child.WaitForExit()
  if ($child.ExitCode -ne 0) { throw "Testnet node exited with code $($child.ExitCode)." }
} finally {
  $child.Dispose()
}
