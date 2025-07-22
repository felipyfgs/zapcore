package errors

import (
	"fmt"
	"net/http"
)

// MediaErrorCode representa códigos de erro específicos para mídia
type MediaErrorCode string

const (
	// Erros de validação
	ErrCodeInvalidMediaType  MediaErrorCode = "INVALID_MEDIA_TYPE"
	ErrCodeUnsupportedFormat MediaErrorCode = "UNSUPPORTED_FORMAT"
	ErrCodeFileTooLarge      MediaErrorCode = "FILE_TOO_LARGE"
	ErrCodeFileEmpty         MediaErrorCode = "FILE_EMPTY"
	ErrCodeInvalidMimeType   MediaErrorCode = "INVALID_MIME_TYPE"

	// Erros de URL
	ErrCodeInvalidURL       MediaErrorCode = "INVALID_URL"
	ErrCodeURLNotAccessible MediaErrorCode = "URL_NOT_ACCESSIBLE"
	ErrCodeDownloadFailed   MediaErrorCode = "DOWNLOAD_FAILED"

	// Erros de base64
	ErrCodeInvalidBase64      MediaErrorCode = "INVALID_BASE64"
	ErrCodeBase64DecodeFailed MediaErrorCode = "BASE64_DECODE_FAILED"

	// Erros de upload
	ErrCodeUploadFailed     MediaErrorCode = "UPLOAD_FAILED"
	ErrCodeProcessingFailed MediaErrorCode = "PROCESSING_FAILED"

	// Erros de sessão
	ErrCodeSessionNotFound     MediaErrorCode = "SESSION_NOT_FOUND"
	ErrCodeSessionNotActive    MediaErrorCode = "SESSION_NOT_ACTIVE"
	ErrCodeSessionNotConnected MediaErrorCode = "SESSION_NOT_CONNECTED"

	// Erros de envio
	ErrCodeSendFailed    MediaErrorCode = "SEND_FAILED"
	ErrCodeWhatsAppError MediaErrorCode = "WHATSAPP_ERROR"

	// Erros internos
	ErrCodeInternalError MediaErrorCode = "INTERNAL_ERROR"
	ErrCodeDatabaseError MediaErrorCode = "DATABASE_ERROR"
)

// MediaError representa um erro específico de mídia
type MediaError struct {
	Code       MediaErrorCode `json:"code"`
	Message    string         `json:"message"`
	Details    string         `json:"details,omitempty"`
	HTTPStatus int            `json:"-"`
	Cause      error          `json:"-"`
}

