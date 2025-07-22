package repository

import (
	"context"
	"fmt"
	"time"

	"zapcore/internal/domain/chat"
	"zapcore/pkg/logger"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ChatRepository implementa o repositório de chats usando Bun ORM
type ChatRepository struct {
	db     *bun.DB
	logger *logger.Logger
}

// NewChatRepository cria uma nova instância do repositório
func NewChatRepository(db *bun.DB) *ChatRepository {
	return &ChatRepository{
		db:     db,
		logger: logger.Get(),
	}
}

// Create cria um novo chat
func (r *ChatRepository) Create(ctx context.Context, c *chat.Chat) error {
	// Garantir que timestamps estão definidos
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = time.Now()
	}

	_, err := r.db.NewInsert().
		Model(c).
		Exec(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("chat_jid", c.JID).Msg("Erro ao criar chat")
		return fmt.Errorf("erro ao criar chat: %w", err)
	}

	r.logger.Info().Str("chat_jid", c.JID).Str("session_id", c.SessionID.String()).Msg("Chat criado com sucesso")
	return nil
}

// GetByID busca um chat pelo ID
func (r *ChatRepository) GetByID(ctx context.Context, id uuid.UUID) (*chat.Chat, error) {
	c := new(chat.Chat)
	err := r.db.NewSelect().
		Model(c).
		Where(`"id" = ?`, id).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, chat.ErrChatNotFound
		}
		return nil, fmt.Errorf("erro ao buscar chat por ID: %w", err)
	}

	return c, nil
}

// GetByJID busca um chat pelo JID e session ID
func (r *ChatRepository) GetByJID(ctx context.Context, sessionID uuid.UUID, jid string) (*chat.Chat, error) {
	c := new(chat.Chat)
	err := r.db.NewSelect().
		Model(c).
		Where(`"sessionId" = ? AND "jid" = ?`, sessionID, jid).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, chat.ErrChatNotFound
		}
		return nil, fmt.Errorf("erro ao buscar chat por JID: %w", err)
	}

	return c, nil
}

// Update atualiza um chat
func (r *ChatRepository) Update(ctx context.Context, c *chat.Chat) error {
	c.UpdatedAt = time.Now()

	result, err := r.db.NewUpdate().
		Model(c).
		Where(`"id" = ?`, c.ID).
		Exec(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("chat_jid", c.JID).Msg("Erro ao atualizar chat")
		return fmt.Errorf("erro ao atualizar chat: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return chat.ErrChatNotFound
	}

	r.logger.Info().Str("chat_jid", c.JID).Msg("Chat atualizado com sucesso")
	return nil
}

// Delete remove um chat
func (r *ChatRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.NewDelete().
		Model((*chat.Chat)(nil)).
		Where(`"id" = ?`, id).
		Exec(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("chat_id", id.String()).Msg("Erro ao deletar chat")
		return fmt.Errorf("erro ao deletar chat: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return chat.ErrChatNotFound
	}

	r.logger.Info().Str("chat_id", id.String()).Msg("Chat deletado com sucesso")
	return nil
}

// List retorna uma lista de chats com filtros
func (r *ChatRepository) List(ctx context.Context, filters chat.ListFilters) ([]*chat.Chat, error) {
	query := r.db.NewSelect().Model(&[]*chat.Chat{})

	// Aplicar filtros se fornecidos
	if filters.SessionID != nil {
		query = query.Where(`"sessionId" = ?`, *filters.SessionID)
	}

	// Definir limite padrão
	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}

	var chats []*chat.Chat
	err := query.
		OrderExpr(`"lastMessageTime" DESC NULLS LAST`).
		Limit(limit).
		Offset(filters.Offset).
		Scan(ctx, &chats)

	if err != nil {
		r.logger.Error().Err(err).Msg("Erro ao listar chats")
		return nil, fmt.Errorf("erro ao listar chats: %w", err)
	}

	return chats, nil
}

// ListBySessionID lista chats por session ID com paginação
func (r *ChatRepository) ListBySessionID(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*chat.Chat, error) {
	var chats []*chat.Chat

	err := r.db.NewSelect().
		Model(&chats).
		Where(`"sessionId" = ?`, sessionID).
		OrderExpr(`CASE WHEN "isPinned" THEN 0 ELSE 1 END, "lastMessageTime" DESC NULLS LAST`).
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao listar chats")
		return nil, fmt.Errorf("erro ao listar chats: %w", err)
	}

	return chats, nil
}

// ListByType lista chats por tipo
func (r *ChatRepository) ListByType(ctx context.Context, sessionID uuid.UUID, chatType chat.ChatType, limit, offset int) ([]*chat.Chat, error) {
	var chats []*chat.Chat

	err := r.db.NewSelect().
		Model(&chats).
		Where(`"sessionId" = ? AND "chatType" = ?`, sessionID, chatType).
		OrderExpr(`"lastMessageTime" DESC NULLS LAST`).
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		r.logger.Error().Err(err).
			Str("session_id", sessionID.String()).
			Str("chat_type", string(chatType)).
			Msg("Erro ao listar chats por tipo")
		return nil, fmt.Errorf("erro ao listar chats por tipo: %w", err)
	}

	return chats, nil
}

