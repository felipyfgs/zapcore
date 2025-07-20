-- Remover trigger
DROP TRIGGER IF EXISTS update_chats_updated_at ON zapcore_chats;

-- Remover índices
DROP INDEX IF EXISTS idx_chats_unread_count;
DROP INDEX IF EXISTS idx_chats_is_archived;
DROP INDEX IF EXISTS idx_chats_is_pinned;
DROP INDEX IF EXISTS idx_chats_is_muted;
DROP INDEX IF EXISTS idx_chats_last_message_time;
DROP INDEX IF EXISTS idx_chats_type;
DROP INDEX IF EXISTS idx_chats_jid;
DROP INDEX IF EXISTS idx_chats_session_id;

-- Remover tabela
DROP TABLE IF EXISTS zapcore_chats;
