# 🎮 GoAnime Admin Dashboard

Dashboard de administração completo para o GoAnime com:
- 📊 **Visão geral** de estatísticas em tempo real
- 👥 **Gestão de usuários** (VIP, ban, delete)
- 🌱 **Monitoramento de Seeding** comunitário
- 📝 **Logs de administração**

## 🚀 Instalação Rápida

### Opção 1: Rodar Localmente (Conectando na VPS)

1. **Configure o PostgreSQL na VPS para aceitar conexões externas:**
   ```bash
   ssh root@[2804:54:c100:2::11]
   
   # Editar pg_hba.conf
   nano /etc/postgresql/*/main/pg_hba.conf
   # Adicionar: host all all 0.0.0.0/0 md5
   
   # Editar postgresql.conf
   nano /etc/postgresql/*/main/postgresql.conf
   # Mudar: listen_addresses = '*'
   
   # Reiniciar
   systemctl restart postgresql
   
   # Abrir porta
   ufw allow 5432/tcp
   ```

2. **Crie o arquivo .env:**
   ```powershell
   cd c:\Users\th\Documents\codigos\GoAnimeGUI\admin
   Copy-Item .env.example .env
   # Edite .env com a senha correta do PostgreSQL
   ```

3. **Execute:**
   ```powershell
   .\run_admin_local.ps1
   ```

### Opção 2: Deploy na VPS

```powershell
cd c:\Users\th\Documents\codigos\GoAnimeGUI\admin
.\deploy_to_vps.ps1
```

## 🔐 Credenciais Padrão

| Campo | Valor |
|-------|-------|
| URL | http://localhost:9090 |
| Usuário | admin |
| Senha | goanime2024 |

⚠️ **MUDE A SENHA EM PRODUÇÃO!**

## 📊 Funcionalidades

### Dashboard Principal
- Total de usuários registrados
- Usuários online (últimos 5 min)
- Usuários VIP ativos
- Usuários banidos
- Seeders ativos contribuindo
- Total de dados semeados (bytes)
- Encodes pendentes na fila
- Novos registros (últimas 24h)

### Gestão de Usuários
- **Busca** por nome ou código de compartilhamento
- **Filtros**: Todos, Online, VIP, Banidos, Semeando
- **Ações**:
  - ⭐ Dar/Remover VIP (com duração em dias)
  - 🚫 Banir/Desbanir (com motivo)
  - 🔄 Resetar contagem de seeding
  - 🗑️ Deletar usuário

### Sistema de Seeding
- Ranking dos top 10 seeders
- Jobs de encode em fila
- Criar novos jobs manualmente
- Ver status de cada job (pending, assigned, completed, error)

### Logs de Admin
- Histórico de todas as ações
- Quem fez, o que fez, para quem
- IP e timestamp

## 🔧 Configuração

### Variáveis de Ambiente

| Variável | Descrição | Padrão |
|----------|-----------|--------|
| ADMIN_PORT | Porta do servidor | 9090 |
| ADMIN_USER | Usuário de login | admin |
| ADMIN_PASS | Senha de login | goanime2024 |
| DATABASE_URL | String de conexão PostgreSQL | postgres://...localhost... |

### Estrutura de Arquivos

```
admin/
├── admin_server.go      # Servidor Go principal
├── seeding_handlers.go  # Handlers para sistema de seeding
├── go.mod              # Dependências Go
├── dashboard/
│   └── index.html      # Frontend do dashboard
├── deploy_to_vps.ps1   # Script de deploy
├── run_admin_local.ps1 # Script para rodar local
├── .env.example        # Exemplo de configuração
└── README.md           # Este arquivo
```

## 🗄️ Tabelas do Banco de Dados

O admin cria automaticamente as seguintes colunas/tabelas:

```sql
-- Colunas adicionadas a social_users
ALTER TABLE social_users ADD COLUMN is_vip BOOLEAN DEFAULT FALSE;
ALTER TABLE social_users ADD COLUMN vip_expires_at TIMESTAMP;
ALTER TABLE social_users ADD COLUMN is_banned BOOLEAN DEFAULT FALSE;
ALTER TABLE social_users ADD COLUMN ban_reason TEXT;
ALTER TABLE social_users ADD COLUMN seeding_active BOOLEAN DEFAULT FALSE;
ALTER TABLE social_users ADD COLUMN seeding_bytes BIGINT DEFAULT 0;
ALTER TABLE social_users ADD COLUMN created_at TIMESTAMP DEFAULT NOW();

-- Tabela de jobs de seeding
CREATE TABLE seeding_jobs (
    id BIGSERIAL PRIMARY KEY,
    anime_name VARCHAR(500),
    episode_num INTEGER,
    file_url TEXT,
    file_size BIGINT,
    status VARCHAR(20) DEFAULT 'pending',
    assigned_to VARCHAR(64),
    assigned_at TIMESTAMP,
    completed_at TIMESTAMP,
    gofile_url TEXT,
    error_msg TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Tabela de logs de admin
CREATE TABLE admin_logs (
    id BIGSERIAL PRIMARY KEY,
    admin_user VARCHAR(50),
    action VARCHAR(50),
    target_user VARCHAR(64),
    details JSONB,
    ip_address VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW()
);
```

## 🔌 API Endpoints

| Método | Endpoint | Descrição | Auth |
|--------|----------|-----------|------|
| POST | /api/login | Login | ❌ |
| GET | /api/health | Health check | ❌ |
| POST | /api/logout | Logout | ✅ |
| GET | /api/dashboard | Estatísticas gerais | ✅ |
| GET | /api/users | Lista usuários | ✅ |
| POST | /api/users/action | Executar ação em usuário | ✅ |
| GET | /api/seeding/stats | Estatísticas de seeding | ✅ |
| GET/POST/DELETE | /api/seeding/jobs | Gerenciar jobs | ✅ |
| GET | /api/logs | Logs de admin | ✅ |

## 🛡️ Segurança

- Sessões com tokens aleatórios de 256 bits
- Expiração automática de sessões (24h)
- Hash SHA-256 para senhas
- CORS configurado
- Logs de todas as ações de admin

## 📱 Responsivo

O dashboard é responsivo e funciona em:
- 🖥️ Desktop
- 📱 Tablet
- 📱 Mobile (sidebar oculta)

## 🐛 Troubleshooting

### Erro de conexão com PostgreSQL
```
Verifique:
1. PostgreSQL está rodando na VPS
2. Porta 5432 está aberta
3. pg_hba.conf permite conexões externas
4. Senha está correta no .env
```

### Dashboard não carrega
```
Verifique:
1. Pasta dashboard/ existe com index.html
2. Porta 9090 não está em uso
3. Console do navegador para erros JS
```

### Sessão expira rapidamente
```
Por padrão sessões duram 24h.
Se estiver testando, verifique se o relógio do sistema está correto.
```

## 📞 Suporte

Desenvolvido para o projeto GoAnime.
Para bugs ou sugestões, abra uma issue no repositório.
