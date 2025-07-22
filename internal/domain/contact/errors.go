package contact

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Erros específicos do domínio de contato
var (
	ErrContactNotFound      = errors.New("contato não encontrado")
	ErrContactAlreadyExists = errors.New("contato já existe")
	ErrInvalidJID           = errors.New("JID inválido")
	ErrInvalidContactName   = errors.New("nome do contato inválido")
	ErrContactBlocked       = errors.New("contato está bloqueado")
)

// ContactError representa um erro específico de contato com contexto
type ContactError struct {
	ContactID uuid.UUID
	Operation string
	Err       error
}

func (e *ContactError) Error() string {
	return fmt.Sprintf("erro no contato %s durante %s: %v", e.ContactID, e.Operation, e.Err)
}

func (e *ContactError) Unwrap() error {
	return e.Err
}

// NewContactError cria um novo erro de contato
func NewContactError(contactID uuid.UUID, operation string, err error) *ContactError {
	return &ContactError{
		ContactID: contactID,
		Operation: operation,
		Err:       err,
	}
}
