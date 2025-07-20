package session

import (
	"time"

	"github.com/google/uuid"
)

// WhatsAppSessionStatus representa os possíveis status de uma sessão WhatsApp
type WhatsAppSessionStatus string

const (
	WhatsAppStatusDisconnected WhatsAppSessionStatus = "disconnected"
	WhatsAppStatusConnecting   WhatsAppSessionStatus = "connecting"
	WhatsAppStatusConnected    WhatsAppSessionStatus = "connected"
	WhatsAppStatusQRCode       WhatsAppSessionStatus = "qr_code"
	WhatsAppStatusPairing      WhatsAppSessionStatus = "pairing"
	WhatsAppStatusError        WhatsAppSessionStatus = "error"
)

// Session representa uma sessão do WhatsApp
type Session struct {
	ID        uuid.UUID              `json:"id"`
	Name      string                 `json:"name"`
	Status    WhatsAppSessionStatus  `json:"status"`
	JID       string                 `json:"jid,omitempty"`
	QRCode    string                 `json:"qr_code,omitempty"`
	ProxyURL  string                 `json:"proxy_url,omitempty"`
	Webhook   string                 `json:"webhook,omitempty"`
	IsActive  bool                   `json:"is_active"`
	LastSeen  *time.Time             `json:"last_seen,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// NewSession cria uma nova instância de Session
func NewSession(name string) *Session {
	now := time.Now()
	return &Session{
		ID:        uuid.New(),
		Name:      name,
		Status:    WhatsAppStatusDisconnected,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]any),
	}
}

// UpdateStatus atualiza o status da sessão
func (s *Session) UpdateStatus(status WhatsAppSessionStatus) {
	s.Status = status
	s.UpdatedAt = time.Now()
}

// SetJID define o JID da sessão
func (s *Session) SetJID(jid string) {
	s.JID = jid
	s.UpdatedAt = time.Now()
}

// SetQRCode define o QR Code da sessão
func (s *Session) SetQRCode(qrCode string) {
	s.QRCode = qrCode
	s.UpdatedAt = time.Now()
}

// SetProxy define o proxy da sessão
func (s *Session) SetProxy(proxyURL string) {
	s.ProxyURL = proxyURL
	s.UpdatedAt = time.Now()
}

// SetWebhook define o webhook da sessão
func (s *Session) SetWebhook(webhook string) {
	s.Webhook = webhook
	s.UpdatedAt = time.Now()
}

// Activate ativa a sessão
func (s *Session) Activate() {
	s.IsActive = true
	s.UpdatedAt = time.Now()
}

// Deactivate desativa a sessão
func (s *Session) Deactivate() {
	s.IsActive = false
	s.Status = WhatsAppStatusDisconnected
	s.UpdatedAt = time.Now()
}

// UpdateLastSeen atualiza o último acesso da sessão
func (s *Session) UpdateLastSeen() {
	now := time.Now()
	s.LastSeen = &now
	s.UpdatedAt = now
}

// IsConnected verifica se a sessão está conectada
func (s *Session) IsConnected() bool {
	return s.Status == WhatsAppStatusConnected
}

// CanConnect verifica se a sessão pode ser conectada
func (s *Session) CanConnect() bool {
	return s.IsActive && (s.Status == WhatsAppStatusDisconnected || s.Status == WhatsAppStatusError)
}

// SetMetadata define um valor nos metadados
func (s *Session) SetMetadata(key string, value any) {
	if s.Metadata == nil {
		s.Metadata = make(map[string]any)
	}
	s.Metadata[key] = value
	s.UpdatedAt = time.Now()
}

// GetMetadata obtém um valor dos metadados
func (s *Session) GetMetadata(key string) (any, bool) {
	if s.Metadata == nil {
		return nil, false
	}
	value, exists := s.Metadata[key]
	return value, exists
}

