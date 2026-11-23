param([Parameter(Mandatory=$true)][string]$LockPath)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$lease = $null
try {
    try { $lease = [IO.File]::Open($LockPath, [IO.FileMode]::OpenOrCreate, [IO.FileAccess]::ReadWrite, [IO.FileShare]::None) }
    catch [IO.IOException] { [Console]::Out.WriteLine('BUSY'); exit 73 }
    [Console]::Out.WriteLine('LOCKED')
    while ($null -ne [Console]::In.ReadLine()) { }
} finally { if ($null -ne $lease) { $lease.Dispose() } }
