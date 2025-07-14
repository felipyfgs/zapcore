package validator

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// ValidationError representa um erro de validação
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// ValidationErrors representa múltiplos erros de validação
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	if len(e) == 1 {
		return e[0].Error()
	}
	
	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// HasErrors verifica se há erros
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// Validator representa um validador
type Validator struct {
	errors ValidationErrors
}

// New cria um novo validador
func New() *Validator {
	return &Validator{
		errors: make(ValidationErrors, 0),
	}
}

// AddError adiciona um erro de validação
func (v *Validator) AddError(field, message string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

// HasErrors verifica se há erros
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// Errors retorna os erros de validação
func (v *Validator) Errors() ValidationErrors {
	return v.errors
}

// Required valida se um campo é obrigatório
func (v *Validator) Required(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.AddError(field, "is required")
	}
}

// MinLength valida comprimento mínimo
func (v *Validator) MinLength(field, value string, min int) {
	if len(value) < min {
		v.AddError(field, fmt.Sprintf("must be at least %d characters long", min))
	}
}

// MaxLength valida comprimento máximo
func (v *Validator) MaxLength(field, value string, max int) {
	if len(value) > max {
		v.AddError(field, fmt.Sprintf("must be at most %d characters long", max))
	}
}

// Email valida formato de email
func (v *Validator) Email(field, value string) {
	if value != "" {
		if _, err := mail.ParseAddress(value); err != nil {
			v.AddError(field, "must be a valid email address")
		}
	}
}

// URL valida formato de URL
func (v *Validator) URL(field, value string) {
	if value != "" {
		if _, err := url.ParseRequestURI(value); err != nil {
			v.AddError(field, "must be a valid URL")
		}
	}
}

// Phone valida formato de telefone (formato brasileiro)
func (v *Validator) Phone(field, value string) {
	if value != "" {
		// Regex para telefone brasileiro: (XX) XXXXX-XXXX ou (XX) XXXX-XXXX
		phoneRegex := regexp.MustCompile(`^\(\d{2}\)\s\d{4,5}-\d{4}$`)
		if !phoneRegex.MatchString(value) {
			v.AddError(field, "must be a valid phone number format: (XX) XXXXX-XXXX")
		}
	}
}

// Regex valida usando expressão regular
func (v *Validator) Regex(field, value, pattern, message string) {
	if value != "" {
		regex := regexp.MustCompile(pattern)
		if !regex.MatchString(value) {
			v.AddError(field, message)
		}
	}
}

// In valida se valor está em lista de opções válidas
func (v *Validator) In(field, value string, options []string) {
	if value != "" {
		for _, option := range options {
			if value == option {
				return
			}
		}
		v.AddError(field, fmt.Sprintf("must be one of: %s", strings.Join(options, ", ")))
	}
}

// StrongPassword valida se senha é forte
func (v *Validator) StrongPassword(field, value string) {
	if value == "" {
		return
	}

	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)

	for _, char := range value {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if len(value) < 8 {
		v.AddError(field, "must be at least 8 characters long")
	}
	if !hasUpper {
		v.AddError(field, "must contain at least one uppercase letter")
	}
	if !hasLower {
		v.AddError(field, "must contain at least one lowercase letter")
	}
	if !hasNumber {
		v.AddError(field, "must contain at least one number")
	}
	if !hasSpecial {
		v.AddError(field, "must contain at least one special character")
	}
}

// Funções utilitárias standalone

// IsEmail verifica se string é email válido
func IsEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// IsURL verifica se string é URL válida
func IsURL(urlStr string) bool {
	_, err := url.ParseRequestURI(urlStr)
	return err == nil
}

// IsPhone verifica se string é telefone válido (formato brasileiro)
func IsPhone(phone string) bool {
	phoneRegex := regexp.MustCompile(`^\(\d{2}\)\s\d{4,5}-\d{4}$`)
	return phoneRegex.MatchString(phone)
}
