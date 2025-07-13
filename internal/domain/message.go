package domain

import (
	"time"
)

// MessageType representa os tipos de mensagem suportados
type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypeAudio    MessageType = "audio"
	MessageTypeDocument MessageType = "document"
	MessageTypeSticker  MessageType = "sticker"
)

// Constantes para tipos MIME suportados
const (
	// Imagens
	MimeTypeImageJPEG = "image/jpeg"
	MimeTypeImagePNG  = "image/png"
	MimeTypeImageWebP = "image/webp"

	// Áudio
	MimeTypeAudioOGG     = "audio/ogg"
	MimeTypeAudioOGGOpus = "audio/ogg; codecs=opus" // Formato específico do WhatsApp
	MimeTypeAudioMP3     = "audio/mp3"
	MimeTypeAudioMPEG    = "audio/mpeg" // MP3 padrão
	MimeTypeAudioWAV     = "audio/wav"
	MimeTypeAudioAAC     = "audio/aac"

	// Documentos
	MimeTypePDF  = "application/pdf"
	MimeTypeDocx = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	MimeTypeXlsx = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	MimeTypeTxt  = "text/plain"
)

// Constantes para tamanhos máximos de arquivo (em bytes)
const (
	MaxImageSize    = 16 * 1024 * 1024  // 16MB
	MaxAudioSize    = 16 * 1024 * 1024  // 16MB
	MaxDocumentSize = 100 * 1024 * 1024 // 100MB
	MaxStickerSize  = 500 * 1024        // 500KB
)

// BaseMessageRequest estrutura base para todas as mensagens
type BaseMessageRequest struct {
	Phone string `json:"phone" validate:"required,min=10,max=15"`
	ID    string `json:"id,omitempty"`
}

// SendTextMessageRequest estrutura para envio de mensagem de texto
type SendTextMessageRequest struct {
	BaseMessageRequest
	Body string `json:"body" validate:"required,min=1,max=4096"`
}

// SendImageMessageRequest estrutura para envio de imagem
type SendImageMessageRequest struct {
	BaseMessageRequest
	// Múltiplas formas de especificar a imagem (apenas uma deve ser fornecida)
	Image    string `json:"image,omitempty"`    // base64: "data:image/png;base64,..."
	FilePath string `json:"filePath,omitempty"` // Caminho local: "assets/image.png"
	URL      string `json:"url,omitempty"`      // URL externa: "https://example.com/image.png"
	MinioID  string `json:"minioId,omitempty"`  // ID no MinIO: "media/2025/01/02/image_123.png"

	Caption  string `json:"caption,omitempty"`  // Legenda opcional
	MimeType string `json:"mimeType,omitempty"` // Tipo MIME (detectado automaticamente se não fornecido)
	Filename string `json:"filename,omitempty"` // Nome do arquivo opcional
}

// SendAudioMessageRequest estrutura para envio de áudio
type SendAudioMessageRequest struct {
	BaseMessageRequest
	// Múltiplas formas de especificar o áudio (apenas uma deve ser fornecida)
	Audio    string `json:"audio,omitempty"`    // base64: "data:audio/ogg;base64,..."
	FilePath string `json:"filePath,omitempty"` // Caminho local: "assets/audio.ogg"
	URL      string `json:"url,omitempty"`      // URL externa: "https://example.com/audio.ogg"
	MinioID  string `json:"minioId,omitempty"`  // ID no MinIO: "media/2025/01/02/audio_123.ogg"

	Caption  string `json:"caption,omitempty"`  // Legenda opcional
	PTT      bool   `json:"ptt,omitempty"`      // Push-to-Talk (mensagem de voz)
	MimeType string `json:"mimeType,omitempty"` // Tipo MIME
	Duration int    `json:"duration,omitempty"` // Duração em segundos
}

// SendDocumentMessageRequest estrutura para envio de documento
type SendDocumentMessageRequest struct {
	BaseMessageRequest
	// Múltiplas formas de especificar o documento (apenas uma deve ser fornecida)
	Document string `json:"document,omitempty"` // base64: "data:application/pdf;base64,..."
	FilePath string `json:"filePath,omitempty"` // Caminho local: "assets/document.pdf"
	URL      string `json:"url,omitempty"`      // URL externa: "https://example.com/document.pdf"
	MinioID  string `json:"minioId,omitempty"`  // ID no MinIO: "media/2025/01/02/document_123.pdf"

	Filename string `json:"filename" validate:"required"` // Nome do arquivo obrigatório
	Caption  string `json:"caption,omitempty"`            // Legenda opcional
	MimeType string `json:"mimeType,omitempty"`           // Tipo MIME (detectado automaticamente se não fornecido)
}

