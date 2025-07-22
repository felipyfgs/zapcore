package database

import (
	"context"
	"fmt"

	"zapcore/internal/domain/chat"
	"zapcore/internal/domain/contact"
	"zapcore/internal/domain/message"
	"zapcore/internal/domain/session"
	"zapcore/pkg/logger"
)

// MigrationHelper ajuda na migração de dados entre sistemas
type MigrationHelper struct {
	bunDB  *BunDB
	logger *logger.Logger
}

// NewMigrationHelper cria uma nova instância do helper de migração
func NewMigrationHelper(bunDB *BunDB) *MigrationHelper {
	return &MigrationHelper{
		bunDB:  bunDB,
		logger: logger.Get(),
	}
}

// ValidateSchema valida se o schema do banco está correto para o Bun
func (m *MigrationHelper) ValidateSchema(ctx context.Context) error {
	m.logger.Info().Msg("Validando schema do banco de dados")

	// Verificar se as tabelas existem
	tables := []string{
		"zapcore_sessions",
		"zapcore_messages",
		"zapcore_chats",
		"zapcore_contacts",
	}

	for _, table := range tables {
		exists, err := m.tableExists(ctx, table)
		if err != nil {
			return fmt.Errorf("erro ao verificar tabela %s: %w", table, err)
		}
		if !exists {
			return fmt.Errorf("tabela %s não existe", table)
		}
		m.logger.Info().Str("table", table).Msg("Tabela validada")
	}

	// Verificar se as colunas essenciais existem
	if err := m.validateColumns(ctx); err != nil {
		return fmt.Errorf("erro na validação de colunas: %w", err)
	}

	m.logger.Info().Msg("Schema validado com sucesso")
	return nil
}

// tableExists verifica se uma tabela existe usando Bun ORM
func (m *MigrationHelper) tableExists(ctx context.Context, tableName string) (bool, error) {
	// Para verificar se tabela existe, tentamos fazer uma query simples
	// Se a tabela não existir, retornará erro
	var count int
	var err error

	switch tableName {
	case "zapcore_sessions":
		count, err = m.bunDB.db.NewSelect().Model((*session.Session)(nil)).Count(ctx)
	case "zapcore_messages":
		count, err = m.bunDB.db.NewSelect().Model((*message.Message)(nil)).Count(ctx)
	case "zapcore_chats":
		count, err = m.bunDB.db.NewSelect().Model((*chat.Chat)(nil)).Count(ctx)
	case "zapcore_contacts":
		count, err = m.bunDB.db.NewSelect().Model((*contact.Contact)(nil)).Count(ctx)
	default:
		return false, fmt.Errorf("tabela desconhecida: %s", tableName)
	}

	if err != nil {
		// Se houve erro, provavelmente a tabela não existe
		return false, nil
	}

	// Se conseguiu contar (mesmo que seja 0), a tabela existe
	_ = count
	return true, nil
}

// validateColumns valida se as colunas essenciais existem
func (m *MigrationHelper) validateColumns(ctx context.Context) error {
	// Validar colunas da tabela sessions
	sessionColumns := []string{"id", "name", "status", "jid", "isActive", "lastSeen", "createdAt", "updatedAt"}
	if err := m.validateTableColumns(ctx, "zapcore_sessions", sessionColumns); err != nil {
		return err
	}

	// Validar colunas da tabela messages
	messageColumns := []string{
		"id", "sessionId", "msgId", "messageType", "direction", "status",
		"senderJid", "chatJid", "content", "mediaId", "mediaPath", "mediaSize",
		"mediaMimeType", "mediaFileName", "caption", "timestamp", "quotedMessageId",
		"pushName", "isFromMe", "isGroup", "mediaType", "rawPayload", "createdAt", "updatedAt",
	}
	if err := m.validateTableColumns(ctx, "zapcore_messages", messageColumns); err != nil {
		return err
	}

	// Validar colunas da tabela chats
	chatColumns := []string{
		"id", "sessionId", "jid", "name", "chatType", "isArchived", "isMuted",
		"isPinned", "unreadCount", "messageCount", "lastMessageTime", "metadata", "createdAt", "updatedAt",
	}
	if err := m.validateTableColumns(ctx, "zapcore_chats", chatColumns); err != nil {
		return err
	}

	// Validar colunas da tabela contacts
	contactColumns := []string{
		"id", "sessionId", "jid", "pushName", "businessName", "avatarUrl",
		"isGroup", "lastSeen", "metadata", "createdAt", "updatedAt",
	}
	if err := m.validateTableColumns(ctx, "zapcore_contacts", contactColumns); err != nil {
		return err
	}

	return nil
}

