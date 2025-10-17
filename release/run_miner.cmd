@echo off
setlocal
chcp 65001 >NUL
cd /d "C:\Projects\quantumcoin\release"
set "QC_NODE_DIR=C:\Projects\quantumcoin\release"
set "QC_COMMUNITY_ADDRESS=1iZ9PBE1F3gJ5VbBLpqFX1r9uEj5BnHBV"
set "QC_DEV_FUND_ADDRESS=1iZ9PBE1F3gJ5VbBLpqFX1r9uEj5BnHBV"
set "QC_PREMINE_ADDRESS=1iZ9PBE1F3gJ5VbBLpqFX1r9uEj5BnHBV"
set "QC_MINED_PATH=%CD%\mined_balance.json"
set "ADDR=1iZ9PBE1F3gJ5VbBLpqFX1r9uEj5BnHBV"

REM PowerShell var mı? (tee için)
where powershell >NUL 2>&1
if %ERRORLEVEL% EQU 0 (set "HAS_PS=1") else (set "HAS_PS=0")

REM Eski stop bayrağını temizle
del /q "miner_stop.flag" 2>NUL

echo ===============================
echo Mining to: %ADDR%
echo Folder    : %CD%
echo ===============================

:loop
if exist "miner_stop.flag" goto :done

if "%HAS_PS%"=="1" (
  powershell -NoLogo -ExecutionPolicy Bypass -Command ^
    "& { & '.\\quantumcoin.exe' mine "1iZ9PBE1F3gJ5VbBLpqFX1r9uEj5BnHBV" 2>&1 | Tee-Object -File 'miner_out.log' -Append }"
) else (
  ".\\quantumcoin.exe" mine "1iZ9PBE1F3gJ5VbBLpqFX1r9uEj5BnHBV" >> "miner_out.log" 2>&1
)

REM Kısa nefes
timeout /t 1 /nobreak >NUL
goto :loop

:done
echo Stopped by miner_stop.flag
