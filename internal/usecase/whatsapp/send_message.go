package whatsapp

import (
	"fmt"

	entity "wamex/internal/domain/entity"
	domainRepo "wamex/internal/domain/repository"
	domainService "wamex/internal/domain/service"
)

// SendMessageUseCase representa o caso de uso para envio de mensagens
type SendMessageUseCase struct {
	sessionRepo domainRepo.SessionRepository
	whatsappSvc domainService.SessionService
}

// NewSendMessageUseCase cria uma nova instância do use case
func NewSendMessageUseCase(
	sessionRepo domainRepo.SessionRepository,
	whatsappSvc domainService.SessionService,
) *SendMessageUseCase {
	return &SendMessageUseCase{
		sessionRepo: sessionRepo,
		whatsappSvc: whatsappSvc,
	}
}

// SendTextMessage envia uma mensagem de texto
func (uc *SendMessageUseCase) SendTextMessage(sessionName, to, message string) error {
	// Validar sessão
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != entity.StatusConnected {
		return fmt.Errorf("session %s is not connected", sessionName)
	}

	// Enviar mensagem através do service
	return uc.whatsappSvc.SendTextMessage(sessionName, to, message)
}

// SendImageMessage envia uma mensagem de imagem
func (uc *SendMessageUseCase) SendImageMessage(sessionName, to, imageData, caption, mimeType string) error {
	// Validar sessão
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != entity.StatusConnected {
		return fmt.Errorf("session %s is not connected", sessionName)
	}

	// Enviar mensagem através do service
	return uc.whatsappSvc.SendImageMessage(sessionName, to, imageData, caption, mimeType)
}

// SendAudioMessage envia uma mensagem de áudio
func (uc *SendMessageUseCase) SendAudioMessage(sessionName, to, audioData string) error {
	// Validar sessão
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != entity.StatusConnected {
		return fmt.Errorf("session %s is not connected", sessionName)
	}

	// Enviar mensagem através do service
	return uc.whatsappSvc.SendAudioMessage(sessionName, to, audioData)
}

// SendDocumentMessage envia uma mensagem de documento
func (uc *SendMessageUseCase) SendDocumentMessage(sessionName, to, documentData, filename, mimeType string) error {
	// Validar sessão
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != entity.StatusConnected {
		return fmt.Errorf("session %s is not connected", sessionName)
	}

	// Enviar mensagem através do service
	return uc.whatsappSvc.SendDocumentMessage(sessionName, to, documentData, filename, mimeType)
}

// SendLocationMessage envia uma mensagem de localização
func (uc *SendMessageUseCase) SendLocationMessage(sessionName, to string, latitude, longitude float64, name string) error {
	// Validar sessão
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != entity.StatusConnected {
		return fmt.Errorf("session %s is not connected", sessionName)
	}

	// Enviar mensagem através do service
	return uc.whatsappSvc.SendLocationMessage(sessionName, to, latitude, longitude, name)
}

// SendContactMessage envia uma mensagem de contato
func (uc *SendMessageUseCase) SendContactMessage(sessionName, to, name, vcard string) error {
	// Validar sessão
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != entity.StatusConnected {
		return fmt.Errorf("session %s is not connected", sessionName)
	}

	// Enviar mensagem através do service
	return uc.whatsappSvc.SendContactMessage(sessionName, to, name, vcard)
}

// SendStickerMessage envia uma mensagem de sticker
func (uc *SendMessageUseCase) SendStickerMessage(sessionName, to, stickerData string) error {
	// Validar sessão
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != entity.StatusConnected {
		return fmt.Errorf("session %s is not connected", sessionName)
	}

	// Enviar mensagem através do service
	return uc.whatsappSvc.SendStickerMessage(sessionName, to, stickerData)
}

// SendPollMessage envia uma mensagem de enquete
func (uc *SendMessageUseCase) SendPollMessage(sessionName, to, header string, options []string, maxSelections int) error {
	// Validar sessão
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != entity.StatusConnected {
		return fmt.Errorf("session %s is not connected", sessionName)
	}

	// Enviar mensagem através do service
	return uc.whatsappSvc.SendPollMessage(sessionName, to, header, options, maxSelections)
}

// SendListMessage envia uma mensagem de lista interativa
func (uc *SendMessageUseCase) SendListMessage(sessionName, to, header, body, footer, buttonText string, sections []entity.ListSection) error {
	// Validar sessão
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != entity.StatusConnected {
		return fmt.Errorf("session %s is not connected", sessionName)
	}

	// Enviar mensagem através do service
	return uc.whatsappSvc.SendListMessage(sessionName, to, header, body, footer, buttonText, sections)
}