// SendStickerMessageRequest estrutura para envio de sticker
type SendStickerMessageRequest struct {
	BaseMessageRequest
	// Múltiplas formas de especificar o sticker (apenas uma deve ser fornecida)
	Sticker  string `json:"sticker,omitempty"`  // base64: "data:image/webp;base64,..."
	FilePath string `json:"filePath,omitempty"` // Caminho local: "assets/sticker.webp"
	URL      string `json:"url,omitempty"`      // URL externa: "https://example.com/sticker.webp"
	MinioID  string `json:"minioId,omitempty"`  // ID no MinIO: "media/2025/01/02/sticker_123.webp"

	MimeType string `json:"mimeType,omitempty"` // Deve ser image/webp
}

// MessageResponse estrutura de resposta padronizada para envio de mensagens
type MessageResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	Details   *Details  `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	ID        string    `json:"id,omitempty"`
}

// Details informações detalhadas da mensagem enviada
type Details struct {
	MessageID   string      `json:"messageId,omitempty"`
	Phone       string      `json:"phone"`
	Type        MessageType `json:"type"`
	Status      string      `json:"status"`
	SentAt      time.Time   `json:"sentAt"`
	SessionName string      `json:"sessionName,omitempty"`
	MediaInfo   *MediaInfo  `json:"mediaInfo,omitempty"`
}

// MediaInfo informações sobre mídia enviada
type MediaInfo struct {
	OriginalSize int64  `json:"originalSize,omitempty"`
	MimeType     string `json:"mimeType,omitempty"`
	Filename     string `json:"filename,omitempty"`
	Duration     int    `json:"duration,omitempty"`     // Para áudio
	Dimensions   string `json:"dimensions,omitempty"`   // Para imagem (ex: "1920x1080")
	ThumbnailURL string `json:"thumbnailUrl,omitempty"` // URL do thumbnail
	MediaURL     string `json:"mediaUrl,omitempty"`     // URL da mídia
}

// MessageError estrutura de erro específica para mensagens
type MessageError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

// Códigos de erro para mensagens
const (
	ErrorCodeInvalidPhone    = "INVALID_PHONE"
	ErrorCodeInvalidBase64   = "INVALID_BASE64"
	ErrorCodeUnsupportedMime = "UNSUPPORTED_MIME_TYPE"
	ErrorCodeFileTooLarge    = "FILE_TOO_LARGE"
	ErrorCodeSessionNotFound = "SESSION_NOT_FOUND"
	ErrorCodeSessionOffline  = "SESSION_OFFLINE"
	ErrorCodeUploadFailed    = "UPLOAD_FAILED"
	ErrorCodeSendFailed      = "SEND_FAILED"
)

// ValidMimeTypes mapas de tipos MIME válidos por tipo de mensagem
var ValidMimeTypes = map[MessageType][]string{
	MessageTypeImage: {
		MimeTypeImageJPEG,
		MimeTypeImagePNG,
		MimeTypeImageWebP,
	},
	MessageTypeAudio: {
		MimeTypeAudioOGG,
		MimeTypeAudioOGGOpus, // Formato específico do WhatsApp
		MimeTypeAudioMP3,
		MimeTypeAudioMPEG, // MP3 padrão
		MimeTypeAudioWAV,
		MimeTypeAudioAAC,
		"application/ogg", // OGG detectado como application
	},
	MessageTypeDocument: {
		MimeTypePDF,
		MimeTypeDocx,
		MimeTypeXlsx,
		MimeTypeTxt,
		"application/msword",
		"application/vnd.ms-excel",
		"application/vnd.ms-powerpoint",
	},
	MessageTypeSticker: {
		MimeTypeImageWebP,
	},
}

// MaxFileSizes tamanhos máximos por tipo de mensagem
var MaxFileSizes = map[MessageType]int64{
	MessageTypeImage:    MaxImageSize,
	MessageTypeAudio:    MaxAudioSize,
	MessageTypeDocument: MaxDocumentSize,
	MessageTypeSticker:  MaxStickerSize,
}

// IsValidMimeType verifica se um tipo MIME é válido para um tipo de mensagem
func IsValidMimeType(messageType MessageType, mimeType string) bool {
	validTypes, exists := ValidMimeTypes[messageType]
	if !exists {
		return false
	}

	for _, validType := range validTypes {
		if validType == mimeType {
			return true
		}
	}
	return false
}

// GetMaxFileSize retorna o tamanho máximo permitido para um tipo de mensagem
func GetMaxFileSize(messageType MessageType) int64 {
	if size, exists := MaxFileSizes[messageType]; exists {
		return size
	}
	return 0
}
