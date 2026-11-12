; Inno Setup script for ddns-updater on Windows.
;
; Compile with:  ISCC.exe windows\ddns-updater.iss
; (or "C:\Program Files (x86)\Inno Setup 6\ISCC.exe" windows\ddns-updater.iss)
;
; The installer copies the executable, the configuration and the launcher
; scripts to the installation directory, then registers a scheduled task
; running as the SYSTEM account at machine startup, and starts it.
;
; NOTE: the config.json shipped next to this script contains the DuckDNS
; token. Remove it from the [Files] section and let the installer create an
; empty configuration if you distribute the setup to other machines.

#define AppName "ddns-updater"
#define AppVersion "2.10.0"
#define AppPublisher "qdm12"
#define AppURL "https://github.com/qdm12/ddns-updater"
#define AppExeName "ddns-updater.exe"
#define TaskName "ddns-updater"
#define WebUI "http://127.0.0.1:8000"

[Setup]
AppId={{B7C2E9A4-3D51-4F8E-9A62-1C5D8E4F7B03}
AppName={#AppName}
AppVersion={#AppVersion}
AppVerName={#AppName} {#AppVersion}
AppPublisher={#AppPublisher}
AppPublisherURL={#AppURL}
AppSupportURL={#AppURL}
AppUpdatesURL={#AppURL}
DefaultDirName=C:\Tools\ddns-updater
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
DisableDirPage=no
PrivilegesRequired=admin
OutputDir=output
; The setup program keeps a fixed file name, set by OutputBaseFilename below;
; the version is in the installer metadata.
OutputBaseFilename=ddns-updater-setup
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
UninstallDisplayIcon={app}\{#AppExeName}
UninstallDisplayName={#AppName} {#AppVersion}
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
MinVersion=10.0

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "autostart"; Description: "Start ddns-updater automatically at machine startup"; GroupDescription: "Startup:"
Name: "desktopicon"; Description: "Create a desktop shortcut for the web UI"; GroupDescription: "Shortcuts:"; Flags: unchecked

[Files]
Source: "ddns-updater.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "run-ddns-updater.cmd"; DestDir: "{app}"; Flags: ignoreversion
Source: "run-ddns-updater.vbs"; DestDir: "{app}"; Flags: ignoreversion
Source: "install-scheduled-task.ps1"; DestDir: "{app}"; Flags: ignoreversion
Source: "uninstall-scheduled-task.ps1"; DestDir: "{app}"; Flags: ignoreversion
; Only installed when missing, so that re-installing never overwrites
; the credentials of an existing installation.
Source: "data\config.json"; DestDir: "{app}\data"; Flags: onlyifdoesntexist uninsneveruninstall
; The runtime data (history, settings and log) is created by the application
; on first start and is kept on uninstall.

[Dirs]
Name: "{app}\data"

[Icons]
Name: "{group}\Open web UI"; Filename: "{#WebUI}"; WorkingDir: "{app}"
Name: "{group}\Open log folder"; Filename: "{app}\data"
Name: "{autodesktop}\Open web UI"; Filename: "{#WebUI}"; WorkingDir: "{app}"; Tasks: desktopicon
; The uninstall shortcut is created automatically by Inno Setup.

[Run]
Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; \
    Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\install-scheduled-task.ps1"""; \
    WorkingDir: "{app}"; Flags: runhidden waituntilterminated; \
    StatusMsg: "Registering the scheduled task and starting ddns-updater..."; Tasks: autostart
Filename: "{app}\{#AppExeName}"; Parameters: "version"; \
    WorkingDir: "{app}"; Flags: runhidden; Tasks: autostart

[UninstallRun]
; Stop the application and remove the scheduled task before deleting files.
Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; \
    Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\uninstall-scheduled-task.ps1"""; \
    WorkingDir: "{app}"; Flags: runhidden waituntilterminated; RunOnceId: "StopTask"

[Code]
{ Stop the running application and its scheduled task before the
  installation replaces the executable file. Replacing a running executable
  fails with a sharing violation, which aborts the installation. }
function InitializeSetup(): Boolean;
var
  ResultCode: Integer;
begin
  Exec(ExpandConstant('{sys}\WindowsPowerShell\v1.0\powershell.exe'),
    '-NoProfile -ExecutionPolicy Bypass -Command "Stop-ScheduledTask -TaskName ''ddns-updater'' -ErrorAction SilentlyContinue"',
    '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Exec(ExpandConstant('{sys}\taskkill.exe'), '/F /IM ddns-updater.exe',
    '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Result := True;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usPostUninstall then
  begin
    { Remove the runtime data, but keep the configuration file so that
      the credentials survive a re-installation. }
    DeleteFile(ExpandConstant('{app}\data\settings.json'));
    DeleteFile(ExpandConstant('{app}\data\updates.json'));
    DeleteFile(ExpandConstant('{app}\data\ddns-updater.log'));
    { RemoveDir only succeeds if the directory is empty. }
    RemoveDir(ExpandConstant('{app}\data'));
  end;
end;
