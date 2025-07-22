package chat

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository define a interface para persistência de chats
type Repository interface {
	// Create cria um novo chat
	Create(ctx context.Context, chat *Chat) error

	// GetByID busca um chat pelo ID
	GetByID(ctx context.Context, id uuid.UUID) (*Chat, error)

	// GetByJID busca um chat pelo JID
	GetByJID(ctx context.Context, sessionID uuid.UUID, jid string) (*Chat, error)

	// List retorna chats com filtros opcionais
	List(ctx context.Context, filters ListFilters) ([]*Chat, error)

	// Update atualiza um chat existente
	Update(ctx context.Context, chat *Chat) error

	// Delete remove um chat
	Delete(ctx context.Context, id uuid.UUID) error

	// GetBySessionID retorna chats de uma sessão específica
	GetBySessionID(ctx context.Context, sessionID uuid.UUID, filters ListFilters) ([]*Chat, error)

	// UpdateLastMessage atualiza a última mensagem do chat
	UpdateLastMessage(ctx context.Context, sessionID uuid.UUID, jid string, timestamp time.Time) error

	// IncrementMessageCount incrementa o contador de mensagens
	IncrementMessageCount(ctx context.Context, sessionID uuid.UUID, jid string) error

	// IncrementUnreadCount incrementa o contador de não lidas
	IncrementUnreadCount(ctx context.Context, sessionID uuid.UUID, jid string) error

	// MarkAsRead marca chat como lido
	MarkAsRead(ctx context.Context, sessionID uuid.UUID, jid string) error

	// GetUnreadCount retorna total de chats não lidos
	GetUnreadCount(ctx context.Context, sessionID uuid.UUID) (int, error)
}

// ListFilters define os filtros para listagem de chats
type ListFilters struct {
	SessionID  *uuid.UUID `json:"session_id,omitempty"`
	Type       *ChatType  `json:"type,omitempty"`
	IsMuted    *bool      `json:"is_muted,omitempty"`
	IsPinned   *bool      `json:"is_pinned,omitempty"`
	IsArchived *bool      `json:"is_archived,omitempty"`
	HasUnread  *bool      `json:"has_unread,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
	OrderBy    string     `json:"order_by,omitempty"`
	OrderDir   string     `json:"order_dir,omitempty"`
}

// DefaultListFilters retorna os filtros padrão para listagem
func DefaultListFilters() ListFilters {
	return ListFilters{
		Limit:    50,
		Offset:   0,
		OrderBy:  "last_message_time",
		OrderDir: "DESC",
	}
}
