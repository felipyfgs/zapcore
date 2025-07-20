package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"zapcore/internal/domain/chat"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ChatRepository implementa o repositório de chats para PostgreSQL
type ChatRepository struct {
	db     *sql.DB
	logger zerolog.Logger
}

// NewChatRepository cria uma nova instância do repositório
func NewChatRepository(db *sql.DB, logger zerolog.Logger) *ChatRepository {
	return &ChatRepository{
		db:     db,
		logger: logger,
	}
}

// Create cria um novo chat
func (r *ChatRepository) Create(ctx context.Context, c *chat.Chat) error {
	query := `
		INSERT INTO zapcore_chats (
			id, session_id, jid, name, type, last_message_time,
			message_count, unread_count, is_muted, is_pinned, is_archived,
			metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	metadataJSON, err := json.Marshal(c.Metadata)
	if err != nil {
		return fmt.Errorf("erro ao serializar metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		c.ID,
		c.SessionID,
		c.JID,
		c.Name,
		c.Type,
		c.LastMessageTime,
		c.MessageCount,
		c.UnreadCount,
		c.IsMuted,
		c.IsPinned,
		c.IsArchived,
		metadataJSON,
		c.CreatedAt,
		c.UpdatedAt,
	)

	if err != nil {
		r.logger.Error().Err(err).Str("chat_jid", c.JID).Msg("Erro ao criar chat")
		return fmt.Errorf("erro ao criar chat: %w", err)
	}

	r.logger.Info().Str("chat_jid", c.JID).Str("session_id", c.SessionID.String()).Msg("Chat criado com sucesso")
	return nil
}

// GetByID busca um chat pelo ID
func (r *ChatRepository) GetByID(ctx context.Context, id uuid.UUID) (*chat.Chat, error) {
	query := `
		SELECT id, session_id, jid, name, type, last_message_time,
			   message_count, unread_count, is_muted, is_pinned, is_archived,
			   metadata, created_at, updated_at
		FROM zapcore_chats
		WHERE id = $1
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanChat(row)
}

// GetByJID busca um chat pelo JID e session ID
func (r *ChatRepository) GetByJID(ctx context.Context, sessionID uuid.UUID, jid string) (*chat.Chat, error) {
	query := `
		SELECT id, session_id, jid, name, type, last_message_time,
			   message_count, unread_count, is_muted, is_pinned, is_archived,
			   metadata, created_at, updated_at
		FROM zapcore_chats
		WHERE session_id = $1 AND jid = $2
	`

	row := r.db.QueryRowContext(ctx, query, sessionID, jid)
	return r.scanChat(row)
}

