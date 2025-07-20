-- Remover trigger
DROP TRIGGER IF EXISTS update_contacts_updated_at ON zapcore_contacts;

-- Remover índices
DROP INDEX IF EXISTS idx_contacts_last_seen;
DROP INDEX IF EXISTS idx_contacts_is_wa_contact;
DROP INDEX IF EXISTS idx_contacts_is_my_contact;
DROP INDEX IF EXISTS idx_contacts_is_business;
DROP INDEX IF EXISTS idx_contacts_phone_number;
DROP INDEX IF EXISTS idx_contacts_name;
DROP INDEX IF EXISTS idx_contacts_jid;
DROP INDEX IF EXISTS idx_contacts_session_id;

-- Remover tabela
DROP TABLE IF EXISTS zapcore_contacts;
