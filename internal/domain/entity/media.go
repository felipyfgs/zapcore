package entity

import (
	"time"

	"github.com/uptrace/bun"
)

// MediaFile representa um arquivo de mídia armazenado no sistema
type MediaFile struct {
	bun.BaseModel `bun:"table:media_files,alias:mf"`

	ID          string    `json:"id" bun:"id,pk"`
	Filename    string    `json:"filename" bun:"filename,notnull"`
	MimeType    string    `json:"mimeType" bun:"mime_type,notnull"`
	Size        int64     `json:"size" bun:"size,notnull"`
	MessageType string    `json:"messageType" bun:"message_type,notnull"`
	FilePath    string    `json:"-" bun:"file_path,notnull"`                         // Caminho no MinIO
	SessionID   string    `json:"sessionId,omitempty" bun:"session_id,nullzero"`     // ID da sessão que fez upload (opcional)
	SessionName string    `json:"sessionName,omitempty" bun:"session_name,nullzero"` // Nome da sessão que fez upload (opcional)
	CreatedAt   time.Time `json:"createdAt" bun:"created_at,nullzero,notnull,default:current_timestamp"`
	ExpiresAt   time.Time `json:"expiresAt" bun:"expires_at,nullzero,notnull"`
}

// SendMediaMessageRequest representa a requisição para envio de mídia via WhatsApp
// Suporta múltiplas fontes: MediaID (MinIO), Base64, URL, ou Upload direto
type SendMediaMessageRequest struct {
	BaseMessageRequest

	// Múltiplas fontes de mídia (apenas uma deve ser fornecida)
	MediaID string `json:"mediaId,omitempty"` // MinIO ID existente (compatibilidade)
	Base64  string `json:"base64,omitempty"`  // Data URL base64: "data:image/jpeg;base64,..."
	URL     string `json:"url,omitempty"`     // URL pública: "https://example.com/image.jpg"
	// File via multipart será tratado separadamente no handler

	// Metadados opcionais
	Caption     string `json:"caption,omitempty"`
	MessageType string `json:"messageType,omitempty"` // Override manual (auto-detectado se não fornecido)
	Filename    string `json:"filename,omitempty"`    // Nome customizado do arquivo
}

// MediaUploadResponse representa a resposta do upload de mídia
type MediaUploadResponse struct {
	Success bool      `json:"success"`
	Message string    `json:"message"`
	Data    MediaFile `json:"data"`
}

// MediaListResponse representa a resposta da listagem de mídias
type MediaListResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    []MediaFile `json:"data"`
	Total   int         `json:"total"`
	Page    int         `json:"page"`
	Limit   int         `json:"limit"`
}

// ProcessedMedia representa mídia processada pronta para envio
type ProcessedMedia struct {
	Data           []byte        `json:"-"`              // Dados binários da mídia
	MimeType       string        `json:"mimeType"`       // Tipo MIME detectado
	MessageType    MessageType   `json:"messageType"`    // Tipo de mensagem WhatsApp
	Size           int64         `json:"size"`           // Tamanho em bytes
	Filename       string        `json:"filename"`       // Nome do arquivo
	Source         string        `json:"source"`         // Fonte: "mediaId", "base64", "url", "upload"
	ProcessingTime time.Duration `json:"processingTime"` // Tempo de processamento
}

// SendMediaMessageResponse representa a resposta unificada do envio de mídia
type SendMediaMessageResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Details   struct {
		Phone       string    `json:"phone"`
		Type        string    `json:"type"`
		Status      string    `json:"status"`
		SentAt      time.Time `json:"sentAt"`
		SessionName string    `json:"sessionName"`
		Source      string    `json:"source"`
		MediaInfo   struct {
			Filename       string `json:"filename"`
			MimeType       string `json:"mimeType"`
			OriginalSize   int64  `json:"originalSize"`
			DetectedType   string `json:"detectedType"`
			ProcessingTime string `json:"processingTime"`
		} `json:"mediaInfo"`
		WhatsappInfo struct {
			MessageID  string `json:"messageId,omitempty"`
			DirectPath string `json:"directPath,omitempty"`
			URL        string `json:"url,omitempty"`
		} `json:"whatsappInfo,omitempty"`
	} `json:"details"`
}

