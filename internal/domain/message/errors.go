package message

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Erros específicos do domínio de mensagem
var (
	ErrMessageNotFound      = errors.New("mensagem não encontrada")
	ErrMessageAlreadyExists = errors.New("mensagem já existe")
	ErrInvalidMessageType   = errors.New("tipo de mensagem inválido")
	ErrInvalidContent       = errors.New("conteúdo da mensagem inválido")
	ErrInvalidJID           = errors.New("JID inválido")
	ErrMessageTooLarge      = errors.New("mensagem muito grande")
	ErrMediaNotFound        = errors.New("mídia não encontrada")
	ErrInvalidMediaType     = errors.New("tipo de mídia inválido")
	ErrMessageSendFailed    = errors.New("falha ao enviar mensagem")
	ErrMessageEditFailed    = errors.New("falha ao editar mensagem")
)

// MessageError representa um erro específico de mensagem com contexto
type MessageError struct {
	MessageID uuid.UUID
	Operation string
	Err       error
}

func (e *MessageError) Error() string {
	return fmt.Sprintf("erro na mensagem %s durante %s: %v", e.MessageID, e.Operation, e.Err)
}

func (e *MessageError) Unwrap() error {
	return e.Err
}

// NewMessageError cria um novo erro de mensagem
func NewMessageError(messageID uuid.UUID, operation string, err error) *MessageError {
	return &MessageError{
		MessageID: messageID,
		Operation: operation,
		Err:       err,
	}
}

// MediaError representa um erro relacionado à mídia
type MediaError struct {
	MediaID   uuid.UUID
	MediaType string
	Err       error
}

func (e *MediaError) Error() string {
	return fmt.Sprintf("erro na mídia %s (tipo: %s): %v", e.MediaID, e.MediaType, e.Err)
}

func (e *MediaError) Unwrap() error {
	return e.Err
}

// NewMediaError cria um novo erro de mídia
func NewMediaError(mediaID uuid.UUID, mediaType string, err error) *MediaError {
	return &MediaError{
		MediaID:   mediaID,
		MediaType: mediaType,
		Err:       err,
	}
}

