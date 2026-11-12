@echo off
rem Launcher for the ddns-updater executable.
rem It sets the environment variables used by the application and starts it.
rem This script is used by the "ddns-updater" scheduled task.

set "APPDIR=%~dp0"
if "%APPDIR:~-1%"=="\" set "APPDIR=%APPDIR:~0,-1%"

rem Data directory, where updates.json and settings.json are stored.
set "DATADIR=%APPDIR%\data"

rem JSON file with the DNS records to watch, in the data directory.
rem Fall back to the root of the installation directory if there is none there.
set "CONFIG_FILEPATH=%APPDIR%\data\config.json"
if not exist "%CONFIG_FILEPATH%" if exist "%APPDIR%\config.json" set "CONFIG_FILEPATH=%APPDIR%\config.json"

rem JSON file with the runtime settings, reloaded from the web UI.
set "SETTINGS_FILEPATH=%APPDIR%\data\settings.json"

rem Period between two public IP address checks.
set "PERIOD=1m"

rem Cooldown between two updates of the same record.
set "UPDATE_COOLDOWN_PERIOD=1m"

rem The local DNS server returns NXDOMAIN for duckdns.org and plain DNS
rem on port 53 is blocked on this network, so DNS over HTTPS on port 443
rem is used to resolve the provider and record host names.
set "RESOLVER_DOH_URL=https://cloudflare-dns.com/dns-query"

rem Only use the HTTP public IP fetchers, since the DNS ones need
rem direct DNS access on port 53.
set "PUBLICIP_FETCHERS=http"

rem Web UI, bound to the loopback interface only.
set "LISTENING_ADDRESS=127.0.0.1:12345"

set "LOG_LEVEL=info"

cd /d "%APPDIR%"

if not exist "%DATADIR%" mkdir "%DATADIR%"

rem Kill any stale instance first. A leftover process would keep the log
rem file locked, and the redirect below would fail with a sharing violation.
taskkill /F /IM ddns-updater.exe >nul 2>&1

"%APPDIR%\ddns-updater.exe" >> "%DATADIR%\ddns-updater.log" 2>&1
