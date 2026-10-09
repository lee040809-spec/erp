@echo off
chcp 65001 >nul
cd /d "%~dp0"
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0Start-ERP.ps1" -ResetDatabasePassword -OpenBrowser
pause
