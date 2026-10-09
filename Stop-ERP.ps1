$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'scripts\ERP-Runtime.ps1')
try {
    $erpScheduler = New-Object -ComObject 'Schedule.Service'
    $erpScheduler.Connect()
    $erpTask = Get-ErpScheduledTask ($erpScheduler.GetFolder('\'))
    if ($erpTask -and $erpTask.State -eq 4) {
        Set-Content -LiteralPath $erpStopRequestPath -Value (Get-Date -Format o)
        $erpStopDeadline = [DateTime]::UtcNow.AddSeconds(15)
        do {
            Start-Sleep -Milliseconds 500
            $erpTask = Get-ErpScheduledTask ($erpScheduler.GetFolder('\'))
        } while ($erpTask.State -eq 4 -and [DateTime]::UtcNow -lt $erpStopDeadline)
        if ($erpTask.State -eq 4) { $erpTask.Stop(0) }
    }
    if (Test-Path -LiteralPath $erpProcessManifestPath) {
        $erpManifest = Get-Content -LiteralPath $erpProcessManifestPath -Raw | ConvertFrom-Json
        Stop-ErpTrackedProcess $erpManifest.backend (Join-Path $erpBackendRoot 'bin\erp-server-memory.exe')
        $erpNodeCommand = Get-Command node.exe -ErrorAction SilentlyContinue
        $erpExpectedNodePath = if ($erpNodeCommand) { $erpNodeCommand.Source } else { Join-Path $env:ProgramFiles 'nodejs\node.exe' }
        Stop-ErpTrackedProcess $erpManifest.frontend $erpExpectedNodePath
    }
    if ((Test-ErpPort 3000) -or (Test-ErpPort 28080)) { throw '端口仍被占用，可能存在其他启动方式的服务；未强行停止其他程序。' }
    Write-Host 'ERP 已停止。下次双击 start.bat 即可启动。' -ForegroundColor Green
    exit 0
} catch { Write-Host ('停止失败：' + $_.Exception.Message) -ForegroundColor Red; exit 1 }
