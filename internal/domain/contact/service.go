package contact

import (
	"context"

	"github.com/google/uuid"
)

// Service define a interface para o serviço de contatos
type Service interface {
	// GetOrCreate obtém um contato existente ou cria um novo
	GetOrCreate(ctx context.Context, sessionID uuid.UUID, jid string) (*Contact, error)

	// UpdateInfo atualiza informações do contato
	UpdateInfo(ctx context.Context, sessionID uuid.UUID, jid string, info *ContactInfo) error

	// UpdatePresence atualiza a presença do contato
	UpdatePresence(ctx context.Context, sessionID uuid.UUID, jid string, isOnline bool) error

	// Search busca contatos por nome ou JID
	Search(ctx context.Context, sessionID uuid.UUID, query string, filters ListFilters) ([]*Contact, error)

	// GetBusinessContacts obtém apenas contatos business
	GetBusinessContacts(ctx context.Context, sessionID uuid.UUID, filters ListFilters) ([]*Contact, error)

	// GetGroupContacts obtém apenas grupos
	GetGroupContacts(ctx context.Context, sessionID uuid.UUID, filters ListFilters) ([]*Contact, error)

	// SyncContacts sincroniza contatos com o WhatsApp
	SyncContacts(ctx context.Context, sessionID uuid.UUID) error

	// GetProfilePicture obtém a foto de perfil do contato
	GetProfilePicture(ctx context.Context, sessionID uuid.UUID, jid string) (string, error)

	// Block bloqueia um contato
	Block(ctx context.Context, sessionID uuid.UUID, jid string) error

	// Unblock desbloqueia um contato
	Unblock(ctx context.Context, sessionID uuid.UUID, jid string) error

	// IsBlocked verifica se um contato está bloqueado
	IsBlocked(ctx context.Context, sessionID uuid.UUID, jid string) (bool, error)
}

// ContactInfo representa informações atualizáveis de um contato
type ContactInfo struct {
	Name         string `json:"name,omitempty"`
	PushName     string `json:"push_name,omitempty"`
	BusinessName string `json:"business_name,omitempty"`
	AvatarURL    string `json:"avatar_url,omitempty"`
	IsBusiness   *bool  `json:"is_business,omitempty"`
	IsGroup      *bool  `json:"is_group,omitempty"`
}
