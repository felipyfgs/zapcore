package session

import (
	"context"

	"github.com/google/uuid"
)

// Repository define a interface para persistência de sessões
type Repository interface {
	// Create cria uma nova sessão
	Create(ctx context.Context, session *Session) error

	// GetByID busca uma sessão pelo ID
	GetByID(ctx context.Context, id uuid.UUID) (*Session, error)

	// GetByName busca uma sessão pelo nome
	GetByName(ctx context.Context, name string) (*Session, error)

	// List retorna todas as sessões com filtros opcionais
	List(ctx context.Context, filters ListFilters) ([]*Session, error)

	// Update atualiza uma sessão existente
	Update(ctx context.Context, session *Session) error

	// Delete remove uma sessão
	Delete(ctx context.Context, id uuid.UUID) error

	// GetActiveCount retorna o número de sessões ativas
	GetActiveCount(ctx context.Context) (int, error)

	// UpdateStatus atualiza apenas o status de uma sessão
	UpdateStatus(ctx context.Context, id uuid.UUID, status WhatsAppSessionStatus) error

	// UpdateLastSeen atualiza o último acesso de uma sessão
	UpdateLastSeen(ctx context.Context, id uuid.UUID) error
}

// ListFilters define os filtros para listagem de sessões
type ListFilters struct {
	Status   *WhatsAppSessionStatus `json:"status,omitempty"`
	IsActive *bool                  `json:"is_active,omitempty"`
	Limit    int                    `json:"limit,omitempty"`
	Offset   int                    `json:"offset,omitempty"`
	OrderBy  string                 `json:"order_by,omitempty"`
	OrderDir string                 `json:"order_dir,omitempty"`
}

// DefaultListFilters retorna os filtros padrão para listagem
func DefaultListFilters() ListFilters {
	return ListFilters{
		Limit:    50,
		Offset:   0,
		OrderBy:  "created_at",
		OrderDir: "DESC",
	}
}

