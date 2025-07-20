-- Script de inicialização automática do banco de dados ZapCore
-- Este arquivo será executado automaticamente pelo PostgreSQL

-- Criar extensão para UUID se não existir
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Criar função para atualizar updated_at (será usada por todas as tabelas)
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- ============================================================================
-- MIGRAÇÃO 001: Criar tabela de sessões
-- ============================================================================

-- Tabela de sessões
CREATE TABLE IF NOT EXISTS zapcore_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(50) NOT NULL DEFAULT 'disconnected',
    jid VARCHAR(100),
    qr_code TEXT,
    proxy_url VARCHAR(500),
    webhook VARCHAR(500),
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_seen TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índices para otimização
CREATE INDEX IF NOT EXISTS idx_zapcore_sessions_name ON zapcore_sessions(name);
CREATE INDEX IF NOT EXISTS idx_zapcore_sessions_jid ON zapcore_sessions(jid);
CREATE INDEX IF NOT EXISTS idx_zapcore_sessions_status ON zapcore_sessions(status);
CREATE INDEX IF NOT EXISTS idx_zapcore_sessions_is_active ON zapcore_sessions(is_active);
CREATE INDEX IF NOT EXISTS idx_zapcore_sessions_created_at ON zapcore_sessions(created_at);

-- Trigger para atualizar updated_at automaticamente
DROP TRIGGER IF EXISTS update_zapcore_sessions_updated_at ON zapcore_sessions;
CREATE TRIGGER update_zapcore_sessions_updated_at
    BEFORE UPDATE ON zapcore_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- MIGRAÇÃO 002: Criar tabela de mensagens
-- ============================================================================

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
CREATE INDEX IF NOT EXISTS idx_zapcore_messages_session_id ON zapcore_messages(session_id);
CREATE INDEX IF NOT EXISTS idx_zapcore_messages_message_id ON zapcore_messages(message_id);
CREATE INDEX IF NOT EXISTS idx_zapcore_messages_type ON zapcore_messages(type);
CREATE INDEX IF NOT EXISTS idx_zapcore_messages_direction ON zapcore_messages(direction);
CREATE INDEX IF NOT EXISTS idx_zapcore_messages_status ON zapcore_messages(status);
CREATE INDEX IF NOT EXISTS idx_zapcore_messages_from_jid ON zapcore_messages(from_jid);
CREATE INDEX IF NOT EXISTS idx_zapcore_messages_to_jid ON zapcore_messages(to_jid);
CREATE INDEX IF NOT EXISTS idx_zapcore_messages_timestamp ON zapcore_messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_zapcore_messages_created_at ON zapcore_messages(created_at);

-- Trigger para atualizar updated_at automaticamente
CREATE TRIGGER update_zapcore_messages_updated_at 
    BEFORE UPDATE ON zapcore_messages 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- MIGRAÇÃO 003: Criar tabela de chats
-- ============================================================================

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
CREATE INDEX IF NOT EXISTS idx_zapcore_chats_session_id ON zapcore_chats(session_id);
CREATE INDEX IF NOT EXISTS idx_zapcore_chats_jid ON zapcore_chats(jid);
CREATE INDEX IF NOT EXISTS idx_zapcore_chats_type ON zapcore_chats(type);
CREATE INDEX IF NOT EXISTS idx_zapcore_chats_last_message_time ON zapcore_chats(last_message_time);
CREATE INDEX IF NOT EXISTS idx_zapcore_chats_unread_count ON zapcore_chats(unread_count);
CREATE INDEX IF NOT EXISTS idx_zapcore_chats_is_muted ON zapcore_chats(is_muted);
CREATE INDEX IF NOT EXISTS idx_zapcore_chats_is_pinned ON zapcore_chats(is_pinned);
CREATE INDEX IF NOT EXISTS idx_zapcore_chats_is_archived ON zapcore_chats(is_archived);
CREATE INDEX IF NOT EXISTS idx_zapcore_chats_created_at ON zapcore_chats(created_at);

-- Trigger para atualizar updated_at automaticamente
CREATE TRIGGER update_zapcore_chats_updated_at 
    BEFORE UPDATE ON zapcore_chats 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- MIGRAÇÃO 004: Criar tabela de contatos
-- ============================================================================

-- Tabela de contatos
CREATE TABLE IF NOT EXISTS zapcore_contacts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES zapcore_sessions(id) ON DELETE CASCADE,
    jid VARCHAR(100) NOT NULL,
    name VARCHAR(255),
    notify_name VARCHAR(255),
    phone_number VARCHAR(50),
    is_business BOOLEAN DEFAULT FALSE,
    is_enterprise BOOLEAN DEFAULT FALSE,
    is_my_contact BOOLEAN DEFAULT FALSE,
    is_wa_contact BOOLEAN DEFAULT TRUE,
    profile_picture_url TEXT,
    status_message TEXT,
    last_seen TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(session_id, jid)
);

-- Índices para otimização
CREATE INDEX IF NOT EXISTS idx_zapcore_contacts_session_id ON zapcore_contacts(session_id);
CREATE INDEX IF NOT EXISTS idx_zapcore_contacts_jid ON zapcore_contacts(jid);
CREATE INDEX IF NOT EXISTS idx_zapcore_contacts_phone_number ON zapcore_contacts(phone_number);
CREATE INDEX IF NOT EXISTS idx_zapcore_contacts_name ON zapcore_contacts(name);
CREATE INDEX IF NOT EXISTS idx_zapcore_contacts_notify_name ON zapcore_contacts(notify_name);
CREATE INDEX IF NOT EXISTS idx_zapcore_contacts_is_business ON zapcore_contacts(is_business);
CREATE INDEX IF NOT EXISTS idx_zapcore_contacts_is_my_contact ON zapcore_contacts(is_my_contact);
CREATE INDEX IF NOT EXISTS idx_zapcore_contacts_is_wa_contact ON zapcore_contacts(is_wa_contact);
CREATE INDEX IF NOT EXISTS idx_zapcore_contacts_created_at ON zapcore_contacts(created_at);

-- Trigger para atualizar updated_at automaticamente
CREATE TRIGGER update_zapcore_contacts_updated_at 
    BEFORE UPDATE ON zapcore_contacts 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- FINALIZAÇÃO
-- ============================================================================

-- Log de inicialização
DO $$
BEGIN
    RAISE NOTICE 'ZapCore Database initialized successfully!';
    RAISE NOTICE 'Tables created: zapcore_sessions, zapcore_messages, zapcore_chats, zapcore_contacts';
END $$;
