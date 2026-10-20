@echo off
chcp 65001 >nul
title ERP Startup
cd /d "%~dp0"
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0Start-ERP.ps1" -OpenBrowser
if errorlevel 1 (
  echo.
  echo ERP startup failed. Please read the message above.
  pause
  exit /b 1
)
timeout /t 3 /nobreak >nul 2>nul
exit /b 0