// Error implementa a interface error
func (e *MediaError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap retorna o erro causa
func (e *MediaError) Unwrap() error {
	return e.Cause
}

// GetHTTPStatus retorna o status HTTP apropriado
func (e *MediaError) GetHTTPStatus() int {
	if e.HTTPStatus != 0 {
		return e.HTTPStatus
	}

	// Status padrão baseado no código
	switch e.Code {
	case ErrCodeInvalidMediaType, ErrCodeUnsupportedFormat, ErrCodeFileTooLarge,
		ErrCodeFileEmpty, ErrCodeInvalidMimeType, ErrCodeInvalidURL,
		ErrCodeInvalidBase64, ErrCodeBase64DecodeFailed:
		return http.StatusBadRequest
	case ErrCodeSessionNotFound:
		return http.StatusNotFound
	case ErrCodeSessionNotActive, ErrCodeSessionNotConnected:
		return http.StatusConflict
	case ErrCodeURLNotAccessible, ErrCodeDownloadFailed:
		return http.StatusBadGateway
	case ErrCodeUploadFailed, ErrCodeSendFailed, ErrCodeWhatsAppError:
		return http.StatusServiceUnavailable
	case ErrCodeInternalError, ErrCodeDatabaseError, ErrCodeProcessingFailed:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// NewMediaError cria um novo erro de mídia
func NewMediaError(code MediaErrorCode, message string) *MediaError {
	return &MediaError{
		Code:    code,
		Message: message,
	}
}

// NewMediaErrorWithDetails cria um novo erro de mídia com detalhes
func NewMediaErrorWithDetails(code MediaErrorCode, message, details string) *MediaError {
	return &MediaError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// NewMediaErrorWithCause cria um novo erro de mídia com causa
func NewMediaErrorWithCause(code MediaErrorCode, message string, cause error) *MediaError {
	return &MediaError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Funções de conveniência para criar erros específicos

// ErrInvalidMediaType cria erro de tipo de mídia inválido
func ErrInvalidMediaType(mediaType string) *MediaError {
	return NewMediaErrorWithDetails(
		ErrCodeInvalidMediaType,
		"Tipo de mídia não suportado",
		fmt.Sprintf("Tipo '%s' não é suportado", mediaType),
	)
}

// ErrFileTooLarge cria erro de arquivo muito grande
func ErrFileTooLarge(size, maxSize int64) *MediaError {
	return NewMediaErrorWithDetails(
		ErrCodeFileTooLarge,
		"Arquivo muito grande",
		fmt.Sprintf("Tamanho: %d bytes, máximo permitido: %d bytes", size, maxSize),
	)
}

// ErrUnsupportedFormat cria erro de formato não suportado
func ErrUnsupportedFormat(format, mediaType string) *MediaError {
	return NewMediaErrorWithDetails(
		ErrCodeUnsupportedFormat,
		"Formato de arquivo não suportado",
		fmt.Sprintf("Formato '%s' não é suportado para %s", format, mediaType),
	)
}

// ErrInvalidURL cria erro de URL inválida
func ErrInvalidURL(url string) *MediaError {
	return NewMediaErrorWithDetails(
		ErrCodeInvalidURL,
		"URL inválida",
		fmt.Sprintf("URL '%s' não é válida", url),
	)
}

// ErrDownloadFailed cria erro de falha no download
func ErrDownloadFailed(url string, cause error) *MediaError {
	return &MediaError{
		Code:    ErrCodeDownloadFailed,
		Message: "Falha ao baixar mídia da URL",
		Details: fmt.Sprintf("URL: %s", url),
		Cause:   cause,
	}
}

// ErrBase64DecodeFailed cria erro de falha na decodificação base64
func ErrBase64DecodeFailed(cause error) *MediaError {
	return NewMediaErrorWithCause(
		ErrCodeBase64DecodeFailed,
		"Falha ao decodificar dados base64",
		cause,
	)
}

// ErrSessionNotFound cria erro de sessão não encontrada
func ErrSessionNotFound(sessionID string) *MediaError {
	return NewMediaErrorWithDetails(
		ErrCodeSessionNotFound,
		"Sessão não encontrada",
		fmt.Sprintf("Sessão '%s' não existe", sessionID),
	)
}

// ErrSessionNotActive cria erro de sessão não ativa
func ErrSessionNotActive(sessionID string) *MediaError {
	return NewMediaErrorWithDetails(
		ErrCodeSessionNotActive,
		"Sessão não está ativa",
		fmt.Sprintf("Sessão '%s' precisa estar ativa para enviar mensagens", sessionID),
	)
}

// ErrSessionNotConnected cria erro de sessão não conectada
func ErrSessionNotConnected(sessionID string) *MediaError {
	return NewMediaErrorWithDetails(
		ErrCodeSessionNotConnected,
		"Sessão não está conectada ao WhatsApp",
		fmt.Sprintf("Sessão '%s' precisa estar conectada", sessionID),
	)
}

// ErrSendFailed cria erro de falha no envio
func ErrSendFailed(mediaType string, cause error) *MediaError {
	return &MediaError{
		Code:    ErrCodeSendFailed,
		Message: fmt.Sprintf("Falha ao enviar %s", mediaType),
		Cause:   cause,
	}
}

// ErrProcessingFailed cria erro de falha no processamento
func ErrProcessingFailed(operation string, cause error) *MediaError {
	return &MediaError{
		Code:    ErrCodeProcessingFailed,
		Message: fmt.Sprintf("Falha ao processar mídia: %s", operation),
		Cause:   cause,
	}
}

// ErrInternalError cria erro interno
func ErrInternalError(message string, cause error) *MediaError {
	return &MediaError{
		Code:    ErrCodeInternalError,
		Message: message,
		Cause:   cause,
	}
}

// IsMediaError verifica se um erro é do tipo MediaError
func IsMediaError(err error) bool {
	_, ok := err.(*MediaError)
	return ok
}

// GetMediaError extrai MediaError de um erro
func GetMediaError(err error) *MediaError {
	if mediaErr, ok := err.(*MediaError); ok {
		return mediaErr
	}
	return nil
}

// WrapError converte um erro genérico em MediaError
func WrapError(err error, code MediaErrorCode, message string) *MediaError {
	return &MediaError{
		Code:    code,
		Message: message,
		Cause:   err,
	}
}
