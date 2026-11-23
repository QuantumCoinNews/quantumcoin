@echo off
setlocal EnableExtensions
call "%~dp0release\run_miner.cmd"
exit /b %ERRORLEVEL%
