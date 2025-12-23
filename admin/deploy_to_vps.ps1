# deploy_to_vps.ps1 - Deploy do Admin Dashboard para VPS via SSH
# Execute: .\deploy_to_vps.ps1

param(
    [string]$VpsHost = "2804:54:c100:2::11",
    [string]$VpsUser = "root",
    [int]$AdminPort = 9090
)

Write-Host "🚀 Deploy do Admin Dashboard para VPS" -ForegroundColor Cyan
Write-Host "=" * 50

# Diretório local
$LocalDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $LocalDir

# Verifica arquivos
Write-Host "📋 Verificando arquivos..." -ForegroundColor Yellow
$RequiredFiles = @("admin_server.go", "go.mod", "dashboard\index.html")
foreach ($file in $RequiredFiles) {
    if (!(Test-Path $file)) {
        Write-Host "❌ Arquivo não encontrado: $file" -ForegroundColor Red
        exit 1
    }
}
Write-Host "✅ Todos os arquivos encontrados" -ForegroundColor Green

# Cria diretório na VPS
Write-Host "`n📁 Criando diretórios na VPS..." -ForegroundColor Yellow
ssh ${VpsUser}@${VpsHost} "mkdir -p /opt/goanime/admin/dashboard"

# Copia arquivos
Write-Host "`n📤 Copiando arquivos para VPS..." -ForegroundColor Yellow
scp admin_server.go ${VpsUser}@${VpsHost}:/opt/goanime/admin/
scp go.mod ${VpsUser}@${VpsHost}:/opt/goanime/admin/
scp dashboard/index.html ${VpsUser}@${VpsHost}:/opt/goanime/admin/dashboard/
scp seeding_handlers.go ${VpsUser}@${VpsHost}:/opt/goanime/admin/

# Compila e configura serviço
Write-Host "`n🔨 Compilando e configurando na VPS..." -ForegroundColor Yellow
$RemoteCommands = @"
cd /opt/goanime/admin
go mod tidy
go build -o admin_server admin_server.go

# Cria serviço systemd
cat > /etc/systemd/system/goanime-admin.service << 'EOF'
[Unit]
Description=GoAnime Admin Dashboard
After=network.target postgresql.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/goanime/admin
ExecStart=/opt/goanime/admin/admin_server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable goanime-admin
systemctl restart goanime-admin

# Abre porta no firewall
ufw allow $AdminPort/tcp 2>/dev/null || true

sleep 2
systemctl status goanime-admin --no-pager
"@

ssh ${VpsUser}@${VpsHost} $RemoteCommands

Write-Host "`n" -ForegroundColor Green
Write-Host "╔══════════════════════════════════════════════════════╗" -ForegroundColor Green
Write-Host "║         ✅ Admin Dashboard Instalado!                ║" -ForegroundColor Green
Write-Host "╠══════════════════════════════════════════════════════╣" -ForegroundColor Green
Write-Host "║                                                      ║" -ForegroundColor Green
Write-Host "║  🌐 URL: http://[$VpsHost]:$AdminPort                ║" -ForegroundColor Green
Write-Host "║  👤 User: admin                                      ║" -ForegroundColor Green
Write-Host "║  🔑 Pass: goanime2024                                ║" -ForegroundColor Green
Write-Host "║                                                      ║" -ForegroundColor Green
Write-Host "║  ⚠️  MUDE A SENHA EM PRODUÇÃO!                       ║" -ForegroundColor Yellow
Write-Host "║                                                      ║" -ForegroundColor Green
Write-Host "╚══════════════════════════════════════════════════════╝" -ForegroundColor Green
Write-Host ""

# Abre no navegador
$Url = "http://[$VpsHost]:$AdminPort"
Write-Host "🌐 Abrindo no navegador: $Url" -ForegroundColor Cyan
Start-Process $Url
