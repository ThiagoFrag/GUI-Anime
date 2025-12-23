#!/bin/bash
# deploy_admin.sh - Script para deploy do Admin Dashboard na VPS
# Execute: bash deploy_admin.sh

set -e

echo "🚀 Iniciando deploy do Admin Dashboard..."

# Cores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Diretórios
ADMIN_DIR="/opt/goanime/admin"
SERVICE_NAME="goanime-admin"

# Criar diretório
echo -e "${YELLOW}📁 Criando diretórios...${NC}"
sudo mkdir -p $ADMIN_DIR/dashboard

# Copiar arquivos
echo -e "${YELLOW}📋 Copiando arquivos...${NC}"
sudo cp admin_server.go $ADMIN_DIR/
sudo cp go.mod $ADMIN_DIR/
sudo cp -r dashboard/* $ADMIN_DIR/dashboard/

# Entrar no diretório e compilar
cd $ADMIN_DIR
echo -e "${YELLOW}🔨 Compilando servidor...${NC}"
sudo go mod tidy
sudo go build -o admin_server admin_server.go

# Criar serviço systemd
echo -e "${YELLOW}⚙️ Configurando serviço systemd...${NC}"
sudo tee /etc/systemd/system/$SERVICE_NAME.service > /dev/null <<EOF
[Unit]
Description=GoAnime Admin Dashboard
After=network.target postgresql.service

[Service]
Type=simple
User=root
WorkingDirectory=$ADMIN_DIR
ExecStart=$ADMIN_DIR/admin_server
Restart=always
RestartSec=5
Environment=PATH=/usr/local/go/bin:/usr/bin

[Install]
WantedBy=multi-user.target
EOF

# Recarregar e iniciar
echo -e "${YELLOW}🔄 Iniciando serviço...${NC}"
sudo systemctl daemon-reload
sudo systemctl enable $SERVICE_NAME
sudo systemctl restart $SERVICE_NAME

# Verificar status
sleep 2
if sudo systemctl is-active --quiet $SERVICE_NAME; then
    echo -e "${GREEN}✅ Admin Dashboard iniciado com sucesso!${NC}"
    echo -e "${GREEN}🌐 Acesse: http://SEU_IP:9090${NC}"
    echo -e "${YELLOW}📝 Login: admin / goanime2024${NC}"
else
    echo -e "${RED}❌ Erro ao iniciar serviço${NC}"
    sudo journalctl -u $SERVICE_NAME -n 20
fi

# Abrir porta no firewall (se ufw estiver ativo)
if command -v ufw &> /dev/null; then
    echo -e "${YELLOW}🔥 Abrindo porta 9090 no firewall...${NC}"
    sudo ufw allow 9090/tcp
fi

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║    GoAnime Admin Dashboard Instalado!        ║${NC}"
echo -e "${GREEN}╠══════════════════════════════════════════════╣${NC}"
echo -e "${GREEN}║  URL: http://[IP]:9090                       ║${NC}"
echo -e "${GREEN}║  User: admin                                 ║${NC}"
echo -e "${GREEN}║  Pass: goanime2024                           ║${NC}"
echo -e "${GREEN}║                                              ║${NC}"
echo -e "${GREEN}║  ⚠️  MUDE A SENHA EM PRODUÇÃO!               ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════╝${NC}"