// ListArchived lista chats arquivados
func (r *ChatRepository) ListArchived(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*chat.Chat, error) {
	var chats []*chat.Chat

	err := r.db.NewSelect().
		Model(&chats).
		Where(`"sessionId" = ? AND "isArchived" = ?`, sessionID, true).
		OrderExpr(`"lastMessageTime" DESC NULLS LAST`).
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao listar chats arquivados")
		return nil, fmt.Errorf("erro ao listar chats arquivados: %w", err)
	}

	return chats, nil
}

// ListPinned lista chats fixados
func (r *ChatRepository) ListPinned(ctx context.Context, sessionID uuid.UUID) ([]*chat.Chat, error) {
	var chats []*chat.Chat

	err := r.db.NewSelect().
		Model(&chats).
		Where(`"sessionId" = ? AND "isPinned" = ?`, sessionID, true).
		OrderExpr(`"lastMessageTime" DESC NULLS LAST`).
		Scan(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao listar chats fixados")
		return nil, fmt.Errorf("erro ao listar chats fixados: %w", err)
	}

	return chats, nil
}

// UpdateLastMessage atualiza o timestamp da última mensagem
func (r *ChatRepository) UpdateLastMessage(ctx context.Context, sessionID uuid.UUID, jid string, timestamp time.Time) error {
	result, err := r.db.NewUpdate().
		Model((*chat.Chat)(nil)).
		Set(`"lastMessageTime" = ?`, timestamp).
		Set(`"updatedAt" = ?`, time.Now()).
		Where(`"sessionId" = ? AND "jid" = ?`, sessionID, jid).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("erro ao atualizar última mensagem do chat: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return chat.ErrChatNotFound
	}

	return nil
}

// IncrementMessageCount incrementa o contador de mensagens
func (r *ChatRepository) IncrementMessageCount(ctx context.Context, sessionID uuid.UUID, jid string) error {
	result, err := r.db.NewUpdate().
		Model((*chat.Chat)(nil)).
		Set("messageCount = messageCount + 1").
		Set(`"updatedAt" = ?`, time.Now()).
		Where(`"sessionId" = ? AND "jid" = ?`, sessionID, jid).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("erro ao incrementar contador de mensagens: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return chat.ErrChatNotFound
	}

	return nil
}

// MarkAsRead marca todas as mensagens como lidas
func (r *ChatRepository) MarkAsRead(ctx context.Context, sessionID uuid.UUID, jid string) error {
	result, err := r.db.NewUpdate().
		Model((*chat.Chat)(nil)).
		Set("\"unreadCount\" = ?", 0).
		Set("\"updatedAt\" = ?", time.Now()).
		Where("\"sessionId\" = ? AND \"jid\" = ?", sessionID, jid).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("erro ao marcar chat como lido: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return chat.ErrChatNotFound
	}

	return nil
}

// Count conta o total de chats
func (r *ChatRepository) Count(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	count, err := r.db.NewSelect().
		Model((*chat.Chat)(nil)).
		Where("\"sessionId\" = ?", sessionID).
		Count(ctx)

	if err != nil {
		return 0, fmt.Errorf("erro ao contar chats: %w", err)
	}

	return int64(count), nil
}

// CountUnread conta chats com mensagens não lidas
func (r *ChatRepository) CountUnread(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	count, err := r.db.NewSelect().
		Model((*chat.Chat)(nil)).
		Where(`"sessionId" = ? AND "unreadCount" > ?`, sessionID, 0).
		Count(ctx)

	if err != nil {
		return 0, fmt.Errorf("erro ao contar chats não lidos: %w", err)
	}

	return int64(count), nil
}

// ExistsByJID verifica se um chat existe pelo JID
func (r *ChatRepository) ExistsByJID(ctx context.Context, sessionID uuid.UUID, jid string) (bool, error) {
	exists, err := r.db.NewSelect().
		Model((*chat.Chat)(nil)).
		Where("\"sessionId\" = ? AND \"jid\" = ?", sessionID, jid).
		Exists(ctx)

	if err != nil {
		return false, fmt.Errorf("erro ao verificar existência do chat: %w", err)
	}

	return exists, nil
}

// IncrementUnreadCount incrementa o contador de mensagens não lidas
func (r *ChatRepository) IncrementUnreadCount(ctx context.Context, sessionID uuid.UUID, jid string) error {
	result, err := r.db.NewUpdate().
		Model((*chat.Chat)(nil)).
		Set("unreadCount = unreadCount + 1").
		Set(`"updatedAt" = ?`, time.Now()).
		Where(`"sessionId" = ? AND "jid" = ?`, sessionID, jid).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("erro ao incrementar contador de não lidas: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return chat.ErrChatNotFound
	}

	return nil
}

// GetBySessionID implementa a interface chat.Repository
func (r *ChatRepository) GetBySessionID(ctx context.Context, sessionID uuid.UUID, filters chat.ListFilters) ([]*chat.Chat, error) {
	// Usar o filtro SessionID
	filters.SessionID = &sessionID
	return r.List(ctx, filters)
}

// GetUnreadCount retorna total de chats não lidos (implementa interface)
func (r *ChatRepository) GetUnreadCount(ctx context.Context, sessionID uuid.UUID) (int, error) {
	count, err := r.CountUnread(ctx, sessionID)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}
