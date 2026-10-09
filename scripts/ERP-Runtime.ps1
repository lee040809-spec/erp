# Shared definitions. Never print credentials.
$erpRoot = Split-Path $PSScriptRoot -Parent
$erpRuntimeRoot = Join-Path $erpRoot '.runtime'
$erpCredentialPath = Join-Path $erpRuntimeRoot 'database-password.xml'
$erpProcessManifestPath = Join-Path $erpRuntimeRoot 'services.json'
$erpStopRequestPath = Join-Path $erpRuntimeRoot 'stop.request'
$erpBackendRoot = Join-Path $erpRoot 'platform\go-server'
$erpFrontendRoot = Join-Path $erpRoot 'platform\front'
$erpBackendStatusUrl = 'http://127.0.0.1:28080/api/platform/auth/status'
$erpProxyStatusUrl = 'http://127.0.0.1:3000/api/platform/auth/status'
$erpUrl = 'http://127.0.0.1:3000/platform/'
$erpUserSid = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
$erpPathHash = [Security.Cryptography.SHA256]::Create()
try { $erpRootHash = ([BitConverter]::ToString($erpPathHash.ComputeHash([Text.Encoding]::UTF8.GetBytes($erpRoot.ToLowerInvariant())))).Replace('-', '').Substring(0, 12) }
finally { $erpPathHash.Dispose() }
$erpTaskName = 'MachiningERP-' + $erpUserSid.Split('-')[-1] + '-' + $erpRootHash
function Initialize-ErpRuntime {
    $null = New-Item -ItemType Directory -Path $erpRuntimeRoot -Force
}
function Get-ErpScheduledTask($Folder) {
    foreach ($erpRegisteredTask in $Folder.GetTasks(1)) {
        if ($erpRegisteredTask.Name -eq $erpTaskName) { return $erpRegisteredTask }
    }
    return $null
}
function Test-ErpPort([int]$Port) {
    $erpClient = New-Object Net.Sockets.TcpClient
    try { return $erpClient.ConnectAsync('127.0.0.1', $Port).Wait(500) -and $erpClient.Connected }
    catch { return $false }
    finally { $erpClient.Dispose() }
}
function Test-ErpEndpoint([string]$Url, [switch]$AuthStatus) {
    try {
        if ($AuthStatus) {
            $erpResult = Invoke-RestMethod -Uri $Url -TimeoutSec 3
            return $erpResult.PSObject.Properties.Name -contains 'authenticated'
        }
        $erpResult = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 3
        return $erpResult.StatusCode -eq 200
    } catch { return $false }
}
function Write-ErpRuntimeLog([string]$Message) {
    Add-Content -LiteralPath (Join-Path $erpRuntimeRoot 'launcher.log') -Value ((Get-Date -Format 'yyyy-MM-dd HH:mm:ss') + ' ' + $Message) -Encoding UTF8
}
function Stop-ErpTrackedProcess($Record, [string]$ExpectedPath) {
    if (-not $Record) { return }
    $erpTrackedProcess = Get-Process -Id ([int]$Record.pid) -ErrorAction SilentlyContinue
    if (-not $erpTrackedProcess) { return }
    if ($erpTrackedProcess.StartTime.ToUniversalTime().Ticks.ToString() -ne $Record.start_ticks -or
        -not [string]::Equals($erpTrackedProcess.Path, $ExpectedPath, [StringComparison]::OrdinalIgnoreCase)) {
        throw '进程身份已变化，未停止可能属于其他程序的进程。'
    }
    & "$env:SystemRoot\System32\taskkill.exe" /PID $erpTrackedProcess.Id /T /F 2>&1 | Out-Null
    if ($LASTEXITCODE -ne 0 -and (Get-Process -Id $erpTrackedProcess.Id -ErrorAction SilentlyContinue)) { throw '未能停止 ERP 子进程。' }
}
