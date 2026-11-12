' Launches the ddns-updater executable without showing a console window.
'
' The scheduled task runs the application in the interactive user session,
' so that the web UI on 127.0.0.1:12345 stays reachable from the browser.
' Running it through this script hides the console window (window style 0)
' and propagates the exit code, so that Task Scheduler can restart the
' application if it fails.

Option Explicit

Dim shell, fso, appDir, scriptPath, command, exitCode

Set shell = CreateObject("WScript.Shell")
Set fso = CreateObject("Scripting.FileSystemObject")

appDir = fso.GetParentFolderName(WScript.ScriptFullName)
scriptPath = appDir & "\run-ddns-updater.cmd"

command = "cmd.exe /c """ & scriptPath & """"

' Window style 0 hides the console window, and waiting for the return
' keeps the scheduled task running for as long as the application runs.
exitCode = shell.Run(command, 0, True)

WScript.Quit exitCode
