package errors

import (
	"fmt"
	"net/http"
)

// AppError representa um erro da aplicação com contexto adicional
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Status  int    `json:"-"`
}

// Error implementa a interface error
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewAppError cria um novo erro da aplicação
func NewAppError(code, message string, status int) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Status:  status,
	}
}

// WithDetails adiciona detalhes ao erro
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// Erros comuns da aplicação
var (
	ErrInvalidInput = NewAppError("INVALID_INPUT", "Entrada inválida", http.StatusBadRequest)
	ErrNotFound     = NewAppError("NOT_FOUND", "Recurso não encontrado", http.StatusNotFound)
	ErrUnauthorized = NewAppError("UNAUTHORIZED", "Não autorizado", http.StatusUnauthorized)
	ErrForbidden    = NewAppError("FORBIDDEN", "Acesso negado", http.StatusForbidden)
	ErrInternal     = NewAppError("INTERNAL_ERROR", "Erro interno do servidor", http.StatusInternalServerError)
	ErrConflict     = NewAppError("CONFLICT", "Conflito de dados", http.StatusConflict)
)

// Erros específicos do domínio
var (
	ErrSessionNotFound    = NewAppError("SESSION_NOT_FOUND", "Sessão não encontrada", http.StatusNotFound)
	ErrSessionExists      = NewAppError("SESSION_EXISTS", "Sessão já existe", http.StatusConflict)
	ErrInvalidMediaType   = NewAppError("INVALID_MEDIA_TYPE", "Tipo de mídia inválido", http.StatusBadRequest)
	ErrMediaTooLarge      = NewAppError("MEDIA_TOO_LARGE", "Arquivo de mídia muito grande", http.StatusRequestEntityTooLarge)
	ErrWhatsAppNotConnected = NewAppError("WHATSAPP_NOT_CONNECTED", "WhatsApp não conectado", http.StatusServiceUnavailable)
)
