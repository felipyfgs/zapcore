package whatsapp

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EventHandler define a interface para manipulação de eventos
type EventHandler interface {
	HandleMessage(ctx context.Context, event *MessageEvent) error
	HandleReceipt(ctx context.Context, event *ReceiptEvent) error
	HandlePresence(ctx context.Context, event *PresenceEvent) error
	HandleConnection(ctx context.Context, event *ConnectionEvent) error
	HandleQRCode(ctx context.Context, event *QRCodeEvent) error
}

// MessageEvent representa um evento de mensagem recebida
type MessageEvent struct {
	SessionID uuid.UUID              `json:"session_id"`
	MessageID string                 `json:"message_id"`
	Type      string                 `json:"type"`
	FromJID   string                 `json:"from_jid"`
	ToJID     string                 `json:"to_jid"`
	Content   string                 `json:"content,omitempty"`
	MediaData []byte                 `json:"media_data,omitempty"`
	MediaType string                 `json:"media_type,omitempty"`
	Caption   string                 `json:"caption,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	IsFromMe  bool                   `json:"is_from_me"`
	IsGroup   bool                   `json:"is_group"`
	PushName  string                 `json:"push_name,omitempty"`
	ReplyToID string                 `json:"reply_to_id,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// ReceiptEvent representa um evento de confirmação de leitura
type ReceiptEvent struct {
	SessionID  uuid.UUID `json:"session_id"`
	MessageIDs []string  `json:"message_ids"`
	FromJID    string    `json:"from_jid"`
	Type       string    `json:"type"` // read, delivered
	Timestamp  time.Time `json:"timestamp"`
}

// PresenceEvent representa um evento de presença
type PresenceEvent struct {
	SessionID   uuid.UUID `json:"session_id"`
	FromJID     string    `json:"from_jid"`
	IsAvailable bool      `json:"is_available"`
	LastSeen    time.Time `json:"last_seen,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// ConnectionEvent representa um evento de conexão
type ConnectionEvent struct {
	SessionID uuid.UUID        `json:"session_id"`
	Status    ConnectionStatus `json:"status"`
	Error     string           `json:"error,omitempty"`
	Timestamp time.Time        `json:"timestamp"`
}

// QRCodeEvent representa um evento de QR Code
type QRCodeEvent struct {
	SessionID uuid.UUID `json:"session_id"`
	QRCode    string    `json:"qr_code"`
	Event     string    `json:"event"` // code, timeout, success
	Timestamp time.Time `json:"timestamp"`
}

