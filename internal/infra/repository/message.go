package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"zapcore/internal/domain/message"
	"zapcore/pkg/logger"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// MessageRepository implementa o repositório de mensagens
type MessageRepository struct {
	db     *sql.DB
	logger *logger.Logger
}

// NewMessageRepository cria uma nova instância do repositório
func NewMessageRepository(db *sql.DB, zeroLogger zerolog.Logger) *MessageRepository {
	return &MessageRepository{
		db:     db,
		logger: logger.NewFromZerolog(zeroLogger),
	}
}

// Create cria uma nova mensagem
func (r *MessageRepository) Create(ctx context.Context, msg *message.Message) error {
	query := `
		INSERT INTO zapcore_messages (
			id, session_id, message_id, type, direction, status, 
			from_jid, to_jid, content, media_id, caption, 
			timestamp, reply_to_id, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	metadataJSON, err := json.Marshal(msg.Metadata)
	if err != nil {
		return fmt.Errorf("erro ao serializar metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		msg.ID,
		msg.SessionID,
		msg.MessageID,
		msg.Type,
		msg.Direction,
		msg.Status,
		msg.FromJID,
		msg.ToJID,
		msg.Content,
		msg.MediaID,
		msg.Caption,
		msg.Timestamp,
		msg.ReplyToID,
		metadataJSON,
		msg.CreatedAt,
		msg.UpdatedAt,
	)

	if err != nil {
		r.logger.Error().Err(err).Str("message_id", msg.MessageID).Msg("Erro ao criar mensagem")
		return fmt.Errorf("erro ao criar mensagem: %w", err)
	}

	r.logger.Info().Str("message_id", msg.MessageID).Str("session_id", msg.SessionID.String()).Msg("Mensagem criada com sucesso")
	return nil
}

// GetByID busca uma mensagem pelo ID
func (r *MessageRepository) GetByID(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	query := `
		SELECT id, session_id, message_id, type, direction, status,
			   from_jid, to_jid, content, media_id, caption,
			   timestamp, reply_to_id, metadata, created_at, updated_at
		FROM zapcore_messages
		WHERE id = $1
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanMessage(row)
}

// GetByMessageID busca uma mensagem pelo message_id do WhatsApp
func (r *MessageRepository) GetByMessageID(ctx context.Context, messageID string) (*message.Message, error) {
	query := `
		SELECT id, session_id, message_id, type, direction, status,
			   from_jid, to_jid, content, media_id, caption,
			   timestamp, reply_to_id, metadata, created_at, updated_at
		FROM zapcore_messages
		WHERE message_id = $1
	`

	row := r.db.QueryRowContext(ctx, query, messageID)
	return r.scanMessage(row)
}

// Update atualiza uma mensagem
func (r *MessageRepository) Update(ctx context.Context, msg *message.Message) error {
	query := `
		UPDATE zapcore_messages 
		SET type = $2, direction = $3, status = $4, from_jid = $5, to_jid = $6,
			content = $7, media_id = $8, caption = $9, timestamp = $10,
			reply_to_id = $11, metadata = $12, updated_at = $13
		WHERE id = $1
	`

	metadataJSON, err := json.Marshal(msg.Metadata)
	if err != nil {
		return fmt.Errorf("erro ao serializar metadata: %w", err)
	}

	msg.UpdatedAt = time.Now()

	_, err = r.db.ExecContext(ctx, query,
		msg.ID,
		msg.Type,
		msg.Direction,
		msg.Status,
		msg.FromJID,
		msg.ToJID,
		msg.Content,
		msg.MediaID,
		msg.Caption,
		msg.Timestamp,
		msg.ReplyToID,
		metadataJSON,
		msg.UpdatedAt,
	)

	if err != nil {
		r.logger.Error().Err(err).Str("message_id", msg.MessageID).Msg("Erro ao atualizar mensagem")
		return fmt.Errorf("erro ao atualizar mensagem: %w", err)
	}

	r.logger.Info().Str("message_id", msg.MessageID).Msg("Mensagem atualizada com sucesso")
	return nil
}

