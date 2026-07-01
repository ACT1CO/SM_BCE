$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
$logsDir = Join-Path $projectRoot ".local-logs"
$composeOut = Join-Path $logsDir "docker-compose.out.log"
$composeErr = Join-Path $logsDir "docker-compose.err.log"
$tunnelOut = Join-Path $logsDir "serveo.out.log"
$tunnelErr = Join-Path $logsDir "serveo.err.log"
$hostPort = if ($env:HOST_PORT) { $env:HOST_PORT } else { "8081" }

New-Item -ItemType Directory -Force -Path $logsDir | Out-Null
Remove-Item -Force -ErrorAction SilentlyContinue $tunnelOut, $tunnelErr

Get-CimInstance Win32_Process |
    Where-Object { $_.Name -ieq "ssh.exe" -and $_.CommandLine -like "*serveo.net*" } |
    ForEach-Object { Stop-Process -Id $_.ProcessId -Force }

Write-Host "Запускаю Соцсети-ВСЁ! через Docker и PostgreSQL..."
$env:HOST_PORT = $hostPort
Start-Process -FilePath "docker" `
    -ArgumentList "compose up -d --build" `
    -WorkingDirectory $projectRoot `
    -RedirectStandardOutput $composeOut `
    -RedirectStandardError $composeErr `
    -WindowStyle Hidden `
    -Wait

Start-Sleep -Seconds 3

Write-Host "Проверяю http://localhost:$hostPort..."
$status = (Invoke-WebRequest -Uri "http://localhost:$hostPort" -UseBasicParsing).StatusCode
if ($status -ne 200) {
    throw "Локальный сервер ответил не 200, а $status"
}

Write-Host "Запускаю публичный SSH tunnel через Serveo..."
Start-Process -FilePath "ssh" `
    -ArgumentList "-o StrictHostKeyChecking=no -o ServerAliveInterval=60 -R 80:localhost:$hostPort serveo.net" `
    -WorkingDirectory $projectRoot `
    -RedirectStandardOutput $tunnelOut `
    -RedirectStandardError $tunnelErr `
    -WindowStyle Hidden

Write-Host "Жду публичную ссылку..."
for ($i = 0; $i -lt 60; $i++) {
    Start-Sleep -Seconds 1
    if (Test-Path $tunnelOut) {
        $log = Get-Content $tunnelOut -Raw
        $match = [regex]::Match($log, "https://[a-z0-9-]+\.serveousercontent\.com")
        if ($match.Success) {
            Write-Host ""
            Write-Host "Публичный адрес чата:"
            Write-Host $match.Value
            Write-Host ""
            exit 0
        }
    }
}

Write-Host "Не удалось получить ссылку. Логи:"
Write-Host $tunnelOut
Write-Host $tunnelErr
