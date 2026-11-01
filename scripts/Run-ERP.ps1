$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'ERP-Runtime.ps1')
Initialize-ErpRuntime
$erpBackendProcess = $null
$erpFrontendProcess = $null
$erpResultCode = 0
try {
    Write-ErpRuntimeLog 'Starting Windows-managed ERP.'
    Remove-Item -LiteralPath $erpStopRequestPath -Force -ErrorAction SilentlyContinue
    if ((Test-ErpPort 28080) -or (Test-ErpPort 3000)) { throw 'ERP ports are occupied; refusing to start duplicate services.' }
    $erpBackendExe = Join-Path $erpBackendRoot 'bin\erp-server-memory.exe'
    $erpViteScript = Join-Path $erpFrontendRoot 'node_modules\vite\bin\vite.js'
    if (-not (Test-Path -LiteralPath $erpBackendExe)) { throw 'Backend executable is missing.' }
    if (-not (Test-Path -LiteralPath $erpViteScript)) { throw 'Frontend dependencies are missing.' }
    $erpNodeCommand = Get-Command node.exe -ErrorAction SilentlyContinue
    $erpNodePath = if ($erpNodeCommand) { $erpNodeCommand.Source } else { Join-Path $env:ProgramFiles 'nodejs\node.exe' }
    if (-not (Test-Path -LiteralPath $erpNodePath)) { throw 'Node.js is missing.' }
    $erpRunStamp = Get-Date -Format 'yyyyMMdd-HHmmss'
    $env:GO_ENV = 'local'
    $env:ERP_BACKUP_STATUS_FILE = Join-Path $erpRuntimeRoot 'backup-status.json'
    $erpBackendProcess = Start-Process -FilePath $erpBackendExe -WorkingDirectory $erpBackendRoot -WindowStyle Hidden -PassThru -RedirectStandardOutput (Join-Path $erpRuntimeRoot "backend-$erpRunStamp.log") -RedirectStandardError (Join-Path $erpRuntimeRoot "backend-$erpRunStamp.err.log")
    Set-Content -LiteralPath (Join-Path $erpRuntimeRoot 'backend.pid') -Value $erpBackendProcess.Id
    $erpFrontendProcess = Start-Process -FilePath $erpNodePath -ArgumentList ('"' + $erpViteScript + '" --host 127.0.0.1 --port 3000 --strictPort') -WorkingDirectory $erpFrontendRoot -WindowStyle Hidden -PassThru -RedirectStandardOutput (Join-Path $erpRuntimeRoot "frontend-$erpRunStamp.log") -RedirectStandardError (Join-Path $erpRuntimeRoot "frontend-$erpRunStamp.err.log")
    Set-Content -LiteralPath (Join-Path $erpRuntimeRoot 'frontend.pid') -Value $erpFrontendProcess.Id
    $erpProcessManifest = @{
        backend = @{ pid=$erpBackendProcess.Id; start_ticks=$erpBackendProcess.StartTime.ToUniversalTime().Ticks.ToString() }
        frontend = @{ pid=$erpFrontendProcess.Id; start_ticks=$erpFrontendProcess.StartTime.ToUniversalTime().Ticks.ToString(); executable=$erpNodePath }
    }
    $erpProcessManifest | ConvertTo-Json -Depth 3 | Set-Content -LiteralPath $erpProcessManifestPath -Encoding UTF8
    Write-ErpRuntimeLog ('Children started: backend=' + $erpBackendProcess.Id + ', frontend=' + $erpFrontendProcess.Id)
    # Keep the task alive; Windows, not Codex, owns this process tree.
    while ($true) {
        if (Test-Path -LiteralPath $erpStopRequestPath) { Write-ErpRuntimeLog 'Stop requested by user.'; break }
        $erpBackendProcess.Refresh()
        $erpFrontendProcess.Refresh()
        if ($erpBackendProcess.HasExited) { throw ('Backend exited (code ' + $erpBackendProcess.ExitCode + '). See backend-' + $erpRunStamp + '.err.log; Windows will retry the task up to three times.') }
        if ($erpFrontendProcess.HasExited) { throw ('Frontend exited (code ' + $erpFrontendProcess.ExitCode + '). See frontend-' + $erpRunStamp + '.err.log; Windows will retry the task up to three times.') }
        Start-Sleep -Seconds 5
    }
} catch {
    Write-ErpRuntimeLog ('Failed: ' + $_.Exception.Message)
    $erpResultCode = 1
} finally {
    # Stop only handles this runner created, never arbitrary PIDs.
    foreach ($erpChildProcess in @($erpFrontendProcess, $erpBackendProcess)) {
        if ($erpChildProcess) {
            $erpChildProcess.Refresh()
            if (-not $erpChildProcess.HasExited) {
                & "$env:SystemRoot\System32\taskkill.exe" /PID $erpChildProcess.Id /T /F 2>&1 | Out-Null
            }
            $erpChildProcess.Dispose()
        }
    }
}
exit $erpResultCode
