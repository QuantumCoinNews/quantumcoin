@echo off
setlocal EnableExtensions EnableDelayedExpansion
chcp 65001 >NUL

REM ============================================================
REM QuantumCoin Miner Runner (release folder)
REM - Uses ONLY: %CD%\quantumcoin.exe
REM - Live output to CMD + append to miner_out.log
REM - Enables VT so ANSI colors (green blocks) work
REM - Stops with miner_stop.flag
REM - Writes miner_pid.txt (quantumcoin.exe PID)
REM ============================================================

REM 1) Work dir = this .cmd folder
cd /d "%~dp0"
set "QC_NODE_DIR=%CD%"
set "QC_MINED_PATH=%QC_NODE_DIR%\mined_balance.json"
if not defined QC_API_BASE set "QC_API_BASE=http://127.0.0.1:8081"

REM logs in English
if not defined QC_LANG set "QC_LANG=en"
set "LANG=en_US.UTF-8"
set "LANGUAGE=en"
set "LC_ALL=C"

REM 2) Address: ENV > wallet_address.txt > miner_address.txt > fallback
if not defined ADDR (
  if exist "%QC_NODE_DIR%\wallet_address.txt" (
    for /f "usebackq delims=" %%A in ("%QC_NODE_DIR%\wallet_address.txt") do set "ADDR=%%A"
  )
)
if not defined ADDR (
  if exist "%QC_NODE_DIR%\miner_address.txt" (
    for /f "usebackq delims=" %%A in ("%QC_NODE_DIR%\miner_address.txt") do set "ADDR=%%A"
  )
)

if not defined ADDR set "ADDR=1iZ9PBE1F3gJ5VbBLpqFX1r9uEj5BnHBV"

REM trim quotes/spaces a bit
set "ADDR=%ADDR:"=%"
set "ADDR=%ADDR: =%"

REM 3) Single-address envs
if not defined QC_COMMUNITY_ADDRESS set "QC_COMMUNITY_ADDRESS=%ADDR%"
if not defined QC_DEV_FUND_ADDRESS  set "QC_DEV_FUND_ADDRESS=%ADDR%"
if not defined QC_PREMINE_ADDRESS   set "QC_PREMINE_ADDRESS=%ADDR%"

REM 4) Files
set "LOG=%QC_NODE_DIR%\miner_out.log"
set "STOPFLAG=%QC_NODE_DIR%\miner_stop.flag"
set "PIDFILE=%QC_NODE_DIR%\miner_pid.txt"
set "PS1=%QC_NODE_DIR%\_qc_miner_stream.ps1"
set "EXE=%QC_NODE_DIR%\quantumcoin.exe"

if not exist "%EXE%" (
  echo [ERROR] quantumcoin.exe not found in release folder:
  echo         "%EXE%"
  goto :done
)

if not exist "%LOG%" (
  >"%LOG%" echo.
)

REM rotate ~10MB
for %%A in ("%LOG%") do set "LOGSIZE=%%~zA"
if defined LOGSIZE (
  if !LOGSIZE! GTR 10485760 (
    if exist "%LOG%.1" del /q "%LOG%.1" 2>NUL
    ren "%LOG%" "miner_out.log.1" 2>NUL
    >"%LOG%" echo.
  )
)

del /q "%STOPFLAG%" 2>NUL

title QuantumCoin Miner

echo ===============================
echo Mining to : %ADDR%
echo Folder   : %QC_NODE_DIR%
echo API Base : %QC_API_BASE%
echo ===============================

REM 5) Choose PowerShell engine (prefer pwsh)
set "PS_EXE="
where pwsh >NUL 2>&1
if %ERRORLEVEL%==0 set "PS_EXE=pwsh"
if not defined PS_EXE (
  where powershell >NUL 2>&1
  if %ERRORLEVEL%==0 set "PS_EXE=powershell"
)

if not defined PS_EXE (
  echo [ERROR] PowerShell not found. Cannot stream+log simultaneously.
  echo         Install PowerShell or use Windows built-in powershell.exe.
  goto :done
)

call :write_ps1

:loop
if exist "%STOPFLAG%" goto :done

"%PS_EXE%" -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%PS1%" ^
  -Dir "%QC_NODE_DIR%" -Addr "%ADDR%" -Log "%LOG%" -Flag "%STOPFLAG%" -PidFile "%PIDFILE%"

timeout /t 1 /nobreak >NUL
goto :loop

:done
echo Stopped by miner_stop.flag>>"%LOG%"
endlocal
exit /b 0

