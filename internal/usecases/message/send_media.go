package message

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"zapcore/internal/domain/message"
	"zapcore/internal/domain/session"
	"zapcore/internal/domain/whatsapp"
	"zapcore/pkg/logger"

	"github.com/google/uuid"
)

// Constantes para limites de tamanho de mídia (em bytes)
const (
	MaxImageSize    = 16 * 1024 * 1024  // 16MB
	MaxVideoSize    = 64 * 1024 * 1024  // 64MB
	MaxAudioSize    = 16 * 1024 * 1024  // 16MB
	MaxDocumentSize = 100 * 1024 * 1024 // 100MB
	MaxStickerSize  = 500 * 1024        // 500KB
)

// Tipos MIME suportados por tipo de mídia
var (
	SupportedImageMimes = []string{
		"image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp",
	}
	SupportedVideoMimes = []string{
		"video/mp4", "video/avi", "video/mov", "video/mkv", "video/webm",
	}
	SupportedAudioMimes = []string{
		"audio/mpeg", "audio/mp3", "audio/wav", "audio/ogg", "audio/aac", "audio/m4a",
	}
	SupportedDocumentMimes = []string{
		"application/pdf", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint", "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"text/plain", "application/zip", "application/rar",
	}
	SupportedStickerMimes = []string{
		"image/webp", "image/png",
	}
)

// SendMediaUseCase representa o caso de uso para enviar mídia
type SendMediaUseCase struct {
	messageRepo    message.Repository
	sessionRepo    session.Repository
	whatsappClient whatsapp.Client
	logger         *logger.Logger
}

// NewSendMediaUseCase cria uma nova instância do caso de uso
func NewSendMediaUseCase(
	messageRepo message.Repository,
	sessionRepo session.Repository,
	whatsappClient whatsapp.Client,
) *SendMediaUseCase {
	return &SendMediaUseCase{
		messageRepo:    messageRepo,
		sessionRepo:    sessionRepo,
		whatsappClient: whatsappClient,
		logger:         logger.Get(),
	}
}

// SendMediaRequest representa a requisição para enviar mídia
type SendMediaRequest struct {
	SessionID  uuid.UUID           `json:"sessionId" validate:"required"`
	ToJID      string              `json:"to_jid" validate:"required"`
	Type       message.MessageType `json:"type" validate:"required"`
	MediaData  io.Reader           `json:"-"`
	MediaURL   string              `json:"media_url,omitempty"`
	Base64Data string              `json:"base64_data,omitempty"` // Dados em base64
	Caption    string              `json:"caption,omitempty"`
	FileName   string              `json:"file_name,omitempty"`
	MimeType   string              `json:"mime_type,omitempty"`
	ReplyToID  string              `json:"reply_to_id,omitempty"`
}

// SendMediaResponse representa a resposta do envio de mídia
type SendMediaResponse struct {
	WhatsAppID string                `json:"whatsapp_id"`
	Status     message.MessageStatus `json:"status"`
	Timestamp  string                `json:"timestamp"`
	Message    string                `json:"message"`
}

