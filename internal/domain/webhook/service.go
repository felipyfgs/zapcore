package webhook

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Service define a interface para o serviço de webhook
type Service interface {
	// Send envia um webhook
	Send(ctx context.Context, event *WebhookEvent) error

	// SendAsync envia um webhook de forma assíncrona
	SendAsync(ctx context.Context, event *WebhookEvent) error

	// Retry reprocessa webhooks com falha
	Retry(ctx context.Context, eventID uuid.UUID) error

	// ProcessPendingEvents processa eventos pendentes
	ProcessPendingEvents(ctx context.Context) error

	// GetDeliveryStats retorna estatísticas de entrega
	GetDeliveryStats(ctx context.Context, sessionID uuid.UUID, period time.Duration) (*DeliveryStats, error)
}

// Repository define a interface para persistência de webhooks
type Repository interface {
	// Create cria um novo evento de webhook
	Create(ctx context.Context, event *WebhookEvent) error

	// GetByID busca um evento pelo ID
	GetByID(ctx context.Context, id uuid.UUID) (*WebhookEvent, error)

	// List retorna eventos com filtros opcionais
	List(ctx context.Context, filters ListFilters) ([]*WebhookEvent, error)

	// Update atualiza um evento existente
	Update(ctx context.Context, event *WebhookEvent) error

	// Delete remove um evento
	Delete(ctx context.Context, id uuid.UUID) error

	// GetPendingEvents retorna eventos pendentes para processamento
	GetPendingEvents(ctx context.Context, limit int) ([]*WebhookEvent, error)

	// GetRetryableEvents retorna eventos que podem ser reenviados
	GetRetryableEvents(ctx context.Context, limit int) ([]*WebhookEvent, error)

	// GetBySessionID retorna eventos de uma sessão específica
	GetBySessionID(ctx context.Context, sessionID uuid.UUID, filters ListFilters) ([]*WebhookEvent, error)

	// GetDeliveryStats retorna estatísticas de entrega
	GetDeliveryStats(ctx context.Context, sessionID uuid.UUID, period time.Duration) (*DeliveryStats, error)

	// CleanupOldEvents remove eventos antigos
	CleanupOldEvents(ctx context.Context, olderThan time.Duration) error
}

// ListFilters define os filtros para listagem de eventos
type ListFilters struct {
	SessionID *uuid.UUID      `json:"session_id,omitempty"`
	EventType *EventType      `json:"event_type,omitempty"`
	Status    *DeliveryStatus `json:"status,omitempty"`
	DateFrom  *time.Time      `json:"date_from,omitempty"`
	DateTo    *time.Time      `json:"date_to,omitempty"`
	Limit     int             `json:"limit,omitempty"`
	Offset    int             `json:"offset,omitempty"`
	OrderBy   string          `json:"order_by,omitempty"`
	OrderDir  string          `json:"order_dir,omitempty"`
}

// DefaultListFilters retorna os filtros padrão para listagem
func DefaultListFilters() ListFilters {
	return ListFilters{
		Limit:    50,
		Offset:   0,
		OrderBy:  "createdAt",
		OrderDir: "DESC",
	}
}

// DeliveryStats representa estatísticas de entrega de webhooks
type DeliveryStats struct {
	TotalEvents    int     `json:"total_events"`
	SentEvents     int     `json:"sent_events"`
	FailedEvents   int     `json:"failed_events"`
	PendingEvents  int     `json:"pending_events"`
	SuccessRate    float64 `json:"success_rate"`
	AverageLatency int64   `json:"average_latency_ms"`
}

// EventSubscription representa uma inscrição de eventos
type EventSubscription struct {
	SessionID uuid.UUID   `json:"sessionId"`
	URL       string      `json:"url"`
	Events    []EventType `json:"events"`
	IsActive  bool        `json:"isActive"`
}

// NewEventSubscription cria uma nova inscrição de eventos
func NewEventSubscription(sessionID uuid.UUID, url string, events []EventType) *EventSubscription {
	return &EventSubscription{
		SessionID: sessionID,
		URL:       url,
		Events:    events,
		IsActive:  true,
	}
}

// IsSubscribedTo verifica se está inscrito em um tipo de evento
func (s *EventSubscription) IsSubscribedTo(eventType EventType) bool {
	for _, event := range s.Events {
		if event == eventType || event == EventTypeAll {
			return true
		}
	}
	return false
}
