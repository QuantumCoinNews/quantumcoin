param([Parameter(Mandatory=$true)][string]$LockPath)
$ErrorActionPreference='Stop'
$lease=$null
try {
 try{$lease=[IO.File]::Open($LockPath,[IO.FileMode]::OpenOrCreate,[IO.FileAccess]::ReadWrite,[IO.FileShare]::ReadWrite)}
 catch [IO.IOException]{[Console]::Out.WriteLine('BUSY');exit 73}
 [Console]::Out.WriteLine('LOCKED')
 while($null -ne [Console]::In.ReadLine()){}
} finally {if($null -ne $lease){$lease.Dispose()}}