// Update atualiza um chat
func (r *ChatRepository) Update(ctx context.Context, c *chat.Chat) error {
	query := `
		UPDATE zapcore_chats 
		SET name = $3, type = $4, last_message_time = $5, message_count = $6,
			unread_count = $7, is_muted = $8, is_pinned = $9, is_archived = $10,
			metadata = $11, updated_at = $12
		WHERE session_id = $1 AND jid = $2
	`

	metadataJSON, err := json.Marshal(c.Metadata)
	if err != nil {
		return fmt.Errorf("erro ao serializar metadata: %w", err)
	}

	c.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		c.SessionID,
		c.JID,
		c.Name,
		c.Type,
		c.LastMessageTime,
		c.MessageCount,
		c.UnreadCount,
		c.IsMuted,
		c.IsPinned,
		c.IsArchived,
		metadataJSON,
		c.UpdatedAt,
	)

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
func (r *ChatRepository) Delete(ctx context.Context, sessionID uuid.UUID, jid string) error {
	query := `DELETE FROM zapcore_chats WHERE session_id = $1 AND jid = $2`

	result, err := r.db.ExecContext(ctx, query, sessionID, jid)
	if err != nil {
		r.logger.Error().Err(err).Str("chat_jid", jid).Msg("Erro ao deletar chat")
		return fmt.Errorf("erro ao deletar chat: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return chat.ErrChatNotFound
	}

	r.logger.Info().Str("chat_jid", jid).Msg("Chat deletado com sucesso")
	return nil
}

// ListBySessionID lista chats por session ID com paginação
func (r *ChatRepository) ListBySessionID(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*chat.Chat, error) {
	query := `
		SELECT id, session_id, jid, name, type, last_message_time,
			   message_count, unread_count, is_muted, is_pinned, is_archived,
			   metadata, created_at, updated_at
		FROM zapcore_chats
		WHERE session_id = $1
		ORDER BY 
			CASE WHEN is_pinned THEN 0 ELSE 1 END,
			last_message_time DESC NULLS LAST
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID, limit, offset)
	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao listar chats")
		return nil, fmt.Errorf("erro ao listar chats: %w", err)
	}
	defer rows.Close()

	return r.scanChats(rows)
}

// CountBySessionID conta chats por session ID
func (r *ChatRepository) CountBySessionID(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	query := `SELECT COUNT(*) FROM zapcore_chats WHERE session_id = $1`

	var count int64
	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(&count)
	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao contar chats")
		return 0, fmt.Errorf("erro ao contar chats: %w", err)
	}

	return count, nil
}

// UpdateUnreadCount atualiza apenas o contador de mensagens não lidas
func (r *ChatRepository) UpdateUnreadCount(ctx context.Context, sessionID uuid.UUID, jid string, count int) error {
	query := `
		UPDATE zapcore_chats 
		SET unread_count = $3, updated_at = NOW()
		WHERE session_id = $1 AND jid = $2
	`

	result, err := r.db.ExecContext(ctx, query, sessionID, jid, count)
	if err != nil {
		r.logger.Error().Err(err).Str("chat_jid", jid).Msg("Erro ao atualizar contador de não lidas")
		return fmt.Errorf("erro ao atualizar contador de não lidas: %w", err)
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

// Upsert cria ou atualiza um chat
func (r *ChatRepository) Upsert(ctx context.Context, c *chat.Chat) error {
	query := `
		INSERT INTO zapcore_chats (
			id, session_id, jid, name, type, last_message_time,
			message_count, unread_count, is_muted, is_pinned, is_archived,
			metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (session_id, jid) 
		DO UPDATE SET
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			last_message_time = EXCLUDED.last_message_time,
			message_count = EXCLUDED.message_count,
			unread_count = EXCLUDED.unread_count,
			is_muted = EXCLUDED.is_muted,
			is_pinned = EXCLUDED.is_pinned,
			is_archived = EXCLUDED.is_archived,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`

	metadataJSON, err := json.Marshal(c.Metadata)
	if err != nil {
		return fmt.Errorf("erro ao serializar metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		c.ID,
		c.SessionID,
		c.JID,
		c.Name,
		c.Type,
		c.LastMessageTime,
		c.MessageCount,
		c.UnreadCount,
		c.IsMuted,
		c.IsPinned,
		c.IsArchived,
		metadataJSON,
		c.CreatedAt,
		c.UpdatedAt,
	)

	if err != nil {
		r.logger.Error().Err(err).Str("chat_jid", c.JID).Msg("Erro ao fazer upsert do chat")
		return fmt.Errorf("erro ao fazer upsert do chat: %w", err)
	}

	return nil
}

// scanChat converte uma linha do banco em uma entidade Chat
func (r *ChatRepository) scanChat(row *sql.Row) (*chat.Chat, error) {
	var c chat.Chat
	var metadataJSON []byte

	err := row.Scan(
		&c.ID,
		&c.SessionID,
		&c.JID,
		&c.Name,
		&c.Type,
		&c.LastMessageTime,
		&c.MessageCount,
		&c.UnreadCount,
		&c.IsMuted,
		&c.IsPinned,
		&c.IsArchived,
		&metadataJSON,
		&c.CreatedAt,
		&c.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, chat.ErrChatNotFound
		}
		return nil, fmt.Errorf("erro ao fazer scan do chat: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &c.Metadata); err != nil {
			r.logger.Warn().Err(err).Msg("Erro ao deserializar metadata do chat")
			c.Metadata = make(map[string]any)
		}
	} else {
		c.Metadata = make(map[string]any)
	}

	return &c, nil
}

// scanChats converte múltiplas linhas do banco em entidades Chat
func (r *ChatRepository) scanChats(rows *sql.Rows) ([]*chat.Chat, error) {
	var chats []*chat.Chat

	for rows.Next() {
		var c chat.Chat
		var metadataJSON []byte

		err := rows.Scan(
			&c.ID,
			&c.SessionID,
			&c.JID,
			&c.Name,
			&c.Type,
			&c.LastMessageTime,
			&c.MessageCount,
			&c.UnreadCount,
			&c.IsMuted,
			&c.IsPinned,
			&c.IsArchived,
			&metadataJSON,
			&c.CreatedAt,
			&c.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("erro ao fazer scan do chat: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &c.Metadata); err != nil {
				r.logger.Warn().Err(err).Msg("Erro ao deserializar metadata do chat")
				c.Metadata = make(map[string]any)
			}
		} else {
			c.Metadata = make(map[string]any)
		}

		chats = append(chats, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar sobre os chats: %w", err)
	}

	return chats, nil
}
