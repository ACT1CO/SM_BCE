$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
$logsDir = Join-Path $projectRoot ".local-logs"
$serverOut = Join-Path $logsDir "server.out.log"
$serverErr = Join-Path $logsDir "server.err.log"
$tunnelOut = Join-Path $logsDir "localhost-run.out.log"
$tunnelErr = Join-Path $logsDir "localhost-run.err.log"

New-Item -ItemType Directory -Force -Path $logsDir | Out-Null

Write-Host "Запускаю локальный сервер Соцсети-ВСЁ!..."
Start-Process -FilePath "go" `
    -ArgumentList "run ./cmd/server" `
    -WorkingDirectory $projectRoot `
    -RedirectStandardOutput $serverOut `
    -RedirectStandardError $serverErr `
    -WindowStyle Hidden

Start-Sleep -Seconds 2

Write-Host "Проверяю http://localhost:8080..."
$status = (Invoke-WebRequest -Uri "http://localhost:8080" -UseBasicParsing).StatusCode
if ($status -ne 200) {
    throw "Локальный сервер ответил не 200, а $status"
}

Write-Host "Запускаю публичный SSH tunnel через localhost.run..."
Start-Process -FilePath "ssh" `
    -ArgumentList "-o StrictHostKeyChecking=no -o ServerAliveInterval=60 -R 80:localhost:8080 nokey@localhost.run" `
    -WorkingDirectory $projectRoot `
    -RedirectStandardOutput $tunnelOut `
    -RedirectStandardError $tunnelErr `
    -WindowStyle Hidden

Write-Host "Жду публичную ссылку..."
for ($i = 0; $i -lt 20; $i++) {
    Start-Sleep -Seconds 1
    if (Test-Path $tunnelOut) {
        $log = Get-Content $tunnelOut -Raw
        $match = [regex]::Match($log, "https://[a-z0-9-]+\.lhr\.life")
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
