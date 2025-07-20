package contact

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository define a interface para persistência de contatos
type Repository interface {
	// Create cria um novo contato
	Create(ctx context.Context, contact *Contact) error

	// GetByID busca um contato pelo ID
	GetByID(ctx context.Context, id uuid.UUID) (*Contact, error)

	// GetByJID busca um contato pelo JID
	GetByJID(ctx context.Context, sessionID uuid.UUID, jid string) (*Contact, error)

	// List retorna contatos com filtros opcionais
	List(ctx context.Context, filters ListFilters) ([]*Contact, error)

	// Update atualiza um contato existente
	Update(ctx context.Context, contact *Contact) error

	// Delete remove um contato
	Delete(ctx context.Context, id uuid.UUID) error

	// GetBySessionID retorna contatos de uma sessão específica
	GetBySessionID(ctx context.Context, sessionID uuid.UUID, filters ListFilters) ([]*Contact, error)

	// UpdateLastSeen atualiza o último acesso do contato
	UpdateLastSeen(ctx context.Context, sessionID uuid.UUID, jid string, lastSeen time.Time) error

	// Search busca contatos por nome ou JID
	Search(ctx context.Context, sessionID uuid.UUID, query string, filters ListFilters) ([]*Contact, error)

	// GetBusinessContacts retorna apenas contatos business
	GetBusinessContacts(ctx context.Context, sessionID uuid.UUID, filters ListFilters) ([]*Contact, error)

	// GetGroupContacts retorna apenas grupos
	GetGroupContacts(ctx context.Context, sessionID uuid.UUID, filters ListFilters) ([]*Contact, error)
}

// ListFilters define os filtros para listagem de contatos
type ListFilters struct {
	SessionID  *uuid.UUID `json:"session_id,omitempty"`
	IsBusiness *bool      `json:"is_business,omitempty"`
	IsGroup    *bool      `json:"is_group,omitempty"`
	Query      string     `json:"query,omitempty"`
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
		OrderBy:  "name",
		OrderDir: "ASC",
	}
}

