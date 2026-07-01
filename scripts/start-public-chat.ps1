$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
$cloudflared = "C:\Program Files (x86)\cloudflared\cloudflared.exe"
$logsDir = Join-Path $projectRoot ".local-logs"
$serverOut = Join-Path $logsDir "server.out.log"
$serverErr = Join-Path $logsDir "server.err.log"
$tunnelOut = Join-Path $logsDir "cloudflared.out.log"
$tunnelErr = Join-Path $logsDir "cloudflared.err.log"

New-Item -ItemType Directory -Force -Path $logsDir | Out-Null

if (-not (Test-Path $cloudflared)) {
    throw "cloudflared не найден: $cloudflared"
}

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

Write-Host "Запускаю Cloudflare Tunnel..."
Start-Process -FilePath $cloudflared `
    -ArgumentList "tunnel --no-prechecks --edge-ip-version 4 --protocol http2 --url http://localhost:8080" `
    -WorkingDirectory $projectRoot `
    -RedirectStandardOutput $tunnelOut `
    -RedirectStandardError $tunnelErr `
    -WindowStyle Hidden

Write-Host "Жду публичную ссылку..."
for ($i = 0; $i -lt 20; $i++) {
    Start-Sleep -Seconds 1
    if (Test-Path $tunnelErr) {
        $log = Get-Content $tunnelErr -Raw
        $match = [regex]::Match($log, "https://[a-z0-9-]+\.trycloudflare\.com")
        if ($match.Success) {
            Write-Host ""
            Write-Host "Публичный адрес чата:"
            Write-Host $match.Value
            Write-Host ""
            Write-Host "Если tunnel сразу остановился, проверь файл:"
            Write-Host $tunnelErr
            exit 0
        }
    }
}

Write-Host "Не удалось получить ссылку. Лог Cloudflare:"
Write-Host $tunnelErr
