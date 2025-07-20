-- Remover trigger
DROP TRIGGER IF EXISTS update_zapcore_sessions_updated_at ON zapcore_sessions;

-- Remover função
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Remover índices
DROP INDEX IF EXISTS idx_zapcore_sessions_created_at;
DROP INDEX IF EXISTS idx_zapcore_sessions_is_active;
DROP INDEX IF EXISTS idx_zapcore_sessions_status;
DROP INDEX IF EXISTS idx_zapcore_sessions_name;

-- Remover tabela
DROP TABLE IF EXISTS zapcore_sessions;
