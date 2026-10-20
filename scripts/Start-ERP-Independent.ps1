param([switch]$OpenBrowser, [switch]$ResetDatabasePassword)
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
. (Join-Path $PSScriptRoot 'ERP-Runtime.ps1')
try {
    Initialize-ErpRuntime
    Write-Host '正在检查 ERP 服务，请稍候……' -ForegroundColor Cyan
    $erpScheduler = New-Object -ComObject 'Schedule.Service'
    $erpScheduler.Connect()
    $erpTaskFolder = $erpScheduler.GetFolder('\')
    $erpExistingTask = Get-ErpScheduledTask $erpTaskFolder
    if ($ResetDatabasePassword -and $erpExistingTask -and $erpExistingTask.State -eq 4) {
        throw '请先运行 stop.bat 停止 ERP，再重新设置数据库密码。'
    }
    if ($ResetDatabasePassword -or -not (Test-Path -LiteralPath $erpCredentialPath)) {
        Write-Host '首次启动需要数据库密码（不是 ERP 登录密码）。' -ForegroundColor Yellow
        Write-Host '输入时不会显示字符或星号，这是正常现象；输入完成后请按回车。'
        Write-Host '密码将使用 Windows 用户加密保存，后续启动不再询问。'
        $erpSecurePassword = Read-Host '请输入数据库密码' -AsSecureString
        try {
            if ($erpSecurePassword.Length -eq 0) { throw '数据库密码不能为空。' }
            $erpSecurePassword | Export-Clixml -LiteralPath $erpCredentialPath
            $erpCredentialAcl = New-Object System.Security.AccessControl.FileSecurity
            $erpCredentialAcl.SetAccessRuleProtection($true, $false)
            foreach ($erpAllowedSid in @($erpUserSid, 'S-1-5-18')) {
                $erpSidObject = New-Object System.Security.Principal.SecurityIdentifier($erpAllowedSid)
                $erpRule = New-Object System.Security.AccessControl.FileSystemAccessRule($erpSidObject, 'FullControl', 'Allow')
                $erpCredentialAcl.AddAccessRule($erpRule)
            }
            Set-Acl -LiteralPath $erpCredentialPath -AclObject $erpCredentialAcl
        } finally { $erpSecurePassword.Dispose() }
    }
    if ($erpExistingTask -and $erpExistingTask.State -eq 4) {
        Write-Host 'Windows 已在运行 ERP，正在确认服务状态……'
    } else {
        if ((Test-ErpPort 28080) -or (Test-ErpPort 3000)) {
            if ((Test-ErpEndpoint $erpBackendStatusUrl -AuthStatus) -and (Test-ErpEndpoint $erpUrl)) {
                Write-Host '检测到已有 ERP。请先关闭原启动方式，再使用新的独立启动。' -ForegroundColor Yellow
                if ($OpenBrowser) { Start-Process $erpUrl }
                exit 0
            }
            throw '3000 或 28080 端口被其他进程占用，请先停止原服务。'
        }
        $erpDefinition = $erpScheduler.NewTask(0)
        $erpDefinition.RegistrationInfo.Description = '机加 ERP 本地后台服务；手动启动，不随 Codex 关闭而退出。'
        $erpDefinition.Principal.UserId = $erpUserSid
        $erpDefinition.Principal.LogonType = 3
        $erpDefinition.Principal.RunLevel = 0
        $erpDefinition.Settings.Enabled = $true
        $erpDefinition.Settings.Hidden = $true
        $erpDefinition.Settings.AllowDemandStart = $true
        $erpDefinition.Settings.DisallowStartIfOnBatteries = $false
        $erpDefinition.Settings.StopIfGoingOnBatteries = $false
        $erpDefinition.Settings.ExecutionTimeLimit = 'PT0S'
        $erpDefinition.Settings.MultipleInstances = 2
        $erpDefinition.Settings.RestartCount = 3
        $erpDefinition.Settings.RestartInterval = 'PT1M'
        $erpAction = $erpDefinition.Actions.Create(0)
        $erpAction.Path = Join-Path $env:SystemRoot 'System32\WindowsPowerShell\v1.0\powershell.exe'
        $erpAction.Arguments = '-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File "' + (Join-Path $erpRoot 'scripts\Run-ERP.ps1') + '"'
        $erpAction.WorkingDirectory = $erpRoot
        $erpTask = $erpTaskFolder.RegisterTaskDefinition($erpTaskName, $erpDefinition, 6, $erpUserSid, $null, 3, $null)
        $null = $erpTask.Run($null)
        Write-Host '已交给 Windows 独立启动，正在连接数据库并加载页面……'
    }
    $erpDeadline = [DateTime]::UtcNow.AddSeconds(125)
    do {
        if ((Test-ErpEndpoint $erpBackendStatusUrl -AuthStatus) -and
            (Test-ErpEndpoint $erpUrl) -and (Test-ErpEndpoint $erpProxyStatusUrl -AuthStatus)) {
            Write-Host ('ERP 已启动：' + $erpUrl) -ForegroundColor Green
            Write-Host '可以关闭此窗口和 Codex，ERP 会继续运行。'
            Write-Host '电脑重启或退出 Windows 后，再双击 start.bat 即可。'
            if ($OpenBrowser) { Start-Process $erpUrl }
            exit 0
        }
        $erpCurrentTask = Get-ErpScheduledTask $erpTaskFolder
        if ($erpCurrentTask -and $erpCurrentTask.State -eq 3 -and $erpCurrentTask.LastTaskResult -ne 0) {
            throw ('后台启动失败。' + [Environment]::NewLine + (Get-ErpStartupFailureDetails))
        }
        Start-Sleep -Milliseconds 500
    } while ([DateTime]::UtcNow -lt $erpDeadline)
    throw ('启动等待超时，请检查数据库网络连接。' + [Environment]::NewLine + (Get-ErpStartupFailureDetails))
} catch {
    Write-Host ('启动失败：' + $_.Exception.Message) -ForegroundColor Red
    exit 1
}