// NewSendMediaMessageResponse cria uma nova resposta de envio de mídia
func NewSendMediaMessageResponse(
	phone, sessionName string,
	processed *ProcessedMedia,
	whatsappMessageID, whatsappDirectPath, whatsappURL string,
) *SendMediaMessageResponse {
	now := time.Now()

	response := &SendMediaMessageResponse{
		Success:   true,
		Message:   "Media message sent successfully",
		Timestamp: now,
	}

	response.Details.Phone = phone
	response.Details.Type = string(processed.MessageType)
	response.Details.Status = "sent"
	response.Details.SentAt = now
	response.Details.SessionName = sessionName
	response.Details.Source = processed.Source

	// Informações da mídia
	response.Details.MediaInfo.Filename = processed.Filename
	response.Details.MediaInfo.MimeType = processed.MimeType
	response.Details.MediaInfo.OriginalSize = processed.Size
	response.Details.MediaInfo.DetectedType = string(processed.MessageType)
	response.Details.MediaInfo.ProcessingTime = processed.ProcessingTime.String()

	// Informações do WhatsApp
	response.Details.WhatsappInfo.MessageID = whatsappMessageID
	response.Details.WhatsappInfo.DirectPath = whatsappDirectPath
	response.Details.WhatsappInfo.URL = whatsappURL

	return response
}

// NewSendMediaMessageErrorResponse cria uma resposta de erro
func NewSendMediaMessageErrorResponse(message string, err error) *SendMediaMessageResponse {
	return &SendMediaMessageResponse{
		Success:   false,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// Constantes para tipos de mídia suportados pelo WhatsApp
const (
	// TTL padrão para arquivos (7 dias)
	DefaultMediaTTL = 7 * 24 * time.Hour
)

// Tipos MIME suportados pelo WhatsApp por categoria
var (
	SupportedImageTypes = []string{
		"image/jpeg",
		"image/png",
	}

	SupportedAudioTypes = []string{
		"audio/aac",
		"audio/mp4",
		"audio/mpeg",
		"audio/amr",
		"audio/ogg",
		"application/ogg", // OGG pode ser detectado como application/ogg
	}

	SupportedVideoTypes = []string{
		"video/mp4",
		"video/3gp",
	}

	SupportedDocumentTypes = []string{
		"text/plain",
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
	}

	SupportedStickerTypes = []string{
		"image/webp",
	}
)

// GetMaxSizeForMessageType retorna o tamanho máximo permitido para um tipo de mensagem
func GetMaxSizeForMessageType(messageType MessageType) int64 {
	switch messageType {
	case MessageTypeImage:
		return MaxImageSize
	case MessageTypeAudio:
		return MaxAudioSize
	case MessageTypeVideo:
		return MaxDocumentSize // Vídeos usam o mesmo limite de documentos
	case MessageTypeDocument:
		return MaxDocumentSize
	case MessageTypeSticker:
		return MaxStickerSize
	default:
		return MaxDocumentSize
	}
}

// GetSupportedMimeTypes retorna os tipos MIME suportados para um tipo de mensagem
func GetSupportedMimeTypes(messageType MessageType) []string {
	switch messageType {
	case MessageTypeImage:
		return SupportedImageTypes
	case MessageTypeAudio:
		return SupportedAudioTypes
	case MessageTypeVideo:
		return SupportedVideoTypes
	case MessageTypeDocument:
		return SupportedDocumentTypes
	case MessageTypeSticker:
		return SupportedStickerTypes
	default:
		return []string{}
	}
}

// DetectMessageTypeFromMime detecta o tipo de mensagem baseado no tipo MIME
func DetectMessageTypeFromMime(mimeType string) MessageType {
	for _, supportedType := range SupportedImageTypes {
		if mimeType == supportedType {
			return MessageTypeImage
		}
	}

	for _, supportedType := range SupportedAudioTypes {
		if mimeType == supportedType {
			return MessageTypeAudio
		}
	}

	for _, supportedType := range SupportedVideoTypes {
		if mimeType == supportedType {
			return MessageTypeVideo
		}
	}

	for _, supportedType := range SupportedStickerTypes {
		if mimeType == supportedType {
			return MessageTypeSticker
		}
	}

	// Default para documento se não encontrar categoria específica
	return MessageTypeDocument
}
