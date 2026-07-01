$ErrorActionPreference = "SilentlyContinue"

Get-Process -Name "cloudflared" | Stop-Process -Force
Get-Process -Name "go" | Stop-Process -Force
Get-Process -Name "server" | Stop-Process -Force

Write-Host "Локальный чат и Cloudflare Tunnel остановлены."
