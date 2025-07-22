package session

import (
	"context"

	"github.com/google/uuid"
)

// Service define a interface para o serviço de sessões
type Service interface {
	// Create cria uma nova sessão
	Create(ctx context.Context, name string) (*Session, error)

	// Connect estabelece conexão com o WhatsApp
	Connect(ctx context.Context, sessionID uuid.UUID) error

	// Disconnect encerra a conexão
	Disconnect(ctx context.Context, sessionID uuid.UUID) error

	// GetQRCode gera QR Code para autenticação
	GetQRCode(ctx context.Context, sessionID uuid.UUID) (string, error)

	// PairPhone emparelha com um número de telefone
	PairPhone(ctx context.Context, sessionID uuid.UUID, phoneNumber string) error

	// GetStatus retorna o status da sessão
	GetStatus(ctx context.Context, sessionID uuid.UUID) (WhatsAppSessionStatus, error)

	// SetProxy configura proxy para a sessão
	SetProxy(ctx context.Context, sessionID uuid.UUID, proxyURL string) error

	// SetWebhook configura webhook para a sessão
	SetWebhook(ctx context.Context, sessionID uuid.UUID, webhookURL string) error

	// Delete remove uma sessão permanentemente
	Delete(ctx context.Context, sessionID uuid.UUID) error

	// List lista todas as sessões com filtros
	List(ctx context.Context, filters ListFilters) ([]*Session, error)

	// GetByName busca sessão pelo nome
	GetByName(ctx context.Context, name string) (*Session, error)

	// GetActiveCount retorna número de sessões ativas
	GetActiveCount(ctx context.Context) (int, error)

	// ReconnectAll reconecta todas as sessões que estavam conectadas
	ReconnectAll(ctx context.Context) error

	// HealthCheck verifica saúde de todas as sessões
	HealthCheck(ctx context.Context) (*HealthStatus, error)
}

// HealthStatus representa o status de saúde das sessões
type HealthStatus struct {
	TotalSessions     int                         `json:"total_sessions"`
	ActiveSessions    int                         `json:"active_sessions"`
	ConnectedSessions int                         `json:"connected_sessions"`
	ErrorSessions     int                         `json:"error_sessions"`
	SessionDetails    map[uuid.UUID]SessionHealth `json:"session_details"`
}

// SessionHealth representa a saúde de uma sessão específica
type SessionHealth struct {
	ID           uuid.UUID             `json:"id"`
	Name         string                `json:"name"`
	Status       WhatsAppSessionStatus `json:"status"`
	IsHealthy    bool                  `json:"is_healthy"`
	LastSeen     string                `json:"last_seen,omitempty"`
	ErrorMessage string                `json:"error_message,omitempty"`
}
