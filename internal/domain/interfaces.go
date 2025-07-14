package domain

import (
	"context"
	"time"
)

// SessionRepository define a interface para operações de sessão no banco de dados
type SessionRepository interface {
	Create(session *Session) error
	GetByID(id string) (*Session, error)
	GetBySession(sessionName string) (*Session, error)
	GetByToken(token string) (*Session, error)
	Update(session *Session) error
	Delete(id string) error
	DeleteBySession(sessionName string) error
	List() ([]*Session, error)
	GetActive() ([]*Session, error)
	GetConnectedSessions() ([]*Session, error)
}

// SessionService define a interface para serviços de sessão WhatsApp
type SessionService interface {
	CreateSession(req *CreateSessionRequest) (*Session, error)
	GetSession(sessionName string) (*Session, error)
	GetSessionByID(sessionID string) (*Session, error)
	UpdateSessionStatus(id string, status Status) error
	DeleteSession(sessionName string) error
	ListSessions() ([]*Session, error)
	GenerateQRCode(sessionName string) (string, error)
	ConnectSession(sessionName string) error
	DisconnectSession(sessionName string) error
	PairPhone(sessionName string, phone string) error
	GetSessionStatus(sessionName string) (*StatusResponse, error)
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
	SendListMessage(sessionName, to, header, body, footer, buttonText string, sections []ListSection) error
}

// MediaRepository define a interface para operações de mídia no banco de dados
type MediaRepository interface {
	Create(ctx context.Context, mediaFile *MediaFile) error
	GetByID(ctx context.Context, id string) (*MediaFile, error)
	List(ctx context.Context, limit, offset int, messageType, sessionID, sessionName string) ([]MediaFile, int, error)
	Delete(ctx context.Context, id string) error
	GetExpiredFiles(ctx context.Context) ([]MediaFile, error)
	UpdateExpiresAt(ctx context.Context, id string, expiresAt time.Time) error
	GetByMessageType(ctx context.Context, messageType string, limit int) ([]MediaFile, error)
	GetStats(ctx context.Context) (map[string]interface{}, error)
	CleanupExpired(ctx context.Context) (int, error)
}

// MediaService define a interface para serviços de processamento de mídia
type MediaService interface {
	DecodeBase64Media(dataURL string) ([]byte, string, error)
	ValidateMediaType(mimeType string, messageType MessageType) error
	ValidateFileSize(data []byte, messageType MessageType) error
	DetectMimeType(data []byte) string
	ProcessImageForWhatsApp(data []byte, mimeType string) ([]byte, error)
	ProcessMediaForUpload(dataURL string, messageType MessageType) (*ProcessedMedia, error)
	UploadToWhatsApp(client interface{}, data []byte, mimeType string, messageType MessageType) (interface{}, error)
}

// StorageService define a interface para serviços de armazenamento (MinIO/S3)
type StorageService interface {
	UploadFile(bucketName, objectName string, data []byte, contentType string) error
	DownloadFile(bucketName, objectName string) ([]byte, error)
	DeleteFile(bucketName, objectName string) error
	GetFileURL(bucketName, objectName string, expiry time.Duration) (string, error)
	FileExists(bucketName, objectName string) (bool, error)
}

// ValidationService define a interface para validações de segurança
type ValidationService interface {
	ValidateURL(url string) error
	ValidateFileContent(data []byte, expectedMimeType string) error
	CheckRateLimit(identifier string) error
	ScanForMalware(data []byte) error
}

// TypeDetector define a interface para detecção automática de tipos
type TypeDetector interface {
	DetectFromData(data []byte) (MessageType, string, error)
	DetectFromURL(url string) (MessageType, string, error)
	DetectFromFilename(filename string) (MessageType, string, error)
	IsValidType(messageType MessageType, mimeType string) bool
}

// ProcessedMedia é definida em media.go
