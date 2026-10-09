param([switch]$OpenBrowser, [switch]$ResetDatabasePassword)
& (Join-Path $PSScriptRoot 'scripts\Start-ERP-Independent.ps1') -OpenBrowser:$OpenBrowser -ResetDatabasePassword:$ResetDatabasePassword
exit $LASTEXITCODE