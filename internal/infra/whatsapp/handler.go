package whatsapp

import (
	"fmt"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"wamex/pkg/logger"
)

// EventHandler gerencia eventos do WhatsApp
type EventHandler struct {
	client      *Client
	sessionName string
}

// NewEventHandler cria um novo handler de eventos
func NewEventHandler(client *Client, sessionName string) *EventHandler {
	return &EventHandler{
		client:      client,
		sessionName: sessionName,
	}
}

// RegisterHandlers registra todos os handlers de eventos
func (h *EventHandler) RegisterHandlers() {
	h.client.AddEventHandler(h.handleEvent)
}

// handleEvent processa todos os eventos do WhatsApp
func (h *EventHandler) handleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Connected:
		h.handleConnected(v)
	case *events.Disconnected:
		h.handleDisconnected(v)
	case *events.LoggedOut:
		h.handleLoggedOut(v)
	case *events.QR:
		h.handleQR(v)
	case *events.PairSuccess:
		h.handlePairSuccess(v)
	case *events.Message:
		h.handleMessage(v)
	case *events.Receipt:
		h.handleReceipt(v)
	case *events.Presence:
		h.handlePresence(v)
	case *events.ChatPresence:
		h.handleChatPresence(v)
	case *events.HistorySync:
		h.handleHistorySync(v)
	default:
		// Log eventos não tratados para debug
		logger.WithComponent("whatsapp").Debug().
			Str("session_name", h.sessionName).
			Str("event_type", fmt.Sprintf("%T", v)).
			Msg("Unhandled WhatsApp event")
	}
}

// handleConnected processa evento de conexão
func (h *EventHandler) handleConnected(evt *events.Connected) {
	h.client.Connected = true

	logger.WithComponent("whatsapp").Info().
		Str("session_name", h.sessionName).
		Msg("WhatsApp connected")
}

// handleDisconnected processa evento de desconexão
func (h *EventHandler) handleDisconnected(evt *events.Disconnected) {
	h.client.Connected = false

	logger.WithComponent("whatsapp").Warn().
		Str("session_name", h.sessionName).
		Msg("WhatsApp disconnected")
}

// handleLoggedOut processa evento de logout
func (h *EventHandler) handleLoggedOut(evt *events.LoggedOut) {
	h.client.Connected = false

	logger.WithComponent("whatsapp").Warn().
		Str("session_name", h.sessionName).
		Str("reason", evt.Reason.String()).
		Msg("WhatsApp logged out")
}

// handleQR processa evento de QR code
func (h *EventHandler) handleQR(evt *events.QR) {
	logger.WithComponent("whatsapp").Info().
		Str("session_name", h.sessionName).
		Str("qr_code", evt.Codes[0]).
		Msg("QR code generated")
}

// handlePairSuccess processa evento de pareamento bem-sucedido
func (h *EventHandler) handlePairSuccess(evt *events.PairSuccess) {
	logger.WithComponent("whatsapp").Info().
		Str("session_name", h.sessionName).
		Str("jid", evt.ID.String()).
		Msg("Phone pairing successful")
}

// handleMessage processa mensagens recebidas
func (h *EventHandler) handleMessage(evt *events.Message) {
	logger.WithComponent("whatsapp").Debug().
		Str("session_name", h.sessionName).
		Str("from", evt.Info.Sender.String()).
		Str("message_id", evt.Info.ID).
		Msg("Message received")

	// Aqui você pode implementar lógica para processar mensagens recebidas
	// Por exemplo, salvar no banco de dados, enviar webhooks, etc.
}

// handleReceipt processa confirmações de entrega
func (h *EventHandler) handleReceipt(evt *events.Receipt) {
	logger.WithComponent("whatsapp").Debug().
		Str("session_name", h.sessionName).
		Str("chat", evt.Chat.String()).
		Str("type", string(evt.Type)).
		Msg("Receipt received")
}

// handlePresence processa eventos de presença
func (h *EventHandler) handlePresence(evt *events.Presence) {
	logger.WithComponent("whatsapp").Debug().
		Str("session_name", h.sessionName).
		Str("from", evt.From.String()).
		Msg("Presence update")
}

// handleChatPresence processa eventos de presença em chat
func (h *EventHandler) handleChatPresence(evt *events.ChatPresence) {
	logger.WithComponent("whatsapp").Debug().
		Str("session_name", h.sessionName).
		Str("chat", evt.Chat.String()).
		Str("state", string(evt.State)).
		Msg("Chat presence update")
}

// handleHistorySync processa sincronização de histórico
func (h *EventHandler) handleHistorySync(evt *events.HistorySync) {
	logger.WithComponent("whatsapp").Info().
		Str("session_name", h.sessionName).
		Int("conversations", len(evt.Data.Conversations)).
		Msg("History sync received")
}

// SendMessage envia uma mensagem de texto
func (h *EventHandler) SendMessage(to, message string) error {
	jid, err := h.parseJID(to)
	if err != nil {
		return fmt.Errorf("invalid JID: %w", err)
	}

	// TODO: Implementar SendMessage com a nova API do whatsmeow
	// _, err = h.client.SendMessage(context.Background(), jid, message)
	_ = jid // Evitar erro de variável não utilizada
	err = fmt.Errorf("SendMessage not implemented yet - needs whatsmeow API update")

	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	logger.WithComponent("whatsapp").Info().
		Str("session_name", h.sessionName).
		Str("to", to).
		Msg("Message sent")

	return nil
}

// parseJID converte string para JID
func (h *EventHandler) parseJID(jidStr string) (types.JID, error) {
	// Implementar parsing de JID
	// Por exemplo: "5511999999999@s.whatsapp.net"
	return types.ParseJID(jidStr)
}

// IsConnected verifica se o cliente está conectado
func (h *EventHandler) IsConnected() bool {
	return h.client.Connected && h.client.IsConnected()
}

// GetDeviceJID retorna o JID do dispositivo
func (h *EventHandler) GetDeviceJID() string {
	if h.client.Store.ID != nil {
		return h.client.Store.ID.String()
	}
	return ""
}
