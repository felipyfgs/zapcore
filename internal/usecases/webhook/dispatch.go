package webhook

import (
	"context"
	"fmt"

	"zapcore/internal/domain/webhook"
	"zapcore/pkg/logger"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DispatchUseCase representa o caso de uso para despachar webhooks
type DispatchUseCase struct {
	webhookRepo    webhook.Repository
	webhookService webhook.Service
	logger         *logger.Logger
}

// NewDispatchUseCase cria uma nova instância do caso de uso
func NewDispatchUseCase(
	webhookRepo webhook.Repository,
	webhookService webhook.Service,
	zeroLogger zerolog.Logger,
) *DispatchUseCase {
	return &DispatchUseCase{
		webhookRepo:    webhookRepo,
		webhookService: webhookService,
		logger:         logger.NewFromZerolog(zeroLogger),
	}
}

// DispatchRequest representa a requisição para despachar webhook
type DispatchRequest struct {
	SessionID uuid.UUID         `json:"session_id" validate:"required"`
	EventType webhook.EventType `json:"event_type" validate:"required"`
	URL       string            `json:"url" validate:"required,url"`
	Payload   map[string]any    `json:"payload" validate:"required"`
}

// DispatchResponse representa a resposta do despacho de webhook
type DispatchResponse struct {
	EventID uuid.UUID `json:"event_id"`
	Message string    `json:"message"`
}

// Execute executa o caso de uso de despacho de webhook
func (uc *DispatchUseCase) Execute(ctx context.Context, req *DispatchRequest) (*DispatchResponse, error) {
	// Criar evento de webhook
	event := webhook.NewWebhookEvent(req.SessionID, req.EventType, req.URL, req.Payload)

	// Salvar evento no repositório
	if err := uc.webhookRepo.Create(ctx, event); err != nil {
		uc.logger.Error().Err(err).Msg("Erro ao salvar evento de webhook")
		return nil, fmt.Errorf("erro ao salvar evento: %w", err)
	}

	// Enviar webhook de forma assíncrona
	if err := uc.webhookService.SendAsync(ctx, event); err != nil {
		uc.logger.Error().Err(err).Msg("Erro ao enviar webhook")
		return nil, fmt.Errorf("erro ao enviar webhook: %w", err)
	}

	uc.logger.Info().Msg("webhook despachado com sucesso")

	return &DispatchResponse{
		EventID: event.ID,
		Message: "Webhook despachado com sucesso",
	}, nil
}

// ProcessPendingUseCase representa o caso de uso para processar webhooks pendentes
type ProcessPendingUseCase struct {
	webhookService webhook.Service
	logger         *logger.Logger
}

// NewProcessPendingUseCase cria uma nova instância do caso de uso
func NewProcessPendingUseCase(webhookService webhook.Service, zeroLogger zerolog.Logger) *ProcessPendingUseCase {
	return &ProcessPendingUseCase{
		webhookService: webhookService,
		logger:         logger.NewFromZerolog(zeroLogger),
	}
}

// ProcessPendingResponse representa a resposta do processamento de webhooks pendentes
type ProcessPendingResponse struct {
	ProcessedCount int    `json:"processed_count"`
	Message        string `json:"message"`
}

// Execute executa o caso de uso de processamento de webhooks pendentes
func (uc *ProcessPendingUseCase) Execute(ctx context.Context) (*ProcessPendingResponse, error) {
	// Processar eventos pendentes
	if err := uc.webhookService.ProcessPendingEvents(ctx); err != nil {
		uc.logger.Error().Err(err).Msg("erro ao processar webhooks pendentes")
		return nil, fmt.Errorf("erro ao processar webhooks pendentes: %w", err)
	}

	uc.logger.Info().Msg("Webhooks pendentes processados com sucesso")

	return &ProcessPendingResponse{
		Message: "Webhooks pendentes processados com sucesso",
	}, nil
}

// RetryUseCase representa o caso de uso para reprocessar webhook com falha
type RetryUseCase struct {
	webhookService webhook.Service
	logger         *logger.Logger
}

// NewRetryUseCase cria uma nova instância do caso de uso
func NewRetryUseCase(webhookService webhook.Service, zeroLogger zerolog.Logger) *RetryUseCase {
	return &RetryUseCase{
		webhookService: webhookService,
		logger:         logger.NewFromZerolog(zeroLogger),
	}
}

// RetryRequest representa a requisição para reprocessar webhook
type RetryRequest struct {
	EventID uuid.UUID `json:"event_id" validate:"required"`
}

// RetryResponse representa a resposta do reprocessamento de webhook
type RetryResponse struct {
	EventID uuid.UUID `json:"event_id"`
	Message string    `json:"message"`
}

// Execute executa o caso de uso de reprocessamento de webhook
func (uc *RetryUseCase) Execute(ctx context.Context, req *RetryRequest) (*RetryResponse, error) {
	// Reprocessar webhook
	if err := uc.webhookService.Retry(ctx, req.EventID); err != nil {
		uc.logger.Error().Err(err).Str("event_id", fmt.Sprintf("%v", req.EventID)).Msg("Erro ao reprocessar webhook")
		return nil, fmt.Errorf("erro ao reprocessar webhook: %w", err)
	}

	uc.logger.Info().Str("event_id", fmt.Sprintf("%v", req.EventID)).Msg("Webhook reprocessado com sucesso")

	return &RetryResponse{
		EventID: req.EventID,
		Message: "Webhook reprocessado com sucesso",
	}, nil
}