// Delete remove uma mensagem
func (r *MessageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM zapcore_messages WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
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
	query := `
		SELECT id, session_id, message_id, type, direction, status,
			   from_jid, to_jid, content, media_id, caption,
			   timestamp, reply_to_id, metadata, created_at, updated_at
		FROM zapcore_messages
		WHERE session_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID, limit, offset)
	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao listar mensagens")
		return nil, fmt.Errorf("erro ao listar mensagens: %w", err)
	}
	defer rows.Close()

	return r.scanMessages(rows)
}

// ListByChatJID lista mensagens por chat JID com paginação
func (r *MessageRepository) ListByChatJID(ctx context.Context, sessionID uuid.UUID, chatJID string, limit, offset int) ([]*message.Message, error) {
	query := `
		SELECT id, session_id, message_id, type, direction, status,
			   from_jid, to_jid, content, media_id, caption,
			   timestamp, reply_to_id, metadata, created_at, updated_at
		FROM zapcore_messages
		WHERE session_id = $1 AND (from_jid = $2 OR to_jid = $2)
		ORDER BY timestamp DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID, chatJID, limit, offset)
	if err != nil {
		r.logger.Error().Err(err).
			Str("session_id", sessionID.String()).
			Str("chat_jid", chatJID).
			Msg("Erro ao listar mensagens do chat")
		return nil, fmt.Errorf("erro ao listar mensagens do chat: %w", err)
	}
	defer rows.Close()

	return r.scanMessages(rows)
}

// CountBySessionID conta mensagens por session ID
func (r *MessageRepository) CountBySessionID(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	query := `SELECT COUNT(*) FROM zapcore_messages WHERE session_id = $1`

	var count int64
	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(&count)
	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao contar mensagens")
		return 0, fmt.Errorf("erro ao contar mensagens: %w", err)
	}

	return count, nil
}

// UpdateStatus atualiza apenas o status de uma mensagem
func (r *MessageRepository) UpdateStatus(ctx context.Context, messageID string, status message.MessageStatus) error {
	query := `
		UPDATE zapcore_messages 
		SET status = $2, updated_at = NOW()
		WHERE message_id = $1
	`

	result, err := r.db.ExecContext(ctx, query, messageID, status)
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

// scanMessage converte uma linha do banco em uma entidade Message
func (r *MessageRepository) scanMessage(row *sql.Row) (*message.Message, error) {
	var msg message.Message
	var metadataJSON []byte

	err := row.Scan(
		&msg.ID,
		&msg.SessionID,
		&msg.MessageID,
		&msg.Type,
		&msg.Direction,
		&msg.Status,
		&msg.FromJID,
		&msg.ToJID,
		&msg.Content,
		&msg.MediaID,
		&msg.Caption,
		&msg.Timestamp,
		&msg.ReplyToID,
		&metadataJSON,
		&msg.CreatedAt,
		&msg.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, message.ErrMessageNotFound
		}
		return nil, fmt.Errorf("erro ao fazer scan da mensagem: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &msg.Metadata); err != nil {
			r.logger.Warn().Err(err).Msg("Erro ao deserializar metadata da mensagem")
			msg.Metadata = make(map[string]any)
		}
	} else {
		msg.Metadata = make(map[string]any)
	}

	return &msg, nil
}

// scanMessages converte múltiplas linhas do banco em entidades Message
func (r *MessageRepository) scanMessages(rows *sql.Rows) ([]*message.Message, error) {
	var messages []*message.Message

	for rows.Next() {
		var msg message.Message
		var metadataJSON []byte

		err := rows.Scan(
			&msg.ID,
			&msg.SessionID,
			&msg.MessageID,
			&msg.Type,
			&msg.Direction,
			&msg.Status,
			&msg.FromJID,
			&msg.ToJID,
			&msg.Content,
			&msg.MediaID,
			&msg.Caption,
			&msg.Timestamp,
			&msg.ReplyToID,
			&metadataJSON,
			&msg.CreatedAt,
			&msg.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("erro ao fazer scan da mensagem: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &msg.Metadata); err != nil {
				r.logger.Warn().Err(err).Msg("Erro ao deserializar metadata da mensagem")
				msg.Metadata = make(map[string]any)
			}
		} else {
			msg.Metadata = make(map[string]any)
		}

		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar sobre as mensagens: %w", err)
	}

	return messages, nil
}
