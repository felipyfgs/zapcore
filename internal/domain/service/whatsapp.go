package domain

import (
	entity "wamex/internal/domain/entity"
)

// SessionService define a interface para serviços de sessão WhatsApp
type SessionService interface {
	CreateSession(req *entity.CreateSessionRequest) (*entity.Session, error)
	GetSession(sessionName string) (*entity.Session, error)
	GetSessionByID(sessionID string) (*entity.Session, error)
	UpdateSessionStatus(id string, status entity.Status) error
	DeleteSession(sessionName string) error
	ListSessions() ([]*entity.Session, error)
	GenerateQRCode(sessionName string) (string, error)
	ConnectSession(sessionName string) error
	DisconnectSession(sessionName string) error
	PairPhone(sessionName string, phone string) error
	GetSessionStatus(sessionName string) (*entity.StatusResponse, error)
	GetConnectedSessionsCount() int
	SendTextMessage(sessionName, to, message string) error
	SendImageMessage(sessionName, to, imageData, caption, mimeType string) error
	SendImageMessageMultiSource(sessionName, to, base64Data, filePath, url, minioID, caption, mimeType string) error
	SendAudioMessage(sessionName, to, audioData string) error
	SendAudioMessageMultiSource(sessionName, to, base64Data, filePath, url, minioID string) error
	SendDocumentMessage(sessionName, to, documentData, filename, mimeType string) error
	SendDocumentMessageMultiSource(sessionName, to, base64Data, filePath, url, minioID, filename, mimeType string) error
	SendStickerMessage(sessionName, to, stickerData string) error
	SendLocationMessage(sessionName, to string, latitude, longitude float64, name string) error
	SendContactMessage(sessionName, to, name, vcard string) error
	ReactToMessage(sessionName, to, messageID, reaction string) error
	SendVideoMessageMultiSource(sessionName, to, base64Data, filePath, url, minioID, caption, mimeType string, jpegThumbnail []byte) error
	EditMessage(sessionName, to, messageID, newText string) error
	SendPollMessage(sessionName, to, header string, options []string, maxSelections int) error
	SendListMessage(sessionName, to, header, body, footer, buttonText string, sections []entity.ListSection) error
}

// SendMessageUseCaseInterface define a interface para o use case de envio de mensagens
type SendMessageUseCaseInterface interface {
	SendTextMessage(sessionName, to, message string) error
	SendImageMessage(sessionName, to, imageData, caption, mimeType string) error
	SendAudioMessage(sessionName, to, audioData string) error
	SendDocumentMessage(sessionName, to, documentData, filename, mimeType string) error
	SendLocationMessage(sessionName, to string, latitude, longitude float64, name string) error
	SendContactMessage(sessionName, to, name, vcard string) error
	SendStickerMessage(sessionName, to, stickerData string) error
	SendPollMessage(sessionName, to, header string, options []string, maxSelections int) error
	SendListMessage(sessionName, to, header, body, footer, buttonText string, sections []entity.ListSection) error
}

// ManageSessionUseCaseInterface define a interface para o use case de gerenciamento de sessões
type ManageSessionUseCaseInterface interface {
	CreateSession(req *entity.CreateSessionRequest) (*entity.Session, error)
	ConnectSession(sessionName string) error
	DisconnectSession(sessionName string) error
	DeleteSession(sessionName string) error
	GetSession(sessionName string) (*entity.Session, error)
	ListSessions() ([]*entity.Session, error)
	GenerateQRCode(sessionName string) (string, error)
	PairPhone(sessionName, phone string) error
	GetSessionStatus(sessionName string) (*entity.StatusResponse, error)
	GetConnectedSessionsCount() int
}

// ProcessMediaUseCaseInterface define a interface para o use case de processamento de mídia
type ProcessMediaUseCaseInterface interface {
	SendImageMessageMultiSource(sessionName, to, base64Data, filePath, url, minioID, caption, mimeType string) error
	SendAudioMessageMultiSource(sessionName, to, base64Data, filePath, url, minioID string) error
	SendDocumentMessageMultiSource(sessionName, to, base64Data, filePath, url, minioID, filename, mimeType string) error
	SendVideoMessageMultiSource(sessionName, to, base64Data, filePath, url, minioID, caption, mimeType string, jpegThumbnail []byte) error
	ProcessMediaForUpload(dataURL string, messageType entity.MessageType) (*entity.ProcessedMedia, error)
	ValidateMediaType(mimeType string, messageType entity.MessageType) error
	ValidateFileSize(data []byte, messageType entity.MessageType) error
	DecodeBase64Media(dataURL string) ([]byte, string, error)
	DetectMimeType(data []byte) string
	ProcessImageForWhatsApp(data []byte, mimeType string) ([]byte, error)
	ReactToMessage(sessionName, to, messageID, reaction string) error
	EditMessage(sessionName, to, messageID, newText string) error
}
