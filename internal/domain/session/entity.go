package session

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// WhatsAppSessionStatus representa os possíveis status de uma sessão WhatsApp
type WhatsAppSessionStatus string

const (
	WhatsAppStatusDisconnected WhatsAppSessionStatus = "disconnected"
	WhatsAppStatusConnecting   WhatsAppSessionStatus = "connecting"
	WhatsAppStatusConnected    WhatsAppSessionStatus = "connected"
)

// Session representa uma sessão do WhatsApp
type Session struct {
	bun.BaseModel `bun:"table:zapcore_sessions,alias:s"`

	ID        uuid.UUID             `bun:"id,pk,type:uuid" json:"id"`
	Name      string                `bun:"name,type:varchar(100),notnull,unique" json:"name"`
	Status    WhatsAppSessionStatus `bun:"status,type:varchar(20),notnull" json:"status"`
	JID       string                `bun:"jid,type:varchar(100)" json:"jid,omitempty"`
	QRCode    string                `bun:"-" json:"qrCode,omitempty"`   // Não persistir no banco
	ProxyURL  string                `bun:"-" json:"proxyUrl,omitempty"` // Não persistir no banco
	Webhook   string                `bun:"-" json:"webhook,omitempty"`  // Não persistir no banco
	IsActive  bool                  `bun:"isActive,type:boolean" json:"isActive"`
	LastSeen  *time.Time            `bun:"lastSeen,type:timestamptz" json:"lastSeen,omitempty"`
	CreatedAt time.Time             `bun:"createdAt,type:timestamptz,notnull" json:"createdAt"`
	UpdatedAt time.Time             `bun:"updatedAt,type:timestamptz,notnull" json:"updatedAt"`
	Metadata  map[string]any        `bun:"-" json:"metadata,omitempty"` // Não persistir no banco por enquanto
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
	return s.IsActive && s.Status == WhatsAppStatusDisconnected
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
