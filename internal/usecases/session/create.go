package session

import (
	"context"
	"fmt"

	"zapcore/internal/domain/session"

	"github.com/rs/zerolog"
)

// CreateUseCase representa o caso de uso para criar sessão
type CreateUseCase struct {
	sessionRepo session.Repository
	logger      zerolog.Logger
}

// NewCreateUseCase cria uma nova instância do caso de uso
func NewCreateUseCase(sessionRepo session.Repository, logger zerolog.Logger) *CreateUseCase {
	return &CreateUseCase{
		sessionRepo: sessionRepo,
		logger:      logger,
	}
}

// CreateRequest representa a requisição para criar sessão
type CreateRequest struct {
	Name    string `json:"name" validate:"required,min=3,max=50"`
	Webhook string `json:"webhook,omitempty" validate:"omitempty,url"`
}

// CreateResponse representa a resposta da criação de sessão
type CreateResponse struct {
	Session *session.Session `json:"session"`
	Message string           `json:"message"`
}

// Execute executa o caso de uso de criação de sessão
func (uc *CreateUseCase) Execute(ctx context.Context, req *CreateRequest) (*CreateResponse, error) {
	// Validar se já existe uma sessão com o mesmo nome
	existingSession, err := uc.sessionRepo.GetByName(ctx, req.Name)
	if err != nil && err != session.ErrSessionNotFound {
		uc.logger.Error().Err(err).Msg("Erro ao verificar sessão existente")
		return nil, fmt.Errorf("erro interno do servidor")
	}

	if existingSession != nil {
		uc.logger.Warn().Str("name", req.Name).Msg("Tentativa de criar sessão com nome duplicado")
		return nil, session.ErrSessionAlreadyExists
	}

	// Criar nova sessão
	newSession := session.NewSession(req.Name)

	// Configurar webhook se fornecido
	if req.Webhook != "" {
		newSession.SetWebhook(req.Webhook)
	}

	// Salvar no repositório
	if err := uc.sessionRepo.Create(ctx, newSession); err != nil {
		uc.logger.Error().Err(err).Str("session_name", req.Name).Msg("Erro ao criar sessão")
		return nil, fmt.Errorf("erro ao criar sessão: %w", err)
	}

	uc.logger.Info().
		Str("session_id", newSession.ID.String()).
		Str("session_name", newSession.Name).
		Msg("Sessão criada com sucesso")

	return &CreateResponse{
		Session: newSession,
		Message: "Sessão criada com sucesso",
	}, nil
}