// Execute executa o caso de uso de envio de mídia
func (uc *SendMediaUseCase) Execute(ctx context.Context, req *SendMediaRequest) (*SendMediaResponse, error) {
	// Verificar se a sessão existe e está conectada
	sess, err := uc.sessionRepo.GetByID(ctx, req.SessionID)
	if err != nil {
		if err == session.ErrSessionNotFound {
			return nil, err
		}
		uc.logger.Error().Err(err).Msg("erro ao buscar sessão")
		return nil, fmt.Errorf("erro interno do servidor")
	}

	if !sess.IsActive {
		return nil, session.ErrSessionNotActive
	}

	if !sess.IsConnected() {
		return nil, session.ErrSessionNotConnected
	}

	// Validar tipo de mídia
	if !isValidMediaType(req.Type) {
		return nil, message.ErrInvalidMediaType
	}

	// Validar entrada de mídia (deve ter dados, URL ou base64)
	if req.MediaData == nil && req.MediaURL == "" && req.Base64Data == "" {
		return nil, fmt.Errorf("é necessário fornecer dados de mídia, URL ou base64")
	}

	// Validar URL se fornecida
	if req.MediaURL != "" {
		if err := validateMediaURL(req.MediaURL); err != nil {
			return nil, fmt.Errorf("URL inválida: %w", err)
		}
	}

	// Validar tipo MIME se fornecido
	if req.MimeType != "" {
		if err := validateMimeType(req.Type, req.MimeType); err != nil {
			return nil, fmt.Errorf("tipo MIME inválido: %w", err)
		}
	}

	// Nota: Validação de tamanho será feita no cliente WhatsApp durante o processamento
	// para evitar consumir o Reader aqui

	uc.logger.Debug().
		Str("session_id", req.SessionID.String()).
		Str("to_jid", req.ToJID).
		Str("media_type", string(req.Type)).
		Msg("Preparando envio de mídia")

	// Enviar via WhatsApp baseado no tipo
	var whatsappResp *whatsapp.MessageResponse

	switch req.Type {
	case message.MessageTypeImage:
		whatsappReq := &whatsapp.SendImageRequest{
			SessionID:  req.SessionID,
			ToJID:      req.ToJID,
			ImageData:  req.MediaData,
			ImageURL:   req.MediaURL,
			Base64Data: req.Base64Data,
			Caption:    req.Caption,
			ReplyToID:  req.ReplyToID,
			MimeType:   req.MimeType,
			FileName:   req.FileName,
		}
		whatsappResp, err = uc.whatsappClient.SendImageMessage(ctx, whatsappReq)

	case message.MessageTypeAudio:
		whatsappReq := &whatsapp.SendAudioRequest{
			SessionID:  req.SessionID,
			ToJID:      req.ToJID,
			AudioData:  req.MediaData,
			AudioURL:   req.MediaURL,
			Base64Data: req.Base64Data,
			ReplyToID:  req.ReplyToID,
			MimeType:   req.MimeType,
			FileName:   req.FileName,
		}
		whatsappResp, err = uc.whatsappClient.SendAudioMessage(ctx, whatsappReq)

	case message.MessageTypeVideo:
		whatsappReq := &whatsapp.SendVideoRequest{
			SessionID:  req.SessionID,
			ToJID:      req.ToJID,
			VideoData:  req.MediaData,
			VideoURL:   req.MediaURL,
			Base64Data: req.Base64Data,
			Caption:    req.Caption,
			ReplyToID:  req.ReplyToID,
			MimeType:   req.MimeType,
			FileName:   req.FileName,
		}
		whatsappResp, err = uc.whatsappClient.SendVideoMessage(ctx, whatsappReq)

	case message.MessageTypeDocument:
		whatsappReq := &whatsapp.SendDocumentRequest{
			SessionID:    req.SessionID,
			ToJID:        req.ToJID,
			DocumentData: req.MediaData,
			DocumentURL:  req.MediaURL,
			Base64Data:   req.Base64Data,
			FileName:     req.FileName,
			Caption:      req.Caption,
			ReplyToID:    req.ReplyToID,
			MimeType:     req.MimeType,
		}
		whatsappResp, err = uc.whatsappClient.SendDocumentMessage(ctx, whatsappReq)

	case message.MessageTypeSticker:
		whatsappReq := &whatsapp.SendStickerRequest{
			SessionID:   req.SessionID,
			ToJID:       req.ToJID,
			StickerData: req.MediaData,
			StickerURL:  req.MediaURL,
			Base64Data:  req.Base64Data,
			ReplyToID:   req.ReplyToID,
			MimeType:    req.MimeType,
		}
		whatsappResp, err = uc.whatsappClient.SendStickerMessage(ctx, whatsappReq)

	default:
		return nil, message.ErrInvalidMediaType
	}

	if err != nil {
		uc.logger.Error().
			Err(err).
			Str("session_id", req.SessionID.String()).
			Str("to_jid", req.ToJID).
			Str("media_type", string(req.Type)).
			Msg("erro ao enviar mídia")

		// Retornar erro mais específico baseado no tipo
		switch req.Type {
		case message.MessageTypeImage:
			return nil, fmt.Errorf("erro ao enviar imagem: %w", err)
		case message.MessageTypeVideo:
			return nil, fmt.Errorf("erro ao enviar vídeo: %w", err)
		case message.MessageTypeAudio:
			return nil, fmt.Errorf("erro ao enviar áudio: %w", err)
		case message.MessageTypeDocument:
			return nil, fmt.Errorf("erro ao enviar documento: %w", err)
		case message.MessageTypeSticker:
			return nil, fmt.Errorf("erro ao enviar sticker: %w", err)
		default:
			return nil, fmt.Errorf("erro ao enviar mídia: %w", err)
		}
	}

	// Atualizar último acesso da sessão
	if err := uc.sessionRepo.UpdateLastSeen(ctx, req.SessionID); err != nil {
		uc.logger.Warn().Err(err).Msg("erro ao atualizar último acesso da sessão")
	}

	uc.logger.Info().
		Str("session_id", req.SessionID.String()).
		Str("to_jid", req.ToJID).
		Str("media_type", string(req.Type)).
		Str("whatsapp_id", whatsappResp.MessageID).
		Msg("mídia enviada com sucesso via WhatsApp")

	return &SendMediaResponse{
		WhatsAppID: whatsappResp.MessageID,
		Status:     message.MessageStatusSent,
		Timestamp:  time.Now().Format("2006-01-02T15:04:05Z07:00"),
		Message:    "Mídia enviada com sucesso",
	}, nil
}

