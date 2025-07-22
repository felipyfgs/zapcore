package repository

import (
	"context"
	"fmt"
	"time"

	"zapcore/internal/domain/message"
	"zapcore/pkg/logger"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// MessageRepository implementa o repositório de mensagens usando Bun ORM
type MessageRepository struct {
	db     *bun.DB
	logger *logger.Logger
}

// NewMessageRepository cria uma nova instância do repositório
func NewMessageRepository(db *bun.DB) *MessageRepository {
	return &MessageRepository{
		db:     db,
		logger: logger.Get(),
	}
}

// ExistsByMsgID verifica se uma mensagem já existe pelo msgId
func (r *MessageRepository) ExistsByMsgID(ctx context.Context, msgID string) (bool, error) {
	exists, err := r.db.NewSelect().
		Model((*message.Message)(nil)).
		Where(`"msgId" = ?`, msgID).
		Exists(ctx)

	if err != nil {
		return false, fmt.Errorf("erro ao verificar existência da mensagem: %w", err)
	}

	return exists, nil
}

// ExistsByMsgIDAndSessionID verifica se uma mensagem já existe pelo msgId e sessionId
func (r *MessageRepository) ExistsByMsgIDAndSessionID(ctx context.Context, msgID string, sessionID uuid.UUID) (bool, error) {
	exists, err := r.db.NewSelect().
		Model((*message.Message)(nil)).
		Where(`"msgId" = ? AND "sessionId" = ?`, msgID, sessionID).
		Exists(ctx)

	if err != nil {
		return false, fmt.Errorf("erro ao verificar existência da mensagem por msgID e sessionID: %w", err)
	}

	return exists, nil
}

// Create cria uma nova mensagem
func (r *MessageRepository) Create(ctx context.Context, msg *message.Message) error {
	// Converter MsgID vazio para NULL se necessário
	if msg.MsgID == "" {
		msg.MsgID = uuid.New().String() // Gerar um ID temporário se vazio
	}

	// Garantir que timestamps estão definidos
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	if msg.UpdatedAt.IsZero() {
		msg.UpdatedAt = time.Now()
	}

	_, err := r.db.NewInsert().
		Model(msg).
		Exec(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("message_id", msg.MsgID).Msg("Erro ao criar mensagem")
		return fmt.Errorf("erro ao criar mensagem: %w", err)
	}

	r.logger.Info().Str("message_id", msg.MsgID).Str("session_id", msg.SessionID.String()).Msg("Mensagem criada com sucesso")
	return nil
}

// GetByID busca uma mensagem pelo ID
func (r *MessageRepository) GetByID(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	msg := new(message.Message)
	err := r.db.NewSelect().
		Model(msg).
		Where(`"id" = ?`, id).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, message.ErrMessageNotFound
		}
		return nil, fmt.Errorf("erro ao buscar mensagem por ID: %w", err)
	}

	return msg, nil
}

// GetByMessageID busca uma mensagem pelo messageId do WhatsApp
func (r *MessageRepository) GetByMessageID(ctx context.Context, messageID string) (*message.Message, error) {
	msg := new(message.Message)
	err := r.db.NewSelect().
		Model(msg).
		Where(`"msgId" = ?`, messageID).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, message.ErrMessageNotFound
		}
		return nil, fmt.Errorf("erro ao buscar mensagem por messageID: %w", err)
	}

	return msg, nil
}

// Update atualiza uma mensagem
func (r *MessageRepository) Update(ctx context.Context, msg *message.Message) error {
	msg.UpdatedAt = time.Now()

	result, err := r.db.NewUpdate().
		Model(msg).
		Where("? = ?", bun.Ident("id"), msg.ID).
		Exec(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("message_id", msg.MsgID).Msg("Erro ao atualizar mensagem")
		return fmt.Errorf("erro ao atualizar mensagem: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return message.ErrMessageNotFound
	}

	r.logger.Info().Str("message_id", msg.MsgID).Msg("Mensagem atualizada com sucesso")
	return nil
}

// Delete remove uma mensagem
func (r *MessageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.NewDelete().
		Model((*message.Message)(nil)).
		Where("? = ?", bun.Ident("id"), id).
		Exec(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("message_id", id.String()).Msg("Erro ao deletar mensagem")
		return fmt.Errorf("erro ao deletar mensagem: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return message.ErrMessageNotFound
	}

	r.logger.Info().Str("message_id", id.String()).Msg("Mensagem deletada com sucesso")
	return nil
}

// ListBySessionID lista mensagens por session ID com paginação
func (r *MessageRepository) ListBySessionID(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*message.Message, error) {
	var messages []*message.Message

	err := r.db.NewSelect().
		Model(&messages).
		Where(`"sessionId" = ?`, sessionID).
		OrderExpr(`"timestamp" DESC`).
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao listar mensagens")
		return nil, fmt.Errorf("erro ao listar mensagens: %w", err)
	}

	return messages, nil
}

// ListByChatJID lista mensagens por chat JID com paginação
func (r *MessageRepository) ListByChatJID(ctx context.Context, sessionID uuid.UUID, chatJID string, limit, offset int) ([]*message.Message, error) {
	var messages []*message.Message

	err := r.db.NewSelect().
		Model(&messages).
		Where("? = ? AND ? = ?", bun.Ident("sessionId"), sessionID, bun.Ident("chatJid"), chatJID).
		OrderExpr("? DESC", bun.Ident("timestamp")).
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		r.logger.Error().Err(err).
			Str("session_id", sessionID.String()).
			Str("chat_jid", chatJID).
			Msg("Erro ao listar mensagens do chat")
		return nil, fmt.Errorf("erro ao listar mensagens do chat: %w", err)
	}

	return messages, nil
}

