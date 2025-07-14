package whatsapp

import (
	"fmt"

	entity "wamex/internal/domain/entity"
	domainRepo "wamex/internal/domain/repository"
	domainService "wamex/internal/domain/service"
)

// ProcessMediaUseCase representa o caso de uso para processamento de mídia do WhatsApp
type ProcessMediaUseCase struct {
	sessionRepo domainRepo.SessionRepository
	mediaRepo   domainRepo.MediaRepository
	whatsappSvc domainService.SessionService
	mediaSvc    domainService.MediaService
}

// NewProcessMediaUseCase cria uma nova instância do use case
func NewProcessMediaUseCase(
	sessionRepo domainRepo.SessionRepository,
	mediaRepo domainRepo.MediaRepository,
	whatsappSvc domainService.SessionService,
	mediaSvc domainService.MediaService,
) *ProcessMediaUseCase {
	return &ProcessMediaUseCase{
		sessionRepo: sessionRepo,
		mediaRepo:   mediaRepo,
		whatsappSvc: whatsappSvc,
		mediaSvc:    mediaSvc,
	}
}

// SendImageMessageMultiSource envia mensagem de imagem com múltiplas fontes
func (uc *ProcessMediaUseCase) SendImageMessageMultiSource(
	sessionName, to, base64Data, filePath, url, minioID, caption, mimeType string,
) error {
	// Validar sessão
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != entity.StatusConnected {
		return fmt.Errorf("session %s is not connected", sessionName)
	}

	// Processar mídia através do service
	return uc.whatsappSvc.SendImageMessageMultiSource(
		sessionName, to, base64Data, filePath, url, minioID, caption, mimeType,
	)
}

// SendAudioMessageMultiSource envia mensagem de áudio com múltiplas fontes
func (uc *ProcessMediaUseCase) SendAudioMessageMultiSource(
	sessionName, to, base64Data, filePath, url, minioID string,
) error {
	// Validar sessão
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != entity.StatusConnected {
		return fmt.Errorf("session %s is not connected", sessionName)
	}

	// Processar mídia através do service
	return uc.whatsappSvc.SendAudioMessageMultiSource(
		sessionName, to, base64Data, filePath, url, minioID,
	)
}

// SendDocumentMessageMultiSource envia mensagem de documento com múltiplas fontes
func (uc *ProcessMediaUseCase) SendDocumentMessageMultiSource(
	sessionName, to, base64Data, filePath, url, minioID, filename, mimeType string,
) error {
	// Validar sessão
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != entity.StatusConnected {
		return fmt.Errorf("session %s is not connected", sessionName)
	}

	// Processar mídia através do service
	return uc.whatsappSvc.SendDocumentMessageMultiSource(
		sessionName, to, base64Data, filePath, url, minioID, filename, mimeType,
	)
}

// SendVideoMessageMultiSource envia mensagem de vídeo com múltiplas fontes
func (uc *ProcessMediaUseCase) SendVideoMessageMultiSource(
	sessionName, to, base64Data, filePath, url, minioID, caption, mimeType string,
	jpegThumbnail []byte,
) error {
	// Validar sessão
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != entity.StatusConnected {
		return fmt.Errorf("session %s is not connected", sessionName)
	}

	// Processar mídia através do service
	return uc.whatsappSvc.SendVideoMessageMultiSource(
		sessionName, to, base64Data, filePath, url, minioID, caption, mimeType, jpegThumbnail,
	)
}

// ProcessMediaForUpload processa mídia para upload
func (uc *ProcessMediaUseCase) ProcessMediaForUpload(
	dataURL string, messageType entity.MessageType,
) (*entity.ProcessedMedia, error) {
	// Processar mídia através do service
	processed, err := uc.mediaSvc.ProcessMediaForUpload(dataURL, messageType)
	if err != nil {
		return nil, fmt.Errorf("failed to process media: %w", err)
	}

	return processed, nil
}

// ValidateMediaType valida o tipo de mídia
func (uc *ProcessMediaUseCase) ValidateMediaType(
	mimeType string, messageType entity.MessageType,
) error {
	return uc.mediaSvc.ValidateMediaType(mimeType, messageType)
}

// ValidateFileSize valida o tamanho do arquivo
func (uc *ProcessMediaUseCase) ValidateFileSize(
	data []byte, messageType entity.MessageType,
) error {
	return uc.mediaSvc.ValidateFileSize(data, messageType)
}

// DecodeBase64Media decodifica mídia em base64
func (uc *ProcessMediaUseCase) DecodeBase64Media(dataURL string) ([]byte, string, error) {
	return uc.mediaSvc.DecodeBase64Media(dataURL)
}

// DetectMimeType detecta o tipo MIME de um arquivo
func (uc *ProcessMediaUseCase) DetectMimeType(data []byte) string {
	return uc.mediaSvc.DetectMimeType(data)
}

// ProcessImageForWhatsApp processa imagem para o WhatsApp
func (uc *ProcessMediaUseCase) ProcessImageForWhatsApp(data []byte, mimeType string) ([]byte, error) {
	return uc.mediaSvc.ProcessImageForWhatsApp(data, mimeType)
}

// ReactToMessage reage a uma mensagem
func (uc *ProcessMediaUseCase) ReactToMessage(sessionName, to, messageID, reaction string) error {
	// Validar sessão
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != entity.StatusConnected {
		return fmt.Errorf("session %s is not connected", sessionName)
	}

	// Reagir através do service
	return uc.whatsappSvc.ReactToMessage(sessionName, to, messageID, reaction)
}

// EditMessage edita uma mensagem
func (uc *ProcessMediaUseCase) EditMessage(sessionName, to, messageID, newText string) error {
	// Validar sessão
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != entity.StatusConnected {
		return fmt.Errorf("session %s is not connected", sessionName)
	}

	// Editar através do service
	return uc.whatsappSvc.EditMessage(sessionName, to, messageID, newText)
}
