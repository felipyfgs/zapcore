package session

import (
	"context"
	"fmt"

	"zapcore/internal/domain/session"
	"zapcore/internal/domain/whatsapp"
	"zapcore/pkg/logger"

	"github.com/google/uuid"
)

// ConnectUseCase representa o caso de uso para conectar sessão
type ConnectUseCase struct {
	sessionRepo    session.Repository
	whatsappClient whatsapp.Client
	logger         *logger.Logger
}

// NewConnectUseCase cria uma nova instância do caso de uso
func NewConnectUseCase(sessionRepo session.Repository, whatsappClient whatsapp.Client) *ConnectUseCase {
	return &ConnectUseCase{
		sessionRepo:    sessionRepo,
		whatsappClient: whatsappClient,
		logger:         logger.Get(),
	}
}

// ConnectRequest representa a requisição para conectar sessão
type ConnectRequest struct {
	SessionID uuid.UUID `json:"sessionId" validate:"required"`
}

// ConnectResponse representa a resposta da conexão de sessão
type ConnectResponse struct {
	SessionID uuid.UUID                     `json:"sessionId"`
	Status    session.WhatsAppSessionStatus `json:"status"`
	QRCode    string                        `json:"qr_code,omitempty"`
	Message   string                        `json:"message"`
}

// Execute executa o caso de uso de conexão de sessão
func (uc *ConnectUseCase) Execute(ctx context.Context, req *ConnectRequest) (*ConnectResponse, error) {
	// Buscar sessão
	sess, err := uc.sessionRepo.GetByID(ctx, req.SessionID)
	if err != nil {
		if err == session.ErrSessionNotFound {
			return nil, err
		}
		uc.logger.Error().Err(err).Msg("Erro ao buscar sessão")
		return nil, fmt.Errorf("erro interno do servidor")
	}

	// Verificar se a sessão está ativa
	if !sess.IsActive {
		uc.logger.Warn().Str("session_id", req.SessionID.String()).Msg("Tentativa de conectar sessão inativa")
		return nil, session.ErrSessionNotActive
	}

	// Verificar se já está conectada
	if sess.IsConnected() {
		uc.logger.Info().Str("session_id", req.SessionID.String()).Msg("Sessão já está conectada")
		return &ConnectResponse{
			SessionID: sess.ID,
			Status:    sess.Status,
			Message:   "Sessão já está conectada",
		}, nil
	}

	// Verificar se pode conectar
	if !sess.CanConnect() {
		uc.logger.Warn().Str("session_id", req.SessionID.String()).Msg("Sessão não pode ser conectada no estado atual")
		return nil, fmt.Errorf("sessão não pode ser conectada no estado atual: %s", sess.Status)
	}

	// Atualizar status para connecting
	sess.UpdateStatus(session.WhatsAppStatusConnecting)
	if err := uc.sessionRepo.Update(ctx, sess); err != nil {
		uc.logger.Error().Err(err).Msg("Erro ao atualizar status da sessão")
		return nil, fmt.Errorf("erro ao atualizar sessão: %w", err)
	}

	// Tentar conectar com WhatsApp
	err = uc.whatsappClient.Connect(ctx, req.SessionID)
	if err != nil {
		// Atualizar status para disconnected em caso de erro
		sess.UpdateStatus(session.WhatsAppStatusDisconnected)
		uc.sessionRepo.Update(ctx, sess)

		uc.logger.Error().Err(err).Str("session_id", req.SessionID.String()).Msg("Erro ao conectar com WhatsApp")
		return nil, fmt.Errorf("erro ao conectar com WhatsApp: %w", err)
	}

	// Verificar se precisa de QR Code
	status, err := uc.whatsappClient.GetStatus(ctx, req.SessionID)
	if err != nil {
		uc.logger.Error().Err(err).Msg("Erro ao obter status do WhatsApp")
		return nil, fmt.Errorf("erro ao obter status: %w", err)
	}

	response := &ConnectResponse{
		SessionID: sess.ID,
		Status:    session.WhatsAppSessionStatus(status),
		Message:   "Conexão iniciada com sucesso",
	}

	// QR Code será exibido no terminal do servidor automaticamente
	// quando necessário durante o processo de conexão

	uc.logger.Info().
		Str("session_id", req.SessionID.String()).
		Str("status", string(status)).
		Msg("Sessão conectada com sucesso")

	return response, nil
}
