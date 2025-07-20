-- Remover trigger
DROP TRIGGER IF EXISTS update_messages_updated_at ON zapcore_messages;

-- Remover índices
DROP INDEX IF EXISTS idx_messages_created_at;
DROP INDEX IF EXISTS idx_messages_timestamp;
DROP INDEX IF EXISTS idx_messages_to_jid;
DROP INDEX IF EXISTS idx_messages_from_jid;
DROP INDEX IF EXISTS idx_messages_status;
DROP INDEX IF EXISTS idx_messages_direction;
DROP INDEX IF EXISTS idx_messages_type;
DROP INDEX IF EXISTS idx_messages_message_id;
DROP INDEX IF EXISTS idx_messages_session_id;

-- Remover tabela
DROP TABLE IF EXISTS zapcore_messages;
