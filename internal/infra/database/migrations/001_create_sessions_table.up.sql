-- Criar extensão para UUID se não existir
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

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
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_zapcore_sessions_updated_at
    BEFORE UPDATE ON zapcore_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
