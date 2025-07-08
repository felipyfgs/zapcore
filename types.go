package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
)

// SessionStatus representa os possíveis status de uma sessão
type SessionStatus string

const (
	StatusDisconnected SessionStatus = "disconnected"
	StatusConnecting   SessionStatus = "connecting"
	StatusConnected    SessionStatus = "connected"
	StatusLoggedOut    SessionStatus = "logged_out"
	StatusError        SessionStatus = "error"
)

// WhatsAppSession representa uma sessão do WhatsApp
type WhatsAppSession struct {
	ID          string                         `json:"id"`
	Name        string                         `json:"name,omitempty"`
	Status      SessionStatus                  `json:"status"`
	DeviceJID   *string                        `json:"device_jid,omitempty"`
	CreatedAt   time.Time                      `json:"created_at"`
	ConnectedAt *time.Time                     `json:"connected_at,omitempty"`
	LastSeen    *time.Time                     `json:"last_seen,omitempty"`
	QRCode      string                         `json:"qr_code,omitempty"`
	Client      *whatsmeow.Client              `json:"-"`
	Device      *store.Device                  `json:"-"`
	EventChan   chan interface{}               `json:"-"`
	QRChan      <-chan whatsmeow.QRChannelItem `json:"-"`
	CancelFunc  context.CancelFunc             `json:"-"`
}

// CreateSessionRequest representa a requisição para criar uma nova sessão
type CreateSessionRequest struct {
	Name string `json:"name,omitempty"`
}

// CreateSessionResponse representa a resposta da criação de sessão
type CreateSessionResponse struct {
	Session *WhatsAppSession `json:"session"`
	Message string           `json:"message"`
}

// SessionListResponse representa a resposta da listagem de sessões
type SessionListResponse struct {
	Sessions []*WhatsAppSession `json:"sessions"`
	Count    int                `json:"count"`
}

// SessionResponse representa a resposta de uma sessão específica
type SessionResponse struct {
	Session *WhatsAppSession `json:"session"`
}

// StatusResponse representa a resposta do status de uma sessão
type StatusResponse struct {
	SessionID string        `json:"session_id"`
	Status    SessionStatus `json:"status"`
	Message   string        `json:"message,omitempty"`
}

// QRResponse representa a resposta do QR Code
type QRResponse struct {
	SessionID string `json:"session_id"`
	QRCode    string `json:"qr_code"`
	Message   string `json:"message,omitempty"`
}

// ErrorResponse representa uma resposta de erro
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// ConnectResponse representa a resposta de conexão
type ConnectResponse struct {
	SessionID string        `json:"session_id"`
	Status    SessionStatus `json:"status"`
	Message   string        `json:"message"`
}

// DisconnectResponse representa a resposta de desconexão
type DisconnectResponse struct {
	SessionID string        `json:"session_id"`
	Status    SessionStatus `json:"status"`
	Message   string        `json:"message"`
}

