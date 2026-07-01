$ErrorActionPreference = "SilentlyContinue"

Get-CimInstance Win32_Process |
    Where-Object { $_.Name -ieq "ssh.exe" -and $_.CommandLine -like "*serveo.net*" } |
    ForEach-Object { Stop-Process -Id $_.ProcessId -Force }

Get-Process -Name "go" | Stop-Process -Force
Get-Process -Name "server" | Stop-Process -Force

Write-Host "Локальный чат и Serveo tunnel остановлены."