REM ------------------------------------------------------------
REM Writes a PowerShell script that:
REM - Enables VT (ANSI colors)
REM - Starts quantumcoin.exe mine <addr>
REM - Streams raw bytes from stdout+stderr to console + log
REM - Writes miner_pid.txt with quantumcoin.exe PID
REM - Kills miner if miner_stop.flag appears
REM ------------------------------------------------------------
:write_ps1
> "%PS1%" echo param([string]$Dir,[string]$Addr,[string]$Log,[string]$Flag,[string]$PidFile)
>>"%PS1%" echo $ErrorActionPreference='SilentlyContinue'
>>"%PS1%" echo if([string]::IsNullOrWhiteSpace($Dir)){ $Dir=(Get-Location).Path }
>>"%PS1%" echo Set-Location $Dir ^| Out-Null
>>"%PS1%" echo if([string]::IsNullOrWhiteSpace($Log)){ $Log=(Join-Path $Dir 'miner_out.log') }
>>"%PS1%" echo if([string]::IsNullOrWhiteSpace($Flag)){ $Flag=(Join-Path $Dir 'miner_stop.flag') }
>>"%PS1%" echo if([string]::IsNullOrWhiteSpace($PidFile)){ $PidFile=(Join-Path $Dir 'miner_pid.txt') }
>>"%PS1%" echo try{ Add-Type -Namespace QC -Name VT -MemberDefinition '[DllImport("kernel32.dll")] public static extern IntPtr GetStdHandle(int n); [DllImport("kernel32.dll")] public static extern bool GetConsoleMode(IntPtr h, out int m); [DllImport("kernel32.dll")] public static extern bool SetConsoleMode(IntPtr h, int m);' -ErrorAction SilentlyContinue }catch{}
>>"%PS1%" echo try{ $h=[QC.VT]::GetStdHandle(-11); $m=0; if([QC.VT]::GetConsoleMode($h,[ref]$m)){ [QC.VT]::SetConsoleMode($h, ($m -bor 4)) ^| Out-Null } }catch{}
>>"%PS1%" echo [Console]::OutputEncoding = [Text.UTF8Encoding]::new($false)
>>"%PS1%" echo $exe = Join-Path $Dir 'quantumcoin.exe'
>>"%PS1%" echo if(-not (Test-Path $exe)){ [Console]::WriteLine('[ERROR] quantumcoin.exe not found: ' + $exe); exit 2 }
>>"%PS1%" echo $utf8 = New-Object Text.UTF8Encoding($false)
>>"%PS1%" echo $fs = New-Object IO.FileStream($Log,[IO.FileMode]::Append,[IO.FileAccess]::Write,[IO.FileShare]::ReadWrite)
>>"%PS1%" echo $out = [Console]::OpenStandardOutput()
>>"%PS1%" echo $psi = New-Object Diagnostics.ProcessStartInfo
>>"%PS1%" echo $psi.FileName = $exe
>>"%PS1%" echo $psi.Arguments = ('mine ' + $Addr)
>>"%PS1%" echo $psi.WorkingDirectory = $Dir
>>"%PS1%" echo $psi.UseShellExecute = $false
>>"%PS1%" echo $psi.RedirectStandardOutput = $true
>>"%PS1%" echo $psi.RedirectStandardError  = $true
>>"%PS1%" echo $psi.StandardOutputEncoding = $utf8
>>"%PS1%" echo $psi.StandardErrorEncoding  = $utf8
>>"%PS1%" echo $p = New-Object Diagnostics.Process
>>"%PS1%" echo $p.StartInfo = $psi
>>"%PS1%" echo if(-not $p.Start()){ $fs.Dispose(); exit 3 }
>>"%PS1%" echo [IO.File]::WriteAllText($PidFile, [string]$p.Id, $utf8)
>>"%PS1%" echo $bsOut = $p.StandardOutput.BaseStream
>>"%PS1%" echo $bsErr = $p.StandardError.BaseStream
>>"%PS1%" echo $bufOut = New-Object byte[] 8192
>>"%PS1%" echo $bufErr = New-Object byte[] 8192
>>"%PS1%" echo $tOut = $bsOut.ReadAsync($bufOut,0,$bufOut.Length)
>>"%PS1%" echo $tErr = $bsErr.ReadAsync($bufErr,0,$bufErr.Length)
>>"%PS1%" echo while($true){
>>"%PS1%" echo ^  if(Test-Path $Flag){ try{ $p.Kill() }catch{} }
>>"%PS1%" echo ^  $tasks = @()
>>"%PS1%" echo ^  if($tOut){ $tasks += $tOut }
>>"%PS1%" echo ^  if($tErr){ $tasks += $tErr }
>>"%PS1%" echo ^  if($tasks.Count -eq 0){ break }
>>"%PS1%" echo ^  $idx = [Threading.Tasks.Task]::WaitAny($tasks, 200)
>>"%PS1%" echo ^  if($idx -lt 0){ if($p.HasExited){ } ; continue }
>>"%PS1%" echo ^  $completed = $tasks[$idx]
>>"%PS1%" echo ^  if($completed -eq $tOut){
>>"%PS1%" echo ^    $n = $tOut.Result
>>"%PS1%" echo ^    if($n -gt 0){ $out.Write($bufOut,0,$n); $fs.Write($bufOut,0,$n); $fs.Flush() } else { $tOut = $null }
>>"%PS1%" echo ^    if($tOut){ $tOut = $bsOut.ReadAsync($bufOut,0,$bufOut.Length) }
>>"%PS1%" echo ^  } elseif($completed -eq $tErr){
>>"%PS1%" echo ^    $n = $tErr.Result
>>"%PS1%" echo ^    if($n -gt 0){ $out.Write($bufErr,0,$n); $fs.Write($bufErr,0,$n); $fs.Flush() } else { $tErr = $null }
>>"%PS1%" echo ^    if($tErr){ $tErr = $bsErr.ReadAsync($bufErr,0,$bufErr.Length) }
>>"%PS1%" echo ^  }
>>"%PS1%" echo }
>>"%PS1%" echo try{ $p.WaitForExit() }catch{}
>>"%PS1%" echo try{ $fs.Dispose() }catch{}
>>"%PS1%" echo exit 0
exit /b 0