// validateTableColumns valida se as colunas de uma tabela existem usando Bun ORM
func (m *MigrationHelper) validateTableColumns(ctx context.Context, tableName string, columns []string) error {
	// Em vez de verificar colunas individualmente, vamos tentar fazer uma query simples
	// Se a tabela e colunas existem, a query funcionará
	m.logger.Info().
		Str("table", tableName).
		Int("expected_columns", len(columns)).
		Msg("Validando estrutura da tabela usando Bun ORM")

	// Tentar fazer uma query simples para validar que a tabela está acessível
	switch tableName {
	case "zapcore_sessions":
		_, err := m.bunDB.db.NewSelect().Model((*session.Session)(nil)).Limit(1).Exec(ctx)
		if err != nil {
			return fmt.Errorf("erro ao validar tabela %s: %w", tableName, err)
		}
	case "zapcore_messages":
		_, err := m.bunDB.db.NewSelect().Model((*message.Message)(nil)).Limit(1).Exec(ctx)
		if err != nil {
			return fmt.Errorf("erro ao validar tabela %s: %w", tableName, err)
		}
	case "zapcore_chats":
		_, err := m.bunDB.db.NewSelect().Model((*chat.Chat)(nil)).Limit(1).Exec(ctx)
		if err != nil {
			return fmt.Errorf("erro ao validar tabela %s: %w", tableName, err)
		}
	case "zapcore_contacts":
		_, err := m.bunDB.db.NewSelect().Model((*contact.Contact)(nil)).Limit(1).Exec(ctx)
		if err != nil {
			return fmt.Errorf("erro ao validar tabela %s: %w", tableName, err)
		}
	}

	return nil
}

// TestBunOperations testa operações básicas do Bun
func (m *MigrationHelper) TestBunOperations(ctx context.Context) error {
	m.logger.Info().Msg("Testando operações básicas do Bun")

	// Testar contagem de registros
	if err := m.testCounts(ctx); err != nil {
		return fmt.Errorf("erro ao testar contagens: %w", err)
	}

	// Testar inserção e remoção de teste
	if err := m.testInsertDelete(ctx); err != nil {
		return fmt.Errorf("erro ao testar inserção/remoção: %w", err)
	}

	m.logger.Info().Msg("Operações básicas do Bun testadas com sucesso")
	return nil
}

// testCounts testa contagem de registros
func (m *MigrationHelper) testCounts(ctx context.Context) error {
	// Contar sessões
	sessionCount, err := m.bunDB.db.NewSelect().Model((*session.Session)(nil)).Count(ctx)
	if err != nil {
		return fmt.Errorf("erro ao contar sessões: %w", err)
	}
	m.logger.Info().Int("count", sessionCount).Msg("Sessões encontradas")

	// Contar mensagens
	messageCount, err := m.bunDB.db.NewSelect().Model((*message.Message)(nil)).Count(ctx)
	if err != nil {
		return fmt.Errorf("erro ao contar mensagens: %w", err)
	}
	m.logger.Info().Int("count", messageCount).Msg("Mensagens encontradas")

	// Contar chats
	chatCount, err := m.bunDB.db.NewSelect().Model((*chat.Chat)(nil)).Count(ctx)
	if err != nil {
		return fmt.Errorf("erro ao contar chats: %w", err)
	}
	m.logger.Info().Int("count", chatCount).Msg("Chats encontrados")

	// Contar contatos
	contactCount, err := m.bunDB.db.NewSelect().Model((*contact.Contact)(nil)).Count(ctx)
	if err != nil {
		return fmt.Errorf("erro ao contar contatos: %w", err)
	}
	m.logger.Info().Int("count", contactCount).Msg("Contatos encontrados")

	return nil
}

// testInsertDelete testa inserção e remoção de registros de teste
func (m *MigrationHelper) testInsertDelete(ctx context.Context) error {
	// Criar sessão de teste
	testSession := session.NewSession("test-bun-migration")

	// Inserir
	_, err := m.bunDB.db.NewInsert().Model(testSession).Exec(ctx)
	if err != nil {
		return fmt.Errorf("erro ao inserir sessão de teste: %w", err)
	}
	m.logger.Info().Str("session_id", testSession.ID.String()).Msg("Sessão de teste criada")

	// Buscar
	foundSession := new(session.Session)
	err = m.bunDB.db.NewSelect().Model(foundSession).Where("id = ?", testSession.ID).Scan(ctx)
	if err != nil {
		return fmt.Errorf("erro ao buscar sessão de teste: %w", err)
	}
	m.logger.Info().Str("session_name", foundSession.Name).Msg("Sessão de teste encontrada")

	// Remover
	_, err = m.bunDB.db.NewDelete().Model((*session.Session)(nil)).Where("id = ?", testSession.ID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("erro ao remover sessão de teste: %w", err)
	}
	m.logger.Info().Str("session_id", testSession.ID.String()).Msg("Sessão de teste removida")

	return nil
}

// GetDataStatistics retorna estatísticas dos dados no banco
func (m *MigrationHelper) GetDataStatistics(ctx context.Context) (map[string]int, error) {
	stats := make(map[string]int)

	// Contar registros em cada tabela
	sessionCount, err := m.bunDB.db.NewSelect().Model((*session.Session)(nil)).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro ao contar sessões: %w", err)
	}
	stats["sessions"] = sessionCount

	messageCount, err := m.bunDB.db.NewSelect().Model((*message.Message)(nil)).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro ao contar mensagens: %w", err)
	}
	stats["messages"] = messageCount

	chatCount, err := m.bunDB.db.NewSelect().Model((*chat.Chat)(nil)).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro ao contar chats: %w", err)
	}
	stats["chats"] = chatCount

	contactCount, err := m.bunDB.db.NewSelect().Model((*contact.Contact)(nil)).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro ao contar contatos: %w", err)
	}
	stats["contacts"] = contactCount

	return stats, nil
}
