param([string]$Dir,[string]$Addr,[string]$Log,[string]$Flag,[string]$PidFile)
$ErrorActionPreference='SilentlyContinue'
if([string]::IsNullOrWhiteSpace($Dir)){ $Dir=(Get-Location).Path }
Set-Location $Dir | Out-Null
if([string]::IsNullOrWhiteSpace($Log)){ $Log=(Join-Path $Dir 'miner_out.log') }
if([string]::IsNullOrWhiteSpace($Flag)){ $Flag=(Join-Path $Dir 'miner_stop.flag') }
if([string]::IsNullOrWhiteSpace($PidFile)){ $PidFile=(Join-Path $Dir 'miner_pid.txt') }
try{ Add-Type -Namespace QC -Name VT -MemberDefinition '[DllImport("kernel32.dll")] public static extern IntPtr GetStdHandle(int n); [DllImport("kernel32.dll")] public static extern bool GetConsoleMode(IntPtr h, out int m); [DllImport("kernel32.dll")] public static extern bool SetConsoleMode(IntPtr h, int m);' -ErrorAction SilentlyContinue }catch{}
try{ $h=[QC.VT]::GetStdHandle(-11); $m=0; if([QC.VT]::GetConsoleMode($h,[ref]$m)){ [QC.VT]::SetConsoleMode($h, ($m -bor 4)) | Out-Null } }catch{}
[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false)
$exe = Join-Path $Dir 'quantumcoin.exe'
if(-not (Test-Path $exe)){ [Console]::WriteLine('[ERROR] quantumcoin.exe not found: ' + $exe); exit 2 }
$utf8 = New-Object Text.UTF8Encoding($false)
$fs = New-Object IO.FileStream($Log,[IO.FileMode]::Append,[IO.FileAccess]::Write,[IO.FileShare]::ReadWrite)
$out = [Console]::OpenStandardOutput()
$psi = New-Object Diagnostics.ProcessStartInfo
$psi.FileName = $exe
$psi.Arguments = ('mine ' + $Addr)
$psi.WorkingDirectory = $Dir
$psi.UseShellExecute = $false
$psi.RedirectStandardOutput = $true
$psi.RedirectStandardError  = $true
$psi.StandardOutputEncoding = $utf8
$psi.StandardErrorEncoding  = $utf8
$p = New-Object Diagnostics.Process
$p.StartInfo = $psi
if(-not $p.Start()){ $fs.Dispose(); exit 3 }
[IO.File]::WriteAllText($PidFile, [string]$p.Id, $utf8)
$bsOut = $p.StandardOutput.BaseStream
$bsErr = $p.StandardError.BaseStream
$bufOut = New-Object byte[] 8192
$bufErr = New-Object byte[] 8192
$tOut = $bsOut.ReadAsync($bufOut,0,$bufOut.Length)
$tErr = $bsErr.ReadAsync($bufErr,0,$bufErr.Length)
while($true){
  if(Test-Path $Flag){ try{ $p.Kill() }catch{} }
  $tasks = @()
  if($tOut){ $tasks += $tOut }
  if($tErr){ $tasks += $tErr }
  if($tasks.Count -eq 0){ break }
  $idx = [Threading.Tasks.Task]::WaitAny($tasks, 200)
  if($idx -lt 0){ if($p.HasExited){ } ; continue }
  $completed = $tasks[$idx]
  if($completed -eq $tOut){
    $n = $tOut.Result
    if($n -gt 0){ $out.Write($bufOut,0,$n); $fs.Write($bufOut,0,$n); $fs.Flush() } else { $tOut = $null }
    if($tOut){ $tOut = $bsOut.ReadAsync($bufOut,0,$bufOut.Length) }
  } elseif($completed -eq $tErr){
    $n = $tErr.Result
    if($n -gt 0){ $out.Write($bufErr,0,$n); $fs.Write($bufErr,0,$n); $fs.Flush() } else { $tErr = $null }
    if($tErr){ $tErr = $bsErr.ReadAsync($bufErr,0,$bufErr.Length) }
  }
}
try{ $p.WaitForExit() }catch{}
try{ $fs.Dispose() }catch{}
exit 0