// CountBySessionID conta mensagens por session ID
func (r *MessageRepository) CountBySessionID(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	count, err := r.db.NewSelect().
		Model((*message.Message)(nil)).
		Where(`"sessionId" = ?`, sessionID).
		Count(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao contar mensagens")
		return 0, fmt.Errorf("erro ao contar mensagens: %w", err)
	}

	return int64(count), nil
}

// UpdateStatus atualiza apenas o status de uma mensagem
func (r *MessageRepository) UpdateStatus(ctx context.Context, messageID string, status message.MessageStatus) error {
	result, err := r.db.NewUpdate().
		Model((*message.Message)(nil)).
		Set("? = ?", bun.Ident("status"), status).
		Set("? = ?", bun.Ident("updatedAt"), time.Now()).
		Where("? = ?", bun.Ident("msgId"), messageID).
		Exec(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("message_id", messageID).Msg("Erro ao atualizar status da mensagem")
		return fmt.Errorf("erro ao atualizar status da mensagem: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return message.ErrMessageNotFound
	}

	r.logger.Info().Str("message_id", messageID).Str("status", string(status)).Msg("Status da mensagem atualizado")
	return nil
}

// CountByStatus conta mensagens por status
func (r *MessageRepository) CountByStatus(ctx context.Context, sessionID uuid.UUID, status message.MessageStatus) (int, error) {
	count, err := r.db.NewSelect().
		Model((*message.Message)(nil)).
		Where("? = ? AND ? = ?", bun.Ident("sessionId"), sessionID, bun.Ident("status"), status).
		Count(ctx)

	if err != nil {
		r.logger.Error().Err(err).
			Str("session_id", sessionID.String()).
			Str("status", string(status)).
			Msg("Erro ao contar mensagens por status")
		return 0, fmt.Errorf("erro ao contar mensagens por status: %w", err)
	}

	return count, nil
}

// List retorna mensagens com filtros opcionais
func (r *MessageRepository) List(ctx context.Context, filters message.ListFilters) ([]*message.Message, error) {
	query := r.db.NewSelect().Model(&[]*message.Message{})

	// Aplicar filtros
	if filters.SessionID != nil {
		query = query.Where("? = ?", bun.Ident("sessionId"), *filters.SessionID)
	}

	// Definir limite padrão
	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}

	var messages []*message.Message
	err := query.
		OrderExpr("? DESC", bun.Ident("timestamp")).
		Limit(limit).
		Offset(filters.Offset).
		Scan(ctx, &messages)

	if err != nil {
		r.logger.Error().Err(err).Msg("Erro ao listar mensagens")
		return nil, fmt.Errorf("erro ao listar mensagens: %w", err)
	}

	return messages, nil
}

// GetBySessionID retorna mensagens de uma sessão específica
func (r *MessageRepository) GetBySessionID(ctx context.Context, sessionID uuid.UUID, filters message.ListFilters) ([]*message.Message, error) {
	return r.ListBySessionID(ctx, sessionID, filters.Limit, filters.Offset)
}

// GetConversation retorna mensagens de uma conversa específica
func (r *MessageRepository) GetConversation(ctx context.Context, sessionID uuid.UUID, jid string, filters message.ListFilters) ([]*message.Message, error) {
	return r.ListByChatJID(ctx, sessionID, jid, filters.Limit, filters.Offset)
}

// GetPendingMessages retorna mensagens pendentes para reenvio
func (r *MessageRepository) GetPendingMessages(ctx context.Context, sessionID uuid.UUID) ([]*message.Message, error) {
	var messages []*message.Message

	err := r.db.NewSelect().
		Model(&messages).
		Where("? = ? AND ? = ?", bun.Ident("sessionId"), sessionID, bun.Ident("status"), message.MessageStatusPending).
		OrderExpr("? ASC", bun.Ident("timestamp")).
		Scan(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao buscar mensagens pendentes")
		return nil, fmt.Errorf("erro ao buscar mensagens pendentes: %w", err)
	}

	return messages, nil
}
