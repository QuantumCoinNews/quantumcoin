@echo off
setlocal EnableExtensions
chcp 65001 >NUL
cd /d "%~dp0"
title QuantumCoin Miner

set "QC_NODE_DIR=%CD%"
set "QC_API_BASE=http://127.0.0.1:8082"
set "QC_HTTP_PORT=:8082"
set "QC_DIFFICULTY_BITS=12"
set "QC_MINING_DELAY_MS=5000"

if not defined ADDR (
  if exist "%CD%\miner_address.txt" (
    for /f "usebackq delims=" %%A in ("%CD%\miner_address.txt") do set "ADDR=%%A"
  )
)

if not defined ADDR (
  if exist "%CD%\wallet_address.txt" (
    for /f "usebackq delims=" %%A in ("%CD%\wallet_address.txt") do set "ADDR=%%A"
  )
)

set "ADDR=%ADDR:"=%"
set "ADDR=%ADDR: =%"

echo ===============================
echo QuantumCoin Stable API-Mempool Miner
echo Runtime    : %CD%
echo API        : %QC_API_BASE%
echo Address    : %ADDR%
echo Difficulty : %QC_DIFFICULTY_BITS%
echo Delay      : %QC_MINING_DELAY_MS% ms
echo Mode       : visible CMD -> API /api/mine loop
echo Chain      : %CD%\chain_data.dat
echo ===============================
echo.

powershell -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%CD%\_qc_api_mempool_miner.ps1" -ApiBase "%QC_API_BASE%" -Address "%ADDR%" -ReleaseDir "%CD%" -DelayMs 5000 -DifficultyBits 12

echo.
echo [QuantumCoin] Miner ended.
pause
endlocal
