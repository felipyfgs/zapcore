package domain

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

// SessionRepository define a interface para operações de sessão
type SessionRepository interface {
	Create(session *Session) error
	GetByID(id string) (*Session, error)
	GetBySession(sessionName string) (*Session, error)
	GetByToken(token string) (*Session, error)
	Update(session *Session) error
	Delete(id string) error
	DeleteBySession(sessionName string) error
	List() ([]*Session, error)
	GetActive() ([]*Session, error)
	GetConnectedSessions() ([]*Session, error)
}

// SessionService define a interface para serviços de sessão
type SessionService interface {
	CreateSession(req *CreateSessionRequest) (*Session, error)
	GetSession(sessionName string) (*Session, error)
	GetSessionByID(sessionID string) (*Session, error)
	UpdateSessionStatus(id string, status Status) error
	DeleteSession(sessionName string) error
	ListSessions() ([]*Session, error)
	GenerateQRCode(sessionName string) (string, error)
	ConnectSession(sessionName string) error
	DisconnectSession(sessionName string) error
	PairPhone(sessionName string, phone string) error
	GetSessionStatus(sessionName string) (*StatusResponse, error)
	GetConnectedSessionsCount() int
	SendTextMessage(sessionName, to, message string) error
	SendImageMessage(sessionName, to, imageData, caption, mimeType string) error
	SendImageMessageMultiSource(sessionName, to, base64Data, filePath, url, minioID, caption, mimeType string) error
	SendAudioMessage(sessionName, to, audioData string) error
	SendAudioMessageMultiSource(sessionName, to, base64Data, filePath, url, minioID string) error
	SendDocumentMessage(sessionName, to, documentData, filename, mimeType string) error
	SendStickerMessage(sessionName, to, stickerData string) error
}
