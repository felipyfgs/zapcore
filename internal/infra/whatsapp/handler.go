package whatsapp

import (
	"context"

	"zapcore/internal/domain/session"
	"zapcore/pkg/logger"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// SessionEventHandler implementa o EventHandler para capturar eventos de sessão
type SessionEventHandler struct {
	sessionRepo interface {
		UpdateJID(ctx context.Context, sessionID uuid.UUID, jid string) error
		UpdateStatus(ctx context.Context, sessionID uuid.UUID, status session.WhatsAppSessionStatus) error
	}
	logger *logger.Logger
}

// NewSessionEventHandler cria uma nova instância do event handler
func NewSessionEventHandler(sessionRepo interface {
	UpdateJID(ctx context.Context, sessionID uuid.UUID, jid string) error
	UpdateStatus(ctx context.Context, sessionID uuid.UUID, status session.WhatsAppSessionStatus) error
}, zeroLogger zerolog.Logger) *SessionEventHandler {
	return &SessionEventHandler{
		sessionRepo: sessionRepo,
		logger:      logger.NewFromZerolog(zeroLogger),
	}
}

// ConnectedEvent representa o evento de conexão estabelecida
type ConnectedEvent struct {
	SessionID uuid.UUID
}

// DisconnectedEvent representa o evento de desconexão
type DisconnectedEvent struct {
	SessionID uuid.UUID
	Reason    string
}

// HandleEvent processa eventos do WhatsApp
func (h *SessionEventHandler) HandleEvent(sessionID uuid.UUID, event interface{}) {
	switch e := event.(type) {
	case *PairSuccessEvent:
		h.handlePairSuccess(sessionID, e)
	case *ConnectedEvent:
		h.handleConnected(sessionID, e)
	case *DisconnectedEvent:
		h.handleDisconnected(sessionID, e)
	default:
		// Outros eventos podem ser processados aqui no futuro
		h.logger.Debug().
			Str("session_id", sessionID.String()).
			Str("event_type", getEventType(event)).
			Msg("Evento WhatsApp recebido")
	}
}

// handlePairSuccess processa o evento de pareamento bem-sucedido
func (h *SessionEventHandler) handlePairSuccess(sessionID uuid.UUID, event *PairSuccessEvent) {
	ctx := context.Background()
	
	// Salvar JID no banco de dados
	err := h.sessionRepo.UpdateJID(ctx, sessionID, event.JID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("session_id", sessionID.String()).
			Str("jid", event.JID).
			Msg("Erro ao salvar JID da sessão após pareamento")
		return
	}

	h.logger.Info().
		Str("session_id", sessionID.String()).
		Str("jid", event.JID).
		Str("business_name", event.BusinessName).
		Str("platform", event.Platform).
		Msg("JID da sessão salvo com sucesso após pareamento")
}

// handleConnected processa o evento de conexão estabelecida
func (h *SessionEventHandler) handleConnected(sessionID uuid.UUID, event *ConnectedEvent) {
	ctx := context.Background()

	// Atualizar status para "connected"
	err := h.sessionRepo.UpdateStatus(ctx, sessionID, session.WhatsAppStatusConnected)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("session_id", sessionID.String()).
			Msg("Erro ao atualizar status da sessão para connected")
		return
	}

	h.logger.Info().
		Str("session_id", sessionID.String()).
		Msg("Status da sessão atualizado para connected")
}

// handleDisconnected processa o evento de desconexão
func (h *SessionEventHandler) handleDisconnected(sessionID uuid.UUID, event *DisconnectedEvent) {
	ctx := context.Background()

	// Atualizar status para "disconnected"
	err := h.sessionRepo.UpdateStatus(ctx, sessionID, session.WhatsAppStatusDisconnected)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("session_id", sessionID.String()).
			Str("reason", event.Reason).
			Msg("Erro ao atualizar status da sessão para disconnected")
		return
	}

	h.logger.Info().
		Str("session_id", sessionID.String()).
		Str("reason", event.Reason).
		Msg("Status da sessão atualizado para disconnected")
}

// getEventType retorna o tipo do evento como string para logging
func getEventType(event interface{}) string {
	switch event.(type) {
	case *PairSuccessEvent:
		return "PairSuccess"
	case *ConnectedEvent:
		return "Connected"
	case *DisconnectedEvent:
		return "Disconnected"
	default:
		return "Unknown"
	}
}