// isValidMediaType verifica se o tipo de mídia é válido
func isValidMediaType(msgType message.MessageType) bool {
	validTypes := []message.MessageType{
		message.MessageTypeImage,
		message.MessageTypeAudio,
		message.MessageTypeVideo,
		message.MessageTypeDocument,
		message.MessageTypeSticker,
	}

	for _, validType := range validTypes {
		if msgType == validType {
			return true
		}
	}

	return false
}

// validateMediaURL valida se a URL é válida (apenas HTTP/HTTPS)
func validateMediaURL(mediaURL string) error {
	// Validar formato da URL
	parsedURL, err := url.Parse(mediaURL)
	if err != nil {
		return fmt.Errorf("formato de URL inválido: %w", err)
	}

	// Verificar se é HTTP ou HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL deve usar protocolo HTTP ou HTTPS")
	}

	// Verificar se o host não está vazio
	if parsedURL.Host == "" {
		return fmt.Errorf("host da URL não pode estar vazio")
	}

	return nil
}

// validateFilePath valida se o caminho do arquivo é válido e acessível
func validateFilePath(filePath string) error {
	// Verificar se o caminho não está vazio
	if strings.TrimSpace(filePath) == "" {
		return fmt.Errorf("caminho do arquivo não pode estar vazio")
	}

	// Verificar se o arquivo existe
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("arquivo não encontrado: %s", filePath)
		}
		return fmt.Errorf("erro ao acessar arquivo: %w", err)
	}

	return nil
}

// validateMimeType valida se o tipo MIME é suportado para o tipo de mídia
func validateMimeType(mediaType message.MessageType, mimeType string) error {
	var supportedMimes []string

	switch mediaType {
	case message.MessageTypeImage:
		supportedMimes = SupportedImageMimes
	case message.MessageTypeVideo:
		supportedMimes = SupportedVideoMimes
	case message.MessageTypeAudio:
		supportedMimes = SupportedAudioMimes
	case message.MessageTypeDocument:
		supportedMimes = SupportedDocumentMimes
	case message.MessageTypeSticker:
		supportedMimes = SupportedStickerMimes
	default:
		return fmt.Errorf("tipo de mídia não suportado: %s", mediaType)
	}

	// Verificar se o MIME type está na lista de suportados
	for _, supported := range supportedMimes {
		if strings.EqualFold(mimeType, supported) {
			return nil
		}
	}

	return fmt.Errorf("tipo MIME '%s' não suportado para %s", mimeType, mediaType)
}

// detectMimeTypeFromPath detecta o tipo MIME baseado na extensão do arquivo
func (uc *SendMediaUseCase) detectMimeTypeFromPath(filePath string) string {
	ext := strings.ToLower(filePath)

	// Imagens
	if strings.HasSuffix(ext, ".jpg") || strings.HasSuffix(ext, ".jpeg") {
		return "image/jpeg"
	}
	if strings.HasSuffix(ext, ".png") {
		return "image/png"
	}
	if strings.HasSuffix(ext, ".gif") {
		return "image/gif"
	}
	if strings.HasSuffix(ext, ".webp") {
		return "image/webp"
	}

	// Vídeos
	if strings.HasSuffix(ext, ".mp4") {
		return "video/mp4"
	}
	if strings.HasSuffix(ext, ".avi") {
		return "video/avi"
	}
	if strings.HasSuffix(ext, ".mov") {
		return "video/mov"
	}

	// Áudios
	if strings.HasSuffix(ext, ".mp3") {
		return "audio/mpeg"
	}
	if strings.HasSuffix(ext, ".wav") {
		return "audio/wav"
	}
	if strings.HasSuffix(ext, ".ogg") {
		return "audio/ogg"
	}

	// Documentos
	if strings.HasSuffix(ext, ".pdf") {
		return "application/pdf"
	}
	if strings.HasSuffix(ext, ".doc") {
		return "application/msword"
	}
	if strings.HasSuffix(ext, ".docx") {
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}
	if strings.HasSuffix(ext, ".xls") {
		return "application/vnd.ms-excel"
	}
	if strings.HasSuffix(ext, ".xlsx") {
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	if strings.HasSuffix(ext, ".txt") {
		return "text/plain"
	}
	if strings.HasSuffix(ext, ".zip") {
		return "application/zip"
	}

	// Padrão
	return "application/octet-stream"
}
