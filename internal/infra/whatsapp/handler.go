package whatsapp

import (
	"context"

	"zapcore/internal/domain/session"
	"zapcore/pkg/logger"

	"github.com/google/uuid"
)

// Constantes para padronização de logging
const (
	LogComponentHandler = "handler"
)

// SessionRepository define a interface para operações de sessão
type SessionRepository interface {
	UpdateJID(ctx context.Context, sessionID uuid.UUID, jid string) error
	UpdateStatus(ctx context.Context, sessionID uuid.UUID, status session.WhatsAppSessionStatus) error
}

// SessionEventHandler implementa o EventHandler para capturar eventos de sessão
type SessionEventHandler struct {
	sessionRepo SessionRepository
	logger      *logger.Logger
}

// NewSessionEventHandler cria uma nova instância do event handler
func NewSessionEventHandler(sessionRepo SessionRepository) *SessionEventHandler {
	return &SessionEventHandler{
		sessionRepo: sessionRepo,
		logger:      logger.Get(),
	}
}

// ConnectedEvent representa o evento de conexão estabelecida
type ConnectedEvent struct {
	SessionID uuid.UUID `json:"sessionId"`
}

// DisconnectedEvent representa o evento de desconexão
type DisconnectedEvent struct {
	SessionID uuid.UUID `json:"sessionId"`
	Reason    string    `json:"reason"`
}

// HandleEvent processa eventos do WhatsApp
func (h *SessionEventHandler) HandleEvent(sessionID uuid.UUID, event any) {
	ctx := context.Background()

	switch e := event.(type) {
	case *PairSuccessEvent:
		h.handlePairSuccess(ctx, sessionID, e)
	case *ConnectedEvent:
		h.handleConnected(ctx, sessionID, e)
	case *DisconnectedEvent:
		h.handleDisconnected(ctx, sessionID, e)
	default:
		// Outros eventos podem ser processados aqui no futuro
		h.logger.Debug().
			Str("session_id", sessionID.String()).
			Str("event_type", getEventType(event)).
			Str("component", LogComponentHandler).
			Msg("Evento WhatsApp recebido")
	}
}

// handlePairSuccess processa o evento de pareamento bem-sucedido
func (h *SessionEventHandler) handlePairSuccess(ctx context.Context, sessionID uuid.UUID, event *PairSuccessEvent) {

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
func (h *SessionEventHandler) handleConnected(ctx context.Context, sessionID uuid.UUID, _ *ConnectedEvent) {

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
func (h *SessionEventHandler) handleDisconnected(ctx context.Context, sessionID uuid.UUID, event *DisconnectedEvent) {

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
func getEventType(event any) string {
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

// CompositeEventHandler combina múltiplos handlers de eventos
type CompositeEventHandler struct {
	sessionHandler *SessionEventHandler
	storageHandler *StorageHandler
	logger         *logger.Logger
}

// NewCompositeEventHandler cria um handler composto que processa eventos de sessão e storage
func NewCompositeEventHandler(
	sessionHandler *SessionEventHandler,
	storageHandler *StorageHandler,
) *CompositeEventHandler {
	return &CompositeEventHandler{
		sessionHandler: sessionHandler,
		storageHandler: storageHandler,
		logger:         logger.Get(),
	}
}

// HandleEvent processa eventos através de ambos os handlers
func (c *CompositeEventHandler) HandleEvent(sessionID uuid.UUID, event any) {
	ctx := context.Background()

	// Processar eventos de sessão primeiro
	if c.sessionHandler != nil {
		c.sessionHandler.HandleEvent(sessionID, event)
	}

	// Processar eventos de storage
	if c.storageHandler != nil {
		if err := c.storageHandler.HandleEvent(ctx, sessionID, event); err != nil {
			c.logger.Error().
				Err(err).
				Str("session_id", sessionID.String()).
				Str("event_type", getEventType(event)).
				Msg("Erro ao processar evento no storage handler")
		}
	}
}

// SetMediaDownloader configura o MediaDownloader no StorageHandler
func (c *CompositeEventHandler) SetMediaDownloader(sessionID uuid.UUID, mediaDownloader *MediaDownloader) {
	if c.storageHandler != nil {
		c.storageHandler.mediaDownloader = mediaDownloader
		c.logger.Info().
			Str("session_id", sessionID.String()).
			Msg("MediaDownloader configurado no StorageHandler")
	}
}
