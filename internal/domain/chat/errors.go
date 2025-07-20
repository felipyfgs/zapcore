package chat

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Erros específicos do domínio de chat
var (
	ErrChatNotFound      = errors.New("chat não encontrado")
	ErrChatAlreadyExists = errors.New("chat já existe")
	ErrInvalidChatType   = errors.New("tipo de chat inválido")
	ErrInvalidJID        = errors.New("JID inválido")
	ErrChatArchived      = errors.New("chat está arquivado")
	ErrChatMuted         = errors.New("chat está silenciado")
	ErrInvalidChatName   = errors.New("nome do chat inválido")
)

// ChatError representa um erro específico de chat com contexto
type ChatError struct {
	ChatID    uuid.UUID
	Operation string
	Err       error
}

func (e *ChatError) Error() string {
	return fmt.Sprintf("erro no chat %s durante %s: %v", e.ChatID, e.Operation, e.Err)
}

func (e *ChatError) Unwrap() error {
	return e.Err
}

// NewChatError cria um novo erro de chat
func NewChatError(chatID uuid.UUID, operation string, err error) *ChatError {
	return &ChatError{
		ChatID:    chatID,
		Operation: operation,
		Err:       err,
	}
}

