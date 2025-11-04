@echo off
setlocal enableextensions enabledelayedexpansion
chcp 65001 >NUL

REM script her zaman kendi bulunduğu klasörde çalışsın
cd /d "%~dp0"

REM --- sabit ortamlar ---
set "QC_NODE_DIR=%CD%"
set "QC_COMMUNITY_ADDRESS=1iZ9PBE1F3gJ5VbBLpqFX1r9uEj5BnHBV"
set "QC_DEV_FUND_ADDRESS=1iZ9PBE1F3gJ5VbBLpqFX1r9uEj5BnHBV"
set "QC_PREMINE_ADDRESS=1iZ9PBE1F3gJ5VbBLpqFX1r9uEj5BnHBV"
set "QC_MINED_PATH=%CD%\mined_balance.json"
set "ADDR=1iZ9PBE1F3gJ5VbBLpqFX1r9uEj5BnHBV"
set "LOG=miner_out.log"

REM eski stop bayrağını temizle
del /q "miner_stop.flag" 2>NUL

echo ===============================
echo Mining to: %ADDR%
echo Folder    : %CD%
echo ===============================

REM PowerShell var mı?
where powershell >NUL 2>&1
if %ERRORLEVEL% EQU 0 (
    set "HAS_PS=1"
) else (
    set "HAS_PS=0"
)

:loop
if exist "miner_stop.flag" goto :done

if "%HAS_PS%"=="1" (
    powershell -NoLogo -NoProfile -ExecutionPolicy Bypass -Command ^
      "& { & '.\quantumcoin.exe' mine '%ADDR%' 2>&1 | Tee-Object -File '%LOG%' -Append }"
) else (
    ".\quantumcoin.exe" mine "%ADDR%" >> "%LOG%" 2>&1
)

REM ufak nefes
timeout /t 1 /nobreak >NUL
goto :loop

:done
echo Stopped by miner_stop.flag>>"%LOG%"
endlocal
