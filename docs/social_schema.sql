-- ============================================
-- SCHEMA DO BANCO DE DADOS POSTGRESQL
-- Sistema Social GoAnime
-- ============================================

-- Extensão para UUID
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================
-- TABELA DE USUÁRIOS
-- ============================================
CREATE TABLE IF NOT EXISTS social_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(64) UNIQUE NOT NULL,
    username VARCHAR(50) NOT NULL,
    share_code VARCHAR(8) UNIQUE NOT NULL,
    avatar_url TEXT,
    auth_token VARCHAR(256),
    show_status BOOLEAN DEFAULT true,
    share_animes BOOLEAN DEFAULT true,
    total_watched INTEGER DEFAULT 0,
    is_online BOOLEAN DEFAULT false,
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Índices para performance
CREATE INDEX IF NOT EXISTS idx_users_share_code ON social_users(share_code);
CREATE INDEX IF NOT EXISTS idx_users_user_id ON social_users(user_id);
CREATE INDEX IF NOT EXISTS idx_users_is_online ON social_users(is_online);

-- ============================================
-- TABELA DE AMIZADES
-- ============================================
CREATE TABLE IF NOT EXISTS social_friendships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(64) NOT NULL REFERENCES social_users(user_id) ON DELETE CASCADE,
    friend_id VARCHAR(64) NOT NULL REFERENCES social_users(user_id) ON DELETE CASCADE,
    status VARCHAR(20) DEFAULT 'accepted', -- pending, accepted, blocked
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, friend_id)
);

-- Índices para performance
CREATE INDEX IF NOT EXISTS idx_friendships_user_id ON social_friendships(user_id);
CREATE INDEX IF NOT EXISTS idx_friendships_friend_id ON social_friendships(friend_id);

-- ============================================
-- TABELA DE STATUS DE VISUALIZAÇÃO
-- ============================================
CREATE TABLE IF NOT EXISTS social_watching_status (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(64) UNIQUE NOT NULL REFERENCES social_users(user_id) ON DELETE CASCADE,
    anime_title VARCHAR(255),
    anime_image TEXT,
    episode_num INTEGER,
    total_episodes INTEGER,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Índice para performance
CREATE INDEX IF NOT EXISTS idx_watching_user_id ON social_watching_status(user_id);

-- ============================================
-- TABELA DE HISTÓRICO DE ATIVIDADES
-- ============================================
CREATE TABLE IF NOT EXISTS social_activity_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(64) NOT NULL REFERENCES social_users(user_id) ON DELETE CASCADE,
    activity_type VARCHAR(50) NOT NULL, -- watch_start, watch_end, episode_complete, friend_added
    anime_title VARCHAR(255),
    episode_num INTEGER,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Índice para performance
CREATE INDEX IF NOT EXISTS idx_activity_user_id ON social_activity_log(user_id);
CREATE INDEX IF NOT EXISTS idx_activity_created_at ON social_activity_log(created_at);

-- ============================================
-- FUNÇÕES AUXILIARES
-- ============================================

-- Função para gerar share code único
CREATE OR REPLACE FUNCTION generate_share_code() RETURNS VARCHAR(8) AS $$
DECLARE
    new_code VARCHAR(8);
    code_exists BOOLEAN;
BEGIN
    LOOP
        new_code := UPPER(encode(gen_random_bytes(4), 'hex'));
        SELECT EXISTS(SELECT 1 FROM social_users WHERE share_code = new_code) INTO code_exists;
        EXIT WHEN NOT code_exists;
    END LOOP;
    RETURN new_code;
END;
$$ LANGUAGE plpgsql;

-- Função para atualizar timestamp
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers para atualizar updated_at automaticamente
CREATE TRIGGER update_users_timestamp
    BEFORE UPDATE ON social_users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_watching_timestamp
    BEFORE UPDATE ON social_watching_status
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- ============================================
-- FUNÇÃO PARA LIMPAR USUÁRIOS INATIVOS
-- (Opcional - executar periodicamente)
-- ============================================
CREATE OR REPLACE FUNCTION cleanup_inactive_status() RETURNS void AS $$
BEGIN
    -- Marca usuários como offline se não enviaram heartbeat em 2 minutos
    UPDATE social_users 
    SET is_online = false 
    WHERE is_online = true 
    AND last_seen < NOW() - INTERVAL '2 minutes';
    
    -- Limpa status de visualização antigos (mais de 1 hora sem atualização)
    DELETE FROM social_watching_status 
    WHERE updated_at < NOW() - INTERVAL '1 hour';
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- VIEWS ÚTEIS
-- ============================================

-- View de atividade dos amigos
CREATE OR REPLACE VIEW v_friends_activity AS
SELECT 
    f.user_id as viewer_id,
    u.user_id,
    u.username,
    u.avatar_url as avatar,
    u.is_online,
    u.last_seen,
    ws.anime_title,
    ws.anime_image,
    ws.episode_num,
    ws.total_episodes,
    ws.started_at as watching_since
FROM social_friendships f
JOIN social_users u ON f.friend_id = u.user_id
LEFT JOIN social_watching_status ws ON u.user_id = ws.user_id
WHERE f.status = 'accepted';

-- ============================================
-- DADOS INICIAIS (OPCIONAL)
-- ============================================

-- Você pode inserir dados de teste aqui se necessário
