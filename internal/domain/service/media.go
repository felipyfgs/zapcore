package domain

import (
	"time"
	entity "wamex/internal/domain/entity"
)

// MediaService define a interface para serviços de processamento de mídia
type MediaService interface {
	DecodeBase64Media(dataURL string) ([]byte, string, error)
	ValidateMediaType(mimeType string, messageType entity.MessageType) error
	ValidateFileSize(data []byte, messageType entity.MessageType) error
	DetectMimeType(data []byte) string
	ProcessImageForWhatsApp(data []byte, mimeType string) ([]byte, error)
	ProcessMediaForUpload(dataURL string, messageType entity.MessageType) (*entity.ProcessedMedia, error)
	UploadToWhatsApp(client interface{}, data []byte, mimeType string, messageType entity.MessageType) (interface{}, error)
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
	DetectFromData(data []byte) (entity.MessageType, string, error)
	DetectFromURL(url string) (entity.MessageType, string, error)
	DetectFromFilename(filename string) (entity.MessageType, string, error)
	IsValidType(messageType entity.MessageType, mimeType string) bool
}
