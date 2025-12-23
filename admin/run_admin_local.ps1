# run_admin_local.ps1 - Roda o Admin Dashboard localmente conectando na VPS
# Execute: .\run_admin_local.ps1

param(
    [int]$Port = 9090
)

Write-Host "🖥️ Iniciando Admin Dashboard Local" -ForegroundColor Cyan
Write-Host "Porta: $Port"
Write-Host ""

# Diretório
$LocalDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $LocalDir

# Compila se necessário
if (!(Test-Path "admin_server.exe") -or ((Get-Item "admin_server.go").LastWriteTime -gt (Get-Item "admin_server.exe").LastWriteTime)) {
    Write-Host "🔨 Compilando..." -ForegroundColor Yellow
    go build -o admin_server.exe .
    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Erro na compilação" -ForegroundColor Red
        exit 1
    }
}

Write-Host ""
Write-Host "╔══════════════════════════════════════════════════╗" -ForegroundColor Green
Write-Host "║     GoAnime Admin Dashboard                      ║" -ForegroundColor Green
Write-Host "╠══════════════════════════════════════════════════╣" -ForegroundColor Green
Write-Host "║  URL: http://localhost:$Port                      ║" -ForegroundColor Green
Write-Host "║  User: admin                                     ║" -ForegroundColor Green
Write-Host "║  Pass: goanime2024                               ║" -ForegroundColor Green
Write-Host "╚══════════════════════════════════════════════════╝" -ForegroundColor Green
Write-Host ""
Write-Host "Pressione Ctrl+C para parar" -ForegroundColor Yellow
Write-Host ""

# Abre navegador após 2 segundos
Start-Job -ScriptBlock {
    Start-Sleep 2
    Start-Process "http://localhost:$using:Port"
} | Out-Null

# Roda o servidor
.\admin_server.exe
