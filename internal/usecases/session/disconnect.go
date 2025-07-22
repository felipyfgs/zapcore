package session

import (
	"context"
	"fmt"

	"zapcore/internal/domain/session"
	"zapcore/internal/domain/whatsapp"
	"zapcore/pkg/logger"

	"github.com/google/uuid"
)

// DisconnectUseCase representa o caso de uso para desconectar sessão
type DisconnectUseCase struct {
	sessionRepo    session.Repository
	whatsappClient whatsapp.Client
	logger         *logger.Logger
}

// NewDisconnectUseCase cria uma nova instância do caso de uso
func NewDisconnectUseCase(sessionRepo session.Repository, whatsappClient whatsapp.Client) *DisconnectUseCase {
	return &DisconnectUseCase{
		sessionRepo:    sessionRepo,
		whatsappClient: whatsappClient,
		logger:         logger.Get(),
	}
}

// DisconnectRequest representa a requisição para desconectar sessão
type DisconnectRequest struct {
	SessionID uuid.UUID `json:"sessionId" validate:"required"`
}

// DisconnectResponse representa a resposta da desconexão de sessão
type DisconnectResponse struct {
	SessionID uuid.UUID                     `json:"sessionId"`
	Status    session.WhatsAppSessionStatus `json:"status"`
	Message   string                        `json:"message"`
}

// Execute executa o caso de uso de desconexão de sessão
func (uc *DisconnectUseCase) Execute(ctx context.Context, req *DisconnectRequest) (*DisconnectResponse, error) {
	// Buscar sessão
	sess, err := uc.sessionRepo.GetByID(ctx, req.SessionID)
	if err != nil {
		if err == session.ErrSessionNotFound {
			return nil, err
		}
		uc.logger.Error().Err(err).Msg("erro ao buscar sessão")
		return nil, fmt.Errorf("erro interno do servidor")
	}

	// Verificar se já está desconectada
	if sess.Status == session.WhatsAppStatusDisconnected {
		uc.logger.Info().Str("session_id", fmt.Sprintf("%v", req.SessionID)).Msg("Sessão já está desconectada")
		return &DisconnectResponse{
			SessionID: sess.ID,
			Status:    sess.Status,
			Message:   "Sessão já está desconectada",
		}, nil
	}

	// Tentar desconectar do WhatsApp
	err = uc.whatsappClient.Disconnect(ctx, req.SessionID)
	if err != nil {
		uc.logger.Error().Err(err).Str("session_id", fmt.Sprintf("%v", req.SessionID)).Msg("Erro ao desconectar do WhatsApp")
		// Continua mesmo com erro para atualizar o status local
	}

	// Atualizar status da sessão
	sess.UpdateStatus(session.WhatsAppStatusDisconnected)
	sess.SetQRCode("") // Limpar QR Code

	if err := uc.sessionRepo.Update(ctx, sess); err != nil {
		uc.logger.Error().Err(err).Msg("erro ao atualizar sessão")
		return nil, fmt.Errorf("erro ao atualizar sessão: %w", err)
	}

	uc.logger.Info().Str("session_id", fmt.Sprintf("%v", req.SessionID)).Msg("Sessão desconectada com sucesso")

	return &DisconnectResponse{
		SessionID: sess.ID,
		Status:    sess.Status,
		Message:   "Sessão desconectada com sucesso",
	}, nil
}
