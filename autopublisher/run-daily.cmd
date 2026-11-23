@echo off
pwsh.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0run-daily.ps1" %*
exit /b %errorlevel%
