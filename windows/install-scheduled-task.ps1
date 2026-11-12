<#
.SYNOPSIS
    Registers the "ddns-updater" scheduled task, starting the application
    when the machine boots.

.DESCRIPTION
    The task runs windows\run-ddns-updater.cmd, which sets the environment
    variables and starts the ddns-updater executable. It restarts the
    application up to 3 times, one minute apart, if it exits with an error.

    The task runs as the SYSTEM account, so the application starts at boot
    without needing anyone to log in. The web UI stays bound to the
    loopback interface (127.0.0.1), which is reachable from the desktop
    session as well.

    This script must be run as Administrator. It elevates itself if needed.

.EXAMPLE
    .\windows\install-scheduled-task.ps1
#>

$ErrorActionPreference = 'Stop'

# Elevate to administrator if not already elevated.
$currentIdentity = [Security.Principal.WindowsIdentity]::GetCurrent()
$currentPrincipal = New-Object Security.Principal.WindowsPrincipal($currentIdentity)
$isAdministrator = $currentPrincipal.IsInRole(
    [Security.Principal.WindowsBuiltInRole]::Administrator)

if (-not $isAdministrator) {
    Write-Host 'Elevating to administrator...'
    $arguments = @(
        '-NoProfile'
        '-ExecutionPolicy', 'Bypass'
        '-File', "`"$PSCommandPath`""
    )
    $process = Start-Process -FilePath 'powershell.exe' -Verb RunAs `
        -ArgumentList $arguments -Wait -PassThru
    exit $process.ExitCode
}

$taskName = 'ddns-updater'
$installDir = $PSScriptRoot
$scriptPath = Join-Path $PSScriptRoot 'run-ddns-updater.vbs'
$exePath = Join-Path $PSScriptRoot 'ddns-updater.exe'
$configPath = Join-Path $PSScriptRoot 'data\config.json'
if (-not (Test-Path $configPath)) {
    # Fall back to the root of the installation directory.
    $configPath = Join-Path $PSScriptRoot 'config.json'
}

if (-not (Test-Path $scriptPath)) {
    throw "Launcher script not found: $scriptPath"
}

if (-not (Test-Path $exePath)) {
    throw "Executable not found: $exePath (build it with: go build -o ddns-updater.exe ./cmd/ddns-updater)"
}

if (-not (Test-Path $configPath)) {
    throw "Configuration file not found: $configPath"
}

# Stop the task and kill any running instance before re-registering it.
# Otherwise the task scheduler considers the task as already running and
# silently ignores the start request (its MultipleInstances policy is
# IgnoreNew), and the old executable keeps holding the listening port.
function Wait-TaskStopped {
    param([string]$Name, [int]$TimeoutSeconds = 60)

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        $task = Get-ScheduledTask -TaskName $Name -ErrorAction SilentlyContinue
        if ($null -eq $task -or $task.State -ne 'Running') {
            return
        }
        Start-Sleep -Milliseconds 500
    }
}

Stop-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
# Stop-Process is used instead of taskkill because taskkill writes
# "process not found" on stderr, which is a terminating error with
# ErrorActionPreference set to Stop.
Stop-Process -Name 'ddns-updater' -Force -ErrorAction SilentlyContinue
Wait-TaskStopped -Name $taskName
Start-Sleep -Seconds 1

# wscript.exe runs the VBScript launcher, which starts the application
# with a hidden console window.
$action = New-ScheduledTaskAction -Execute 'wscript.exe' `
    -Argument """$scriptPath"""

$trigger = New-ScheduledTaskTrigger -AtStartup

$principal = New-ScheduledTaskPrincipal -UserId 'SYSTEM' `
    -LogonType ServiceAccount -RunLevel Highest

$settings = New-ScheduledTaskSettingsSet `
    -RestartCount 3 `
    -RestartInterval (New-TimeSpan -Minutes 1) `
    -ExecutionTimeLimit ([TimeSpan]::Zero) `
    -StartWhenAvailable `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries

Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger `
    -Principal $principal -Settings $settings `
    -Description 'ddns-updater: keeps DNS records in sync with the public IP address' `
    -Force | Out-Null

Write-Host "Scheduled task '$taskName' registered (SYSTEM account, at startup)."

# Start the task and make sure the application actually came up.
$started = $false
foreach ($attempt in 1..3) {
    Start-ScheduledTask -TaskName $taskName
    $deadline = (Get-Date).AddSeconds(15)
    while ((Get-Date) -lt $deadline) {
        if (Get-Process -Name 'ddns-updater' -ErrorAction SilentlyContinue) {
            $started = $true
            break
        }
        Start-Sleep -Seconds 1
    }
    if ($started) {
        break
    }
    Write-Host "The application did not start, retrying ($attempt/3)..."
    Stop-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
    Stop-Process -Name 'ddns-updater' -Force -ErrorAction SilentlyContinue
    Wait-TaskStopped -Name $taskName
}

if (-not $started) {
    Write-Warning "The task is registered but the application did not start. Check $installDir\data\ddns-updater.log"
} else {
    Write-Host 'Task started.'
    Write-Host 'The web UI is available at http://127.0.0.1:12345'
}
