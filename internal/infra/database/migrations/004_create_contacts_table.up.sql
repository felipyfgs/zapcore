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
    is_push_name BOOLEAN DEFAULT FALSE,
    is_wa_contact BOOLEAN DEFAULT FALSE,
    profile_picture_url VARCHAR(500),
    status_message TEXT,
    last_seen TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(session_id, jid)
);

-- Índices para otimização
CREATE INDEX IF NOT EXISTS idx_contacts_session_id ON zapcore_contacts(session_id);
CREATE INDEX IF NOT EXISTS idx_contacts_jid ON zapcore_contacts(jid);
CREATE INDEX IF NOT EXISTS idx_contacts_name ON zapcore_contacts(name);
CREATE INDEX IF NOT EXISTS idx_contacts_phone_number ON zapcore_contacts(phone_number);
CREATE INDEX IF NOT EXISTS idx_contacts_is_business ON zapcore_contacts(is_business);
CREATE INDEX IF NOT EXISTS idx_contacts_is_my_contact ON zapcore_contacts(is_my_contact);
CREATE INDEX IF NOT EXISTS idx_contacts_is_wa_contact ON zapcore_contacts(is_wa_contact);
CREATE INDEX IF NOT EXISTS idx_contacts_last_seen ON zapcore_contacts(last_seen);

-- Trigger para atualizar updated_at automaticamente
CREATE TRIGGER update_contacts_updated_at 
    BEFORE UPDATE ON zapcore_contacts 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
