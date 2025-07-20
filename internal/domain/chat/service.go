package chat

import (
	"context"

	"github.com/google/uuid"
)

// Service define a interface para o serviço de chats
type Service interface {
	// GetOrCreate obtém um chat existente ou cria um novo
	GetOrCreate(ctx context.Context, sessionID uuid.UUID, jid string, chatType ChatType) (*Chat, error)

	// UpdateLastMessage atualiza a última mensagem do chat
	UpdateLastMessage(ctx context.Context, sessionID uuid.UUID, jid string) error

	// MarkAsRead marca todas as mensagens do chat como lidas
	MarkAsRead(ctx context.Context, sessionID uuid.UUID, jid string) error

	// Archive arquiva um chat
	Archive(ctx context.Context, sessionID uuid.UUID, jid string) error

	// Unarchive desarquiva um chat
	Unarchive(ctx context.Context, sessionID uuid.UUID, jid string) error

	// Mute silencia um chat
	Mute(ctx context.Context, sessionID uuid.UUID, jid string) error

	// Unmute remove o silenciamento de um chat
	Unmute(ctx context.Context, sessionID uuid.UUID, jid string) error

	// Pin fixa um chat
	Pin(ctx context.Context, sessionID uuid.UUID, jid string) error

	// Unpin remove a fixação de um chat
	Unpin(ctx context.Context, sessionID uuid.UUID, jid string) error

	// GetActiveChats obtém chats ativos (não arquivados)
	GetActiveChats(ctx context.Context, sessionID uuid.UUID, filters ListFilters) ([]*Chat, error)

	// GetArchivedChats obtém chats arquivados
	GetArchivedChats(ctx context.Context, sessionID uuid.UUID, filters ListFilters) ([]*Chat, error)

	// GetUnreadCount obtém o total de chats não lidos
	GetUnreadCount(ctx context.Context, sessionID uuid.UUID) (int, error)

	// SyncChats sincroniza chats com o WhatsApp
	SyncChats(ctx context.Context, sessionID uuid.UUID) error
}

