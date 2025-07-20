-- Tabela de mensagens
CREATE TABLE IF NOT EXISTS zapcore_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES zapcore_sessions(id) ON DELETE CASCADE,
    message_id VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL CHECK (type IN (
        'textMessage', 'imageMessage', 'videoMessage', 'audioMessage',
        'documentMessage', 'stickerMessage', 'contactMessage', 'locationMessage',
        'liveLocationMessage', 'gifMessage', 'pollMessage', 'reactionMessage',
        'buttonsMessage', 'listMessage'
    )),
    direction VARCHAR(20) NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending', 'sent', 'delivered', 'read', 'failed'
    )),
    from_jid VARCHAR(100) NOT NULL,
    to_jid VARCHAR(100) NOT NULL,
    content TEXT,
    media_id UUID,
    caption TEXT,
    timestamp TIMESTAMPTZ NOT NULL,
    reply_to_id VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índices para otimização
CREATE INDEX IF NOT EXISTS idx_messages_session_id ON zapcore_messages(session_id);
CREATE INDEX IF NOT EXISTS idx_messages_message_id ON zapcore_messages(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_type ON zapcore_messages(type);
CREATE INDEX IF NOT EXISTS idx_messages_direction ON zapcore_messages(direction);
CREATE INDEX IF NOT EXISTS idx_messages_status ON zapcore_messages(status);
CREATE INDEX IF NOT EXISTS idx_messages_from_jid ON zapcore_messages(from_jid);
CREATE INDEX IF NOT EXISTS idx_messages_to_jid ON zapcore_messages(to_jid);
CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON zapcore_messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON zapcore_messages(created_at);

-- Trigger para atualizar updated_at automaticamente
CREATE TRIGGER update_messages_updated_at 
    BEFORE UPDATE ON zapcore_messages 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
