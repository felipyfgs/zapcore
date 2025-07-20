package session

import (
	"context"
	"fmt"

	"zapcore/internal/domain/session"
	"zapcore/pkg/logger"

	"github.com/rs/zerolog"
)

// ListUseCase representa o caso de uso para listar sessões
type ListUseCase struct {
	sessionRepo session.Repository
	logger      *logger.Logger
}

// NewListUseCase cria uma nova instância do caso de uso
func NewListUseCase(sessionRepo session.Repository, zeroLogger zerolog.Logger) *ListUseCase {
	return &ListUseCase{
		sessionRepo: sessionRepo,
		logger:      logger.NewFromZerolog(zeroLogger),
	}
}

// ListRequest representa a requisição para listar sessões
type ListRequest struct {
	Status   *session.WhatsAppSessionStatus `json:"status,omitempty"`
	IsActive *bool                          `json:"is_active,omitempty"`
	Limit    int                            `json:"limit,omitempty"`
	Offset   int                            `json:"offset,omitempty"`
	OrderBy  string                         `json:"order_by,omitempty"`
	OrderDir string                         `json:"order_dir,omitempty"`
}

// ListResponse representa a resposta da listagem de sessões
type ListResponse struct {
	Sessions []*session.Session `json:"sessions"`
	Total    int                `json:"total"`
	Limit    int                `json:"limit"`
	Offset   int                `json:"offset"`
}

// Execute executa o caso de uso de listagem de sessões
func (uc *ListUseCase) Execute(ctx context.Context, req *ListRequest) (*ListResponse, error) {
	// Preparar filtros
	filters := session.ListFilters{
		Status:   req.Status,
		IsActive: req.IsActive,
		Limit:    req.Limit,
		Offset:   req.Offset,
		OrderBy:  req.OrderBy,
		OrderDir: req.OrderDir,
	}

	// Aplicar valores padrão se não fornecidos
	if filters.Limit <= 0 {
		filters.Limit = 50
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}
	if filters.OrderBy == "" {
		filters.OrderBy = "created_at"
	}
	if filters.OrderDir == "" {
		filters.OrderDir = "DESC"
	}

	// Buscar sessões
	sessions, err := uc.sessionRepo.List(ctx, filters)
	if err != nil {
		uc.logger.Error().Err(err).Msg("Erro ao listar sessões")
		return nil, fmt.Errorf("erro ao listar sessões: %w", err)
	}

	// Para obter o total, fazer uma consulta sem limit/offset
	totalFilters := filters
	totalFilters.Limit = 0
	totalFilters.Offset = 0
	allSessions, err := uc.sessionRepo.List(ctx, totalFilters)
	if err != nil {
		uc.logger.Error().Err(err).Msg("Erro ao contar total de sessões")
		return nil, fmt.Errorf("erro ao contar sessões: %w", err)
	}

	uc.logger.Info().
		Int("count", len(sessions)).
		Int("total", len(allSessions)).
		Int("limit", filters.Limit).
		Int("offset", filters.Offset).
		Msg("Sessões listadas com sucesso")

	return &ListResponse{
		Sessions: sessions,
		Total:    len(allSessions),
		Limit:    filters.Limit,
		Offset:   filters.Offset,
	}, nil
}
