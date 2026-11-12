<#
.SYNOPSIS
    Stops and removes the "ddns-updater" scheduled task and application.

.DESCRIPTION
    Called by the uninstaller of the setup program, before it deletes the
    installation files. The application is stopped and the process is given
    time to exit, otherwise its executable file stays locked and cannot be
    deleted.

.EXAMPLE
    .\windows\uninstall-scheduled-task.ps1
#>

$ErrorActionPreference = 'SilentlyContinue'

$taskName = 'ddns-updater'

Stop-ScheduledTask -TaskName $taskName

# Wait for the task instance to stop, otherwise the scheduler ignores
# the unregister request while the task is still running.
for ($i = 0; $i -lt 60; $i++) {
    $task = Get-ScheduledTask -TaskName $taskName
    if ($null -eq $task -or $task.State -ne 'Running') {
        break
    }
    Start-Sleep -Milliseconds 500
}

Stop-Process -Name 'ddns-updater' -Force

# Wait for the process to exit, so that its executable file is released.
for ($i = 0; $i -lt 60; $i++) {
    if (-not (Get-Process -Name 'ddns-updater' -ErrorAction SilentlyContinue)) {
        break
    }
    Start-Sleep -Milliseconds 500
}

Unregister-ScheduledTask -TaskName $taskName -Confirm:$false

if (Get-ScheduledTask -TaskName $taskName) {
    Write-Warning "Scheduled task '$taskName' could not be removed."
} else {
    Write-Host "Scheduled task '$taskName' removed."
}
