package webhook

import (
	"time"

	"github.com/google/uuid"
)

// EventType representa os tipos de eventos de webhook
type EventType string

const (
	EventTypeMessage      EventType = "Message"
	EventTypeReadReceipt  EventType = "ReadReceipt"
	EventTypePresence     EventType = "Presence"
	EventTypeChatPresence EventType = "ChatPresence"
	EventTypeHistorySync  EventType = "HistorySync"
	EventTypeConnected    EventType = "Connected"
	EventTypeDisconnected EventType = "Disconnected"
	EventTypeQRCode       EventType = "QRCode"
	EventTypePairSuccess  EventType = "PairSuccess"
	EventTypeAll          EventType = "All"
)

// DeliveryStatus representa o status de entrega do webhook
type DeliveryStatus string

const (
	DeliveryStatusPending DeliveryStatus = "pending"
	DeliveryStatusSent    DeliveryStatus = "sent"
	DeliveryStatusFailed  DeliveryStatus = "failed"
	DeliveryStatusRetry   DeliveryStatus = "retry"
)

// WebhookEvent representa um evento de webhook
type WebhookEvent struct {
	ID             uuid.UUID      `json:"id"`
	SessionID      uuid.UUID      `json:"sessionId"`
	EventType      EventType      `json:"event_type"`
	Payload        map[string]any `json:"payload"`
	URL            string         `json:"url"`
	Status         DeliveryStatus `json:"status"`
	Attempts       int            `json:"attempts"`
	MaxAttempts    int            `json:"max_attempts"`
	NextRetryAt    *time.Time     `json:"next_retry_at,omitempty"`
	LastError      string         `json:"last_error,omitempty"`
	ResponseStatus int            `json:"response_status,omitempty"`
	ResponseBody   string         `json:"response_body,omitempty"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeliveredAt    *time.Time     `json:"delivered_at,omitempty"`
}

// NewWebhookEvent cria uma nova instância de WebhookEvent
func NewWebhookEvent(sessionID uuid.UUID, eventType EventType, url string, payload map[string]any) *WebhookEvent {
	now := time.Now()
	return &WebhookEvent{
		ID:          uuid.New(),
		SessionID:   sessionID,
		EventType:   eventType,
		Payload:     payload,
		URL:         url,
		Status:      DeliveryStatusPending,
		Attempts:    0,
		MaxAttempts: 3,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// MarkAsSent marca o evento como enviado com sucesso
func (w *WebhookEvent) MarkAsSent(responseStatus int, responseBody string) {
	now := time.Now()
	w.Status = DeliveryStatusSent
	w.ResponseStatus = responseStatus
	w.ResponseBody = responseBody
	w.DeliveredAt = &now
	w.UpdatedAt = now
}

// MarkAsFailed marca o evento como falha
func (w *WebhookEvent) MarkAsFailed(err error, responseStatus int, responseBody string) {
	w.Status = DeliveryStatusFailed
	w.LastError = err.Error()
	w.ResponseStatus = responseStatus
	w.ResponseBody = responseBody
	w.Attempts++
	w.UpdatedAt = time.Now()
}

// ScheduleRetry agenda uma nova tentativa
func (w *WebhookEvent) ScheduleRetry(retryDelay time.Duration) {
	if w.Attempts < w.MaxAttempts {
		w.Status = DeliveryStatusRetry
		nextRetry := time.Now().Add(retryDelay)
		w.NextRetryAt = &nextRetry
		w.UpdatedAt = time.Now()
	} else {
		w.Status = DeliveryStatusFailed
		w.UpdatedAt = time.Now()
	}
}

// CanRetry verifica se o evento pode ser reenviado
func (w *WebhookEvent) CanRetry() bool {
	return w.Attempts < w.MaxAttempts &&
		(w.Status == DeliveryStatusPending || w.Status == DeliveryStatusRetry)
}

// IsReadyForRetry verifica se está pronto para nova tentativa
func (w *WebhookEvent) IsReadyForRetry() bool {
	if !w.CanRetry() {
		return false
	}
	if w.NextRetryAt == nil {
		return true
	}
	return time.Now().After(*w.NextRetryAt)
}

// IncrementAttempt incrementa o contador de tentativas
func (w *WebhookEvent) IncrementAttempt() {
	w.Attempts++
	w.UpdatedAt = time.Now()
}

// SetMaxAttempts define o número máximo de tentativas
func (w *WebhookEvent) SetMaxAttempts(maxAttempts int) {
	w.MaxAttempts = maxAttempts
	w.UpdatedAt = time.Now()
}

// GetRetryDelay calcula o delay para próxima tentativa (exponential backoff)
func (w *WebhookEvent) GetRetryDelay() time.Duration {
	baseDelay := 30 * time.Second
	multiplier := 1 << w.Attempts // 2^attempts
	if multiplier > 8 {
		multiplier = 8 // Máximo de 4 minutos
	}
	return time.Duration(multiplier) * baseDelay
}