// SessionEvent representa um evento de sessão
type SessionEvent struct {
	SessionID string      `json:"session_id"`
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// EventHandler é uma interface para manipular eventos
type EventHandler interface {
	HandleEvent(evt interface{})
}

// WhatsAppEventHandler implementa o EventHandler para eventos do WhatsApp
type WhatsAppEventHandler struct {
	SessionID string
	EventChan chan interface{}
}

func (h *WhatsAppEventHandler) HandleEvent(evt interface{}) {
	select {
	case h.EventChan <- evt:
	default:
		// Canal cheio, descartar evento
	}
}

// Métodos auxiliares para WhatsAppSession

// IsConnected verifica se a sessão está conectada
func (s *WhatsAppSession) IsConnected() bool {
	return s.Status == StatusConnected && s.Client != nil && s.Client.IsConnected()
}

// IsLoggedIn verifica se a sessão está logada
func (s *WhatsAppSession) IsLoggedIn() bool {
	return s.Client != nil && s.Client.IsLoggedIn()
}

// UpdateStatus atualiza o status da sessão
func (s *WhatsAppSession) UpdateStatus(status SessionStatus) {
	s.Status = status
	now := time.Now()
	s.LastSeen = &now

	if status == StatusConnected {
		s.ConnectedAt = &now
	}
}

// GetInfo retorna informações básicas da sessão (sem dados sensíveis)
func (s *WhatsAppSession) GetInfo() *WhatsAppSession {
	return &WhatsAppSession{
		ID:          s.ID,
		Name:        s.Name,
		Status:      s.Status,
		CreatedAt:   s.CreatedAt,
		ConnectedAt: s.ConnectedAt,
		LastSeen:    s.LastSeen,
	}
}

// ValidateSessionName valida se o nome da sessão é seguro para URLs
func ValidateSessionName(name string) error {
	if name == "" {
		return fmt.Errorf("nome da sessão não pode estar vazio")
	}

	// Verificar comprimento
	if len(name) < 3 {
		return fmt.Errorf("nome da sessão deve ter pelo menos 3 caracteres")
	}

	if len(name) > 50 {
		return fmt.Errorf("nome da sessão deve ter no máximo 50 caracteres")
	}

	// Verificar se contém apenas caracteres permitidos: a-z, A-Z, 0-9, _, -
	validNameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validNameRegex.MatchString(name) {
		return fmt.Errorf("nome da sessão deve conter apenas letras (a-z, A-Z), números (0-9), underscore (_) e hífen (-)")
	}

	// Verificar se não começa ou termina com hífen ou underscore
	if strings.HasPrefix(name, "-") || strings.HasPrefix(name, "_") ||
		strings.HasSuffix(name, "-") || strings.HasSuffix(name, "_") {
		return fmt.Errorf("nome da sessão não pode começar ou terminar com hífen (-) ou underscore (_)")
	}

	// Verificar se não contém sequências consecutivas de hífens ou underscores
	if strings.Contains(name, "--") || strings.Contains(name, "__") || strings.Contains(name, "_-") || strings.Contains(name, "-_") {
		return fmt.Errorf("nome da sessão não pode conter sequências consecutivas de hífen ou underscore")
	}

	return nil
}

// SanitizeSessionName sanitiza um nome para torná-lo seguro para URLs
func SanitizeSessionName(name string) string {
	if name == "" {
		return ""
	}

	// Converter para minúsculas
	name = strings.ToLower(name)

	// Remover acentos e caracteres especiais
	name = removeAccents(name)

	// Substituir espaços e caracteres não permitidos por hífen
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
	name = reg.ReplaceAllString(name, "-")

	// Remover hífens consecutivos
	reg = regexp.MustCompile(`-+`)
	name = reg.ReplaceAllString(name, "-")

	// Remover hífens do início e fim
	name = strings.Trim(name, "-_")

	// Limitar comprimento
	if len(name) > 50 {
		name = name[:50]
		name = strings.TrimRight(name, "-_")
	}

	return name
}

// removeAccents remove acentos de caracteres
func removeAccents(s string) string {
	// Mapa de caracteres acentuados para não acentuados
	accentMap := map[rune]string{
		'á': "a", 'à': "a", 'ã': "a", 'â': "a", 'ä': "a",
		'é': "e", 'è': "e", 'ê': "e", 'ë': "e",
		'í': "i", 'ì': "i", 'î': "i", 'ï': "i",
		'ó': "o", 'ò': "o", 'õ': "o", 'ô': "o", 'ö': "o",
		'ú': "u", 'ù': "u", 'û': "u", 'ü': "u",
		'ç': "c", 'ñ': "n",
		'Á': "A", 'À': "A", 'Ã': "A", 'Â': "A", 'Ä': "A",
		'É': "E", 'È': "E", 'Ê': "E", 'Ë': "E",
		'Í': "I", 'Ì': "I", 'Î': "I", 'Ï': "I",
		'Ó': "O", 'Ò': "O", 'Õ': "O", 'Ô': "O", 'Ö': "O",
		'Ú': "U", 'Ù': "U", 'Û': "U", 'Ü': "U",
		'Ç': "C", 'Ñ': "N",
	}

	var result strings.Builder
	for _, r := range s {
		if replacement, exists := accentMap[r]; exists {
			result.WriteString(replacement)
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == ' ' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// Requisição para enviar mensagem
type SendMessageRequest struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

// Resposta para envio de mensagem
type SendMessageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	To      string `json:"to"`
	Content string `json:"content"`
}
