package session

import (
	"context"
	"fmt"

	"zapcore/internal/domain/session"
	"zapcore/internal/domain/whatsapp"
	"zapcore/pkg/logger"

	"github.com/google/uuid"
)

// GetStatusUseCase representa o caso de uso para obter status da sessão
type GetStatusUseCase struct {
	sessionRepo    session.Repository
	whatsappClient whatsapp.Client
	logger         *logger.Logger
}

// NewGetStatusUseCase cria uma nova instância do caso de uso
func NewGetStatusUseCase(sessionRepo session.Repository, whatsappClient whatsapp.Client) *GetStatusUseCase {
	return &GetStatusUseCase{
		sessionRepo:    sessionRepo,
		whatsappClient: whatsappClient,
		logger:         logger.Get(),
	}
}

// GetStatusRequest representa a requisição para obter status
type GetStatusRequest struct {
	SessionID uuid.UUID `json:"sessionId" validate:"required"`
}

// GetStatusResponse representa a resposta do status da sessão
type GetStatusResponse struct {
	SessionID   uuid.UUID                     `json:"sessionId"`
	Name        string                        `json:"name"`
	Status      session.WhatsAppSessionStatus `json:"status"`
	JID         string                        `json:"jid,omitempty"`
	IsActive    bool                          `json:"isActive"`
	LastSeen    string                        `json:"last_seen,omitempty"`
	QRCode      string                        `json:"qr_code,omitempty"`
	IsConnected bool                          `json:"is_connected"`
	CanConnect  bool                          `json:"can_connect"`
	CreatedAt   string                        `json:"createdAt"`
	UpdatedAt   string                        `json:"updatedAt"`
}

// Execute executa o caso de uso de obtenção de status
func (uc *GetStatusUseCase) Execute(ctx context.Context, req *GetStatusRequest) (*GetStatusResponse, error) {
	// Buscar sessão
	sess, err := uc.sessionRepo.GetByID(ctx, req.SessionID)
	if err != nil {
		if err == session.ErrSessionNotFound {
			return nil, err
		}
		uc.logger.Error().Err(err).Msg("Erro ao buscar sessão")
		return nil, fmt.Errorf("erro interno do servidor")
	}

	// Tentar obter status atualizado do WhatsApp se a sessão estiver ativa
	if sess.IsActive {
		whatsappStatus, err := uc.whatsappClient.GetStatus(ctx, req.SessionID)
		if err != nil {
			uc.logger.Warn().Err(err).Str("session_id", req.SessionID.String()).Msg("Erro ao obter status do WhatsApp, usando status local")
		} else {
			// Atualizar status se diferente
			newStatus := session.WhatsAppSessionStatus(whatsappStatus)
			if sess.Status != newStatus {
				sess.UpdateStatus(newStatus)
				uc.sessionRepo.Update(ctx, sess)
			}
		}
	}

	// Preparar resposta
	response := &GetStatusResponse{
		SessionID:   sess.ID,
		Name:        sess.Name,
		Status:      sess.Status,
		JID:         sess.JID,
		IsActive:    sess.IsActive,
		QRCode:      sess.QRCode,
		IsConnected: sess.IsConnected(),
		CanConnect:  sess.CanConnect(),
		CreatedAt:   sess.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   sess.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if sess.LastSeen != nil {
		response.LastSeen = sess.LastSeen.Format("2006-01-02T15:04:05Z07:00")
	}

	uc.logger.Info().
		Str("session_id", req.SessionID.String()).
		Str("status", string(sess.Status)).
		Msg("Status da sessão obtido com sucesso")

	return response, nil
}

// GetByName busca uma sessão pelo nome
func (uc *GetStatusUseCase) GetByName(ctx context.Context, name string) (*session.Session, error) {
	sess, err := uc.sessionRepo.GetByName(ctx, name)
	if err != nil {
		if err == session.ErrSessionNotFound {
			return nil, err
		}
		uc.logger.Error().Err(err).Str("session_name", name).Msg("Erro ao buscar sessão por nome")
		return nil, fmt.Errorf("erro interno do servidor")
	}

	return sess, nil
}
