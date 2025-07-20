package session

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Erros específicos do domínio de sessão
var (
	ErrSessionNotFound      = errors.New("sessão não encontrada")
	ErrSessionAlreadyExists = errors.New("sessão já existe")
	ErrSessionNotActive     = errors.New("sessão não está ativa")
	ErrSessionNotConnected  = errors.New("sessão não está conectada")
	ErrSessionAlreadyConnected = errors.New("sessão já está conectada")
	ErrSessionConnecting    = errors.New("sessão está em processo de conexão")
	ErrInvalidSessionName   = errors.New("nome da sessão inválido")
	ErrInvalidPhoneNumber   = errors.New("número de telefone inválido")
	ErrInvalidProxyURL      = errors.New("URL do proxy inválida")
	ErrInvalidWebhookURL    = errors.New("URL do webhook inválida")
	ErrSessionTimeout       = errors.New("timeout na operação da sessão")
	ErrQRCodeExpired        = errors.New("QR Code expirado")
	ErrPairingFailed        = errors.New("falha no emparelhamento")
)

// SessionError representa um erro específico de sessão com contexto
type SessionError struct {
	SessionID uuid.UUID
	Operation string
	Err       error
}

func (e *SessionError) Error() string {
	return fmt.Sprintf("erro na sessão %s durante %s: %v", e.SessionID, e.Operation, e.Err)
}

func (e *SessionError) Unwrap() error {
	return e.Err
}

// NewSessionError cria um novo erro de sessão
func NewSessionError(sessionID uuid.UUID, operation string, err error) *SessionError {
	return &SessionError{
		SessionID: sessionID,
		Operation: operation,
		Err:       err,
	}
}

// ValidationError representa um erro de validação
type ValidationError struct {
	Field   string
	Value   any
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validação falhou para o campo '%s' com valor '%v': %s", e.Field, e.Value, e.Message)
}

// NewValidationError cria um novo erro de validação
func NewValidationError(field string, value any, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// ConnectionError representa um erro de conexão
type ConnectionError struct {
	SessionID uuid.UUID
	Reason    string
	Err       error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("erro de conexão na sessão %s: %s - %v", e.SessionID, e.Reason, e.Err)
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// NewConnectionError cria um novo erro de conexão
func NewConnectionError(sessionID uuid.UUID, reason string, err error) *ConnectionError {
	return &ConnectionError{
		SessionID: sessionID,
		Reason:    reason,
		Err:       err,
	}
}

