@echo off
setlocal EnableExtensions
chcp 65001 >NUL
title QuantumCoin Miner
echo QuantumCoin isolated local full-block miner
pwsh -NoLogo -NoProfile -NonInteractive -File "%~dp0_qc_api_mempool_miner.ps1"
exit /b %ERRORLEVEL%
