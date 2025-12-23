# Deploy do Sistema Social na VPS

## Arquivos Compilados

- **Windows**: `C:\Users\th\Documents\GoAnime\goanime-server.exe` (10.1 MB)
- **Linux**: `C:\Users\th\Documents\GoAnime\goanime-server-linux` (10.0 MB)

## Comandos de Deploy

### 1. Enviar para VPS

```powershell
# Via SCP (do Windows)
scp C:\Users\th\Documents\GoAnime\goanime-server-linux root@[2804:54:c100:2::11]:/opt/goanime/goanime-server
```

### 2. Na VPS - Criar tabela watching_status

```sql
-- Conectar ao PostgreSQL
psql -U goanime -d goanime_db

-- Executar migration
CREATE TABLE IF NOT EXISTS watching_status (
    id SERIAL PRIMARY KEY,
    user_id INTEGER UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    anime_title VARCHAR(255),
    anime_image TEXT,
    episode_num INTEGER DEFAULT 0,
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_watching_status_user_id ON watching_status(user_id);
CREATE INDEX IF NOT EXISTS idx_watching_status_updated ON watching_status(updated_at DESC);
```

### 3. Na VPS - Reiniciar Serviço

```bash
# Parar serviço atual
systemctl stop goanime-server

# Dar permissão de execução
chmod +x /opt/goanime/goanime-server

# Reiniciar
systemctl start goanime-server

# Verificar status
systemctl status goanime-server
```

### 4. Testar Endpoints

```bash
# Health check geral
curl http://[2804:54:c100:2::11]:8080/health

# Health check social
curl http://[2804:54:c100:2::11]:8080/social/health

# Trending (não requer auth)
curl http://[2804:54:c100:2::11]:8080/social/trending

# Recommendations (não requer auth)
curl http://[2804:54:c100:2::11]:8080/social/recommendations
```

## Endpoints Sociais Adicionados

| Endpoint | Método | Auth | Descrição |
|----------|--------|------|-----------|
| `/social/health` | GET | ❌ | Health check do sistema social |
| `/social/register` | POST | ❌ | Registrar usuário desktop |
| `/social/profile` | GET | ✅ | Obter perfil do usuário |
| `/social/user/lookup` | GET | ✅ | Buscar usuário por código |
| `/social/friends/add` | POST | ✅ | Adicionar amigo |
| `/social/friends/remove` | POST/DELETE | ✅ | Remover amigo |
| `/social/friends/list` | GET | ✅ | Listar amigos |
| `/social/friends/activity` | GET | ✅ | Ver atividade dos amigos |
| `/social/status/update` | POST | ✅ | Atualizar status de visualização |
| `/social/heartbeat` | POST | ✅ | Manter status online |
| `/social/sync` | GET | ✅ | Sincronizar dados do perfil |
| `/social/recommendations` | GET | ❌ | Obter recomendações |
| `/social/trending` | GET | ❌ | Ver animes em alta |

## Variáveis Globais Adicionadas

```go
var onlineUsers = make(map[int]time.Time)
var onlineMutex sync.RWMutex
const HeartbeatTimeout = 2 * time.Minute
```

## GoAnimeGUI - Configuração

O cliente desktop está configurado para usar:

```go
// pkg/social/friends.go
const APIBaseURL = "http://[2804:54:c100:2::11]:8080/social"
```

Após o deploy, o GoAnimeGUI deve mostrar "Conectado ao servidor social" na aba Social.
