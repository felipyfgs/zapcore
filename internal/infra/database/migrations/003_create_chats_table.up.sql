-- Tabela de chats
CREATE TABLE IF NOT EXISTS zapcore_chats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES zapcore_sessions(id) ON DELETE CASCADE,
    jid VARCHAR(100) NOT NULL,
    name VARCHAR(255),
    type VARCHAR(20) NOT NULL,
    last_message_time TIMESTAMPTZ,
    message_count INTEGER DEFAULT 0,
    unread_count INTEGER DEFAULT 0,
    is_muted BOOLEAN DEFAULT FALSE,
    is_pinned BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(session_id, jid)
);

-- Índices para otimização
CREATE INDEX IF NOT EXISTS idx_chats_session_id ON zapcore_chats(session_id);
CREATE INDEX IF NOT EXISTS idx_chats_jid ON zapcore_chats(jid);
CREATE INDEX IF NOT EXISTS idx_chats_type ON zapcore_chats(type);
CREATE INDEX IF NOT EXISTS idx_chats_last_message_time ON zapcore_chats(last_message_time);
CREATE INDEX IF NOT EXISTS idx_chats_is_muted ON zapcore_chats(is_muted);
CREATE INDEX IF NOT EXISTS idx_chats_is_pinned ON zapcore_chats(is_pinned);
CREATE INDEX IF NOT EXISTS idx_chats_is_archived ON zapcore_chats(is_archived);
CREATE INDEX IF NOT EXISTS idx_chats_unread_count ON zapcore_chats(unread_count);

-- Trigger para atualizar updated_at automaticamente
CREATE TRIGGER update_chats_updated_at 
    BEFORE UPDATE ON zapcore_chats 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
