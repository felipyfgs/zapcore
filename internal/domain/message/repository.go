package message

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository define a interface para persistência de mensagens
type Repository interface {
	// Create cria uma nova mensagem
	Create(ctx context.Context, message *Message) error

	// GetByID busca uma mensagem pelo ID
	GetByID(ctx context.Context, id uuid.UUID) (*Message, error)

	// GetByMessageID busca uma mensagem pelo MessageID do WhatsApp
	GetByMessageID(ctx context.Context, messageID string) (*Message, error)

	// ExistsByMsgID verifica se uma mensagem já existe pelo msgId
	ExistsByMsgID(ctx context.Context, msgID string) (bool, error)

	// ExistsByMsgIDAndSessionID verifica se uma mensagem já existe pelo msgId e sessionId
	ExistsByMsgIDAndSessionID(ctx context.Context, msgID string, sessionID uuid.UUID) (bool, error)

	// List retorna mensagens com filtros opcionais
	List(ctx context.Context, filters ListFilters) ([]*Message, error)

	// Update atualiza uma mensagem existente
	Update(ctx context.Context, message *Message) error

	// Delete remove uma mensagem
	Delete(ctx context.Context, id uuid.UUID) error

	// UpdateStatus atualiza apenas o status de uma mensagem
	UpdateStatus(ctx context.Context, messageID string, status MessageStatus) error

	// GetBySessionID retorna mensagens de uma sessão específica
	GetBySessionID(ctx context.Context, sessionID uuid.UUID, filters ListFilters) ([]*Message, error)

	// GetConversation retorna mensagens de uma conversa específica
	GetConversation(ctx context.Context, sessionID uuid.UUID, jid string, filters ListFilters) ([]*Message, error)

	// CountByStatus conta mensagens por status
	CountByStatus(ctx context.Context, sessionID uuid.UUID, status MessageStatus) (int, error)

	// GetPendingMessages retorna mensagens pendentes para reenvio
	GetPendingMessages(ctx context.Context, sessionID uuid.UUID) ([]*Message, error)
}

// ListFilters define os filtros para listagem de mensagens
type ListFilters struct {
	SessionID *uuid.UUID        `json:"session_id,omitempty"`
	Type      *MessageType      `json:"type,omitempty"`
	Direction *MessageDirection `json:"direction,omitempty"`
	Status    *MessageStatus    `json:"status,omitempty"`
	FromJID   string            `json:"from_jid,omitempty"`
	ToJID     string            `json:"to_jid,omitempty"`
	DateFrom  *time.Time        `json:"date_from,omitempty"`
	DateTo    *time.Time        `json:"date_to,omitempty"`
	Limit     int               `json:"limit,omitempty"`
	Offset    int               `json:"offset,omitempty"`
	OrderBy   string            `json:"order_by,omitempty"`
	OrderDir  string            `json:"order_dir,omitempty"`
}

// DefaultListFilters retorna os filtros padrão para listagem
func DefaultListFilters() ListFilters {
	return ListFilters{
		Limit:    50,
		Offset:   0,
		OrderBy:  "timestamp",
		OrderDir: "DESC",
	}
}
