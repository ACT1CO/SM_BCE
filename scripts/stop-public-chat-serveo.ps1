$ErrorActionPreference = "SilentlyContinue"

$projectRoot = Split-Path -Parent $PSScriptRoot

Get-CimInstance Win32_Process |
    Where-Object { $_.Name -ieq "ssh.exe" -and $_.CommandLine -like "*serveo.net*" } |
    ForEach-Object { Stop-Process -Id $_.ProcessId -Force }

docker compose --project-directory $projectRoot down

Write-Host "Docker-чат и Serveo tunnel остановлены."
