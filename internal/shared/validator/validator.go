package validator

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	entity "wamex/internal/domain/entity"
	"wamex/internal/shared/errors"
)

// ValidationError representa um erro de validação
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors representa múltiplos erros de validação
type ValidationErrors []ValidationError

// Error implementa a interface error
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}

	var messages []string
	for _, err := range ve {
		messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}

	return strings.Join(messages, "; ")
}

// Add adiciona um novo erro de validação
func (ve *ValidationErrors) Add(field, message string) {
	*ve = append(*ve, ValidationError{Field: field, Message: message})
}

// HasErrors verifica se há erros de validação
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// Validator contém métodos de validação específicos da aplicação
type Validator struct{}

// NewValidator cria uma nova instância do validador
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateSessionName valida nome de sessão
func (v *Validator) ValidateSessionName(sessionName string) error {
	var errs ValidationErrors

	if sessionName == "" {
		errs.Add("session_name", "Nome da sessão é obrigatório")
		return &errs
	}

	if len(sessionName) < 3 {
		errs.Add("session_name", "Nome da sessão deve ter pelo menos 3 caracteres")
	}

	if len(sessionName) > 50 {
		errs.Add("session_name", "Nome da sessão deve ter no máximo 50 caracteres")
	}

	// Validar caracteres permitidos (alfanuméricos, hífen e underscore)
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, sessionName)
	if !matched {
		errs.Add("session_name", "Nome da sessão deve conter apenas letras, números, hífen e underscore")
	}

	if errs.HasErrors() {
		return &errs
	}

	return nil
}

// ValidatePhoneNumber valida número de telefone
func (v *Validator) ValidatePhoneNumber(phone string) error {
	var errs ValidationErrors

	if phone == "" {
		errs.Add("phone", "Número de telefone é obrigatório")
		return &errs
	}

	// Remove caracteres não numéricos para validação
	cleaned := regexp.MustCompile(`[^\d]`).ReplaceAllString(phone, "")

	if len(cleaned) < 10 {
		errs.Add("phone", "Número de telefone deve ter pelo menos 10 dígitos")
	}

	if len(cleaned) > 15 {
		errs.Add("phone", "Número de telefone deve ter no máximo 15 dígitos")
	}

	if errs.HasErrors() {
		return &errs
	}

	return nil
}

// ValidateMessage valida mensagem de texto
func (v *Validator) ValidateMessage(message string) error {
	var errs ValidationErrors

	if message == "" {
		errs.Add("message", "Mensagem é obrigatória")
		return &errs
	}

	if utf8.RuneCountInString(message) > 4096 {
		errs.Add("message", "Mensagem deve ter no máximo 4096 caracteres")
	}

	if errs.HasErrors() {
		return &errs
	}

	return nil
}

// ValidateMediaType valida tipo de mídia
func (v *Validator) ValidateMediaType(mimeType string, messageType entity.MessageType) error {
	var errs ValidationErrors

	if mimeType == "" {
		errs.Add("mime_type", "Tipo MIME é obrigatório")
		return &errs
	}

	allowedTypes := map[entity.MessageType][]string{
		entity.MessageTypeImage: {
			"image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp",
		},
		entity.MessageTypeAudio: {
			"audio/mpeg", "audio/mp3", "audio/wav", "audio/ogg", "audio/aac",
		},
		entity.MessageTypeVideo: {
			"video/mp4", "video/avi", "video/mov", "video/wmv", "video/webm",
		},
		entity.MessageTypeDocument: {
			"application/pdf", "application/msword", "application/vnd.ms-excel",
			"application/vnd.ms-powerpoint", "text/plain", "text/csv",
		},
	}

	allowed, exists := allowedTypes[messageType]
	if !exists {
		errs.Add("message_type", "Tipo de mensagem não suportado")
		return &errs
	}

	isAllowed := false
	for _, allowedType := range allowed {
		if strings.EqualFold(mimeType, allowedType) {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		errs.Add("mime_type", fmt.Sprintf("Tipo MIME '%s' não permitido para %s", mimeType, messageType))
	}

	if errs.HasErrors() {
		return &errs
	}

	return nil
}

// ValidateURL valida URL
func (v *Validator) ValidateURL(urlStr string) error {
	var errs ValidationErrors

	if urlStr == "" {
		errs.Add("url", "URL é obrigatória")
		return &errs
	}

	_, err := url.Parse(urlStr)
	if err != nil {
		errs.Add("url", "URL inválida")
	}

	if errs.HasErrors() {
		return &errs
	}

	return nil
}

// ValidateFileSize valida tamanho de arquivo
func (v *Validator) ValidateFileSize(size int64, messageType entity.MessageType) error {
	var errs ValidationErrors

	maxSizes := map[entity.MessageType]int64{
		entity.MessageTypeImage:    10 * 1024 * 1024,  // 10MB
		entity.MessageTypeAudio:    16 * 1024 * 1024,  // 16MB
		entity.MessageTypeVideo:    64 * 1024 * 1024,  // 64MB
		entity.MessageTypeDocument: 100 * 1024 * 1024, // 100MB
	}

	maxSize, exists := maxSizes[messageType]
	if !exists {
		errs.Add("message_type", "Tipo de mensagem não suportado")
		return &errs
	}

	if size > maxSize {
		errs.Add("file_size", fmt.Sprintf("Arquivo muito grande. Máximo permitido: %d bytes", maxSize))
	}

	if errs.HasErrors() {
		return &errs
	}

	return nil
}

// ToAppError converte ValidationErrors para AppError
func (ve ValidationErrors) ToAppError() *errors.AppError {
	if !ve.HasErrors() {
		return nil
	}

	return errors.ErrInvalidInput.WithDetails(ve.Error())
}
