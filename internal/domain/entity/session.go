package entity

import (
	"time"
)

// Proxy representa configuração de proxy
type Proxy struct {
	URL string `json:"url" bun:"url"`
}

// Status representa o status de uma sessão WhatsApp
type Status string

const (
	StatusDisconnected Status = "disconnected"
	StatusConnecting   Status = "connecting"
	StatusConnected    Status = "connected"
)

// Session representa uma sessão WhatsApp
type Session struct {
	ID          string    `json:"id" bun:"id,pk"`
	Session     string    `json:"session" bun:"session"`
	Status      Status    `json:"status" bun:"status"`
	DeviceJID   string    `json:"device_jid,omitempty" bun:"device_jid"`
	QRCode      string    `json:"qr_code,omitempty" bun:"qr_code"`
	ProxyConfig *Proxy    `json:"proxy,omitempty" bun:"proxy_config"`
	CreatedAt   time.Time `json:"created_at" bun:"created_at,default:current_timestamp"`
	UpdatedAt   time.Time `json:"updated_at" bun:"updated_at,default:current_timestamp"`
}

// CreateSessionRequest representa a requisição para criar uma sessão
type CreateSessionRequest struct {
	Session    string `json:"session" validate:"required,min=3,max=50"`
	WebhookURL string `json:"webhook_url" validate:"required,url"`
}

// PairPhoneRequest representa a requisição para emparelhar telefone
type PairPhoneRequest struct {
	Phone string `json:"phone" validate:"required,min=10,max=15"`
}

// SessionResponse representa a resposta padrão para operações de sessão
type SessionResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// QRCodeResponse representa a resposta do QR Code
type QRCodeResponse struct {
	Success   bool   `json:"success"`
	SessionID string `json:"session_id"`
	QRCode    string `json:"qr_code"`
	Message   string `json:"message"`
}

// StatusResponse representa a resposta do status da sessão
type StatusResponse struct {
	Success   bool   `json:"success"`
	SessionID string `json:"session_id"`
	Status    Status `json:"status"`
	Phone     string `json:"phone,omitempty"`
	Message   string `json:"message"`
}

// Interfaces movidas para interfaces.go
