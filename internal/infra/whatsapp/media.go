package whatsapp

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"

	"zapcore/internal/infra/storage"
	"zapcore/pkg/logger"
)

// Constantes para tipos de mídia
const (
	MediaTypeImage    = "image"
	MediaTypeVideo    = "video"
	MediaTypeAudio    = "audio"
	MediaTypeDocument = "document"
	MediaTypeSticker  = "sticker"
)

// Constantes para direções de mensagem
const (
	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"
)

// Constantes para padronização de logging
const (
	LogComponentMedia = "media"
	LogFieldDuration  = "duration_ms"
	LogFieldSize      = "size_bytes"
	LogFieldMimeType  = "mime_type"
)

// MediaDownloader gerencia o download de mídias do WhatsApp e upload para MinIO
type MediaDownloader struct {
	client      *whatsmeow.Client
	minioClient *storage.MinIOClient
	logger      *logger.Logger
}

// NewMediaDownloader cria uma nova instância do MediaDownloader
func NewMediaDownloader(client *whatsmeow.Client, minioClient *storage.MinIOClient) *MediaDownloader {
	return &MediaDownloader{
		client:      client,
		minioClient: minioClient,
		logger:      logger.Get().WithField("component", "media_downloader"),
	}
}

// MediaInfo contém informações sobre a mídia baixada e processada
type MediaInfo struct {
	Data       []byte `json:"-"`           // Dados binários da mídia (não serializado)
	MimeType   string `json:"mime_type"`   // Tipo MIME da mídia
	Extension  string `json:"extension"`   // Extensão do arquivo
	Size       int64  `json:"size"`        // Tamanho em bytes
	FileName   string `json:"file_name"`   // Nome do arquivo
	ObjectPath string `json:"object_path"` // Caminho no MinIO
}

// DownloadAndUploadMedia baixa mídia do WhatsApp e faz upload para MinIO
func (md *MediaDownloader) DownloadAndUploadMedia(ctx context.Context, evt *events.Message, sessionID uuid.UUID) (*MediaInfo, error) {
	// Validações de entrada
	if evt == nil || evt.Message == nil {
		return nil, fmt.Errorf("evento ou mensagem é nil")
	}
	if sessionID == uuid.Nil {
		return nil, fmt.Errorf("sessionID inválido")
	}
	if md.client == nil {
		return nil, fmt.Errorf("cliente WhatsApp não configurado")
	}
	if md.minioClient == nil {
		return nil, fmt.Errorf("cliente MinIO não configurado")
	}

	startTime := time.Now()

	md.logger.Info().
		Str("session_id", sessionID.String()).
		Str("message_id", evt.Info.ID).
		Str("chat_jid", evt.Info.Chat.String()).
		Bool("is_from_me", evt.Info.IsFromMe).
		Str("component", LogComponentMedia).
		Msg("Iniciando processamento de mídia")

	var mediaInfo *MediaInfo
	var err error
	var mediaType string

	// Determinar direção da mensagem
	direction := DirectionInbound
	if evt.Info.IsFromMe {
		direction = DirectionOutbound
	}

	// Processar diferentes tipos de mídia com logging específico
	switch {
	case evt.Message.ImageMessage != nil:
		mediaType = MediaTypeImage
		md.logger.Debug().
			Str("session_id", sessionID.String()).
			Str("message_id", evt.Info.ID).
			Str("mime_type", evt.Message.ImageMessage.GetMimetype()).
			Msg("📸 Processando mensagem de imagem")
		mediaInfo, err = md.processImageMessage(ctx, evt.Message.ImageMessage, evt.Info.ID, sessionID, evt.Info.Chat.String(), direction)

	case evt.Message.VideoMessage != nil:
		mediaType = MediaTypeVideo
		md.logger.Debug().
			Str("session_id", sessionID.String()).
			Str("message_id", evt.Info.ID).
			Str("mime_type", evt.Message.VideoMessage.GetMimetype()).
			Msg("🎥 Processando mensagem de vídeo")
		mediaInfo, err = md.processVideoMessage(ctx, evt.Message.VideoMessage, evt.Info.ID, sessionID, evt.Info.Chat.String(), direction)

	case evt.Message.AudioMessage != nil:
		mediaType = MediaTypeAudio
		md.logger.Debug().
			Str("session_id", sessionID.String()).
			Str("message_id", evt.Info.ID).
			Str("mime_type", evt.Message.AudioMessage.GetMimetype()).
			Bool("is_ptt", evt.Message.AudioMessage.GetPTT()).
			Msg("🎵 Processando mensagem de áudio")
		mediaInfo, err = md.processAudioMessage(ctx, evt.Message.AudioMessage, evt.Info.ID, sessionID, evt.Info.Chat.String(), direction)

	case evt.Message.DocumentMessage != nil:
		mediaType = MediaTypeDocument
		fileName := "documento"
		if evt.Message.DocumentMessage.FileName != nil {
			fileName = *evt.Message.DocumentMessage.FileName
		}
		md.logger.Debug().
			Str("session_id", sessionID.String()).
			Str("message_id", evt.Info.ID).
			Str("mime_type", evt.Message.DocumentMessage.GetMimetype()).
			Str("file_name", fileName).
			Msg("📄 Processando mensagem de documento")
		mediaInfo, err = md.processDocumentMessage(ctx, evt.Message.DocumentMessage, evt.Info.ID, sessionID, evt.Info.Chat.String(), direction)

	case evt.Message.StickerMessage != nil:
		mediaType = MediaTypeSticker
		md.logger.Debug().
			Str("session_id", sessionID.String()).
			Str("message_id", evt.Info.ID).
			Str("mime_type", evt.Message.StickerMessage.GetMimetype()).
			Msg("🎭 Processando mensagem de sticker")
		mediaInfo, err = md.processStickerMessage(ctx, evt.Message.StickerMessage, evt.Info.ID, sessionID, evt.Info.Chat.String(), direction)

	default:
		md.logger.Error().
			Str("session_id", sessionID.String()).
			Str("message_id", evt.Info.ID).
			Msg("❌ Tipo de mídia não suportado ou não identificado")
		return nil, fmt.Errorf("tipo de mídia não suportado")
	}

	processingDuration := time.Since(startTime)

	if err != nil {
		md.logger.Error().
			Err(err).
			Str("session_id", sessionID.String()).
			Str("message_id", evt.Info.ID).
			Str("media_type", mediaType).
			Dur("processing_duration", processingDuration).
			Msg("❌ Erro ao processar mídia")
		return nil, fmt.Errorf("erro ao processar mídia %s: %w", mediaType, err)
	}

	md.logger.Info().
		Str("session_id", sessionID.String()).
		Str("message_id", evt.Info.ID).
		Str("media_type", mediaType).
		Str("object_path", mediaInfo.ObjectPath).
		Int64("size", mediaInfo.Size).
		Str("mime_type", mediaInfo.MimeType).
		Str("direction", direction).
		Dur("processing_duration", processingDuration).
		Msg("✅ Mídia processada e armazenada com sucesso")

	return mediaInfo, nil
}

// processImageMessage processa mensagens de imagem
func (md *MediaDownloader) processImageMessage(ctx context.Context, img *waE2E.ImageMessage, messageID string, sessionID uuid.UUID, chatJID, direction string) (*MediaInfo, error) {
	downloadStart := time.Now()

	md.logger.Debug().
		Str("session_id", sessionID.String()).
		Str("message_id", messageID).
		Str("mime_type", img.GetMimetype()).
		Msg("📥 Iniciando download da imagem")

	// Download da imagem
	data, err := md.client.Download(ctx, img)
	if err != nil {
		md.logger.Error().
			Err(err).
			Str("session_id", sessionID.String()).
			Str("message_id", messageID).
			Msg("❌ Erro ao fazer download da imagem")
		return nil, fmt.Errorf("erro ao baixar imagem: %w", err)
	}

	downloadDuration := time.Since(downloadStart)

	md.logger.Debug().
		Str("session_id", sessionID.String()).
		Str("message_id", messageID).
		Int("size_bytes", len(data)).
		Dur("download_duration", downloadDuration).
		Msg("✅ Download da imagem concluído")

	// Determinar extensão
	mimeType := img.GetMimetype()
	extension := md.getExtensionFromMimeType(mimeType, ".jpg")

	uploadStart := time.Now()

	md.logger.Debug().
		Str("session_id", sessionID.String()).
		Str("message_id", messageID).
		Str("extension", extension).
		Msg("📤 Iniciando upload para MinIO")

	// Upload para MinIO
	objectPath, err := md.uploadToMinIO(ctx, data, storage.MediaUploadOptions{
		SessionID:   sessionID,
		ChatJID:     chatJID,
		Direction:   direction,
		MessageID:   messageID,
		ContentType: mimeType,
		Extension:   extension,
		Size:        int64(len(data)),
	})
	if err != nil {
		md.logger.Error().
			Err(err).
			Str("session_id", sessionID.String()).
			Str("message_id", messageID).
			Msg("❌ Erro ao fazer upload da imagem para MinIO")
		return nil, fmt.Errorf("erro ao fazer upload da imagem: %w", err)
	}

	uploadDuration := time.Since(uploadStart)

	md.logger.Debug().
		Str("session_id", sessionID.String()).
		Str("message_id", messageID).
		Str("object_path", objectPath).
		Dur("upload_duration", uploadDuration).
		Msg("✅ Upload da imagem para MinIO concluído")

	return &MediaInfo{
		Data:       data,
		MimeType:   mimeType,
		Extension:  extension,
		Size:       int64(len(data)),
		FileName:   fmt.Sprintf("%s.%s", messageID, extension),
		ObjectPath: objectPath,
	}, nil
}

// processVideoMessage processa mensagens de vídeo
func (md *MediaDownloader) processVideoMessage(ctx context.Context, video *waE2E.VideoMessage, messageID string, sessionID uuid.UUID, chatJID, direction string) (*MediaInfo, error) {
	// Download do vídeo
	data, err := md.client.Download(ctx, video)
	if err != nil {
		return nil, fmt.Errorf("erro ao baixar vídeo: %w", err)
	}

	// Determinar extensão
	mimeType := video.GetMimetype()
	extension := md.getExtensionFromMimeType(mimeType, ".mp4")

	// Upload para MinIO
	objectPath, err := md.uploadToMinIO(ctx, data, storage.MediaUploadOptions{
		SessionID:   sessionID,
		ChatJID:     chatJID,
		Direction:   direction,
		MessageID:   messageID,
		ContentType: mimeType,
		Extension:   extension,
		Size:        int64(len(data)),
	})
	if err != nil {
		return nil, err
	}

	return &MediaInfo{
		Data:       data,
		MimeType:   mimeType,
		Extension:  extension,
		Size:       int64(len(data)),
		FileName:   fmt.Sprintf("%s.%s", messageID, extension),
		ObjectPath: objectPath,
	}, nil
}

// processAudioMessage processa mensagens de áudio
func (md *MediaDownloader) processAudioMessage(ctx context.Context, audio *waE2E.AudioMessage, messageID string, sessionID uuid.UUID, chatJID, direction string) (*MediaInfo, error) {
	// Download do áudio
	data, err := md.client.Download(ctx, audio)
	if err != nil {
		return nil, fmt.Errorf("erro ao baixar áudio: %w", err)
	}

	// Determinar extensão
	mimeType := audio.GetMimetype()
	extension := md.getExtensionFromMimeType(mimeType, ".ogg")

	// Upload para MinIO
	objectPath, err := md.uploadToMinIO(ctx, data, storage.MediaUploadOptions{
		SessionID:   sessionID,
		ChatJID:     chatJID,
		Direction:   direction,
		MessageID:   messageID,
		ContentType: mimeType,
		Extension:   extension,
		Size:        int64(len(data)),
	})
	if err != nil {
		return nil, err
	}

	return &MediaInfo{
		Data:       data,
		MimeType:   mimeType,
		Extension:  extension,
		Size:       int64(len(data)),
		FileName:   fmt.Sprintf("%s.%s", messageID, extension),
		ObjectPath: objectPath,
	}, nil
}

// processDocumentMessage processa mensagens de documento
func (md *MediaDownloader) processDocumentMessage(ctx context.Context, doc *waE2E.DocumentMessage, messageID string, sessionID uuid.UUID, chatJID, direction string) (*MediaInfo, error) {
	// Download do documento
	data, err := md.client.Download(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("erro ao baixar documento: %w", err)
	}

	// Determinar extensão
	mimeType := doc.GetMimetype()
	extension := md.getExtensionFromMimeType(mimeType, ".bin")

	// Se o documento tem nome de arquivo, usar sua extensão
	if doc.FileName != nil && *doc.FileName != "" {
		if ext := filepath.Ext(*doc.FileName); ext != "" {
			extension = strings.TrimPrefix(ext, ".")
		}
	}

	// Upload para MinIO
	objectPath, err := md.uploadToMinIO(ctx, data, storage.MediaUploadOptions{
		SessionID:   sessionID,
		ChatJID:     chatJID,
		Direction:   direction,
		MessageID:   messageID,
		ContentType: mimeType,
		Extension:   extension,
		Size:        int64(len(data)),
	})
	if err != nil {
		return nil, err
	}

	fileName := fmt.Sprintf("%s.%s", messageID, extension)
	if doc.FileName != nil && *doc.FileName != "" {
		fileName = *doc.FileName
	}

	return &MediaInfo{
		Data:       data,
		MimeType:   mimeType,
		Extension:  extension,
		Size:       int64(len(data)),
		FileName:   fileName,
		ObjectPath: objectPath,
	}, nil
}

// processStickerMessage processa mensagens de sticker
func (md *MediaDownloader) processStickerMessage(ctx context.Context, sticker *waE2E.StickerMessage, messageID string, sessionID uuid.UUID, chatJID, direction string) (*MediaInfo, error) {
	// Download do sticker
	data, err := md.client.Download(ctx, sticker)
	if err != nil {
		return nil, fmt.Errorf("erro ao baixar sticker: %w", err)
	}

	// Determinar extensão
	mimeType := sticker.GetMimetype()
	extension := md.getExtensionFromMimeType(mimeType, ".webp")

	// Upload para MinIO
	objectPath, err := md.uploadToMinIO(ctx, data, storage.MediaUploadOptions{
		SessionID:   sessionID,
		ChatJID:     chatJID,
		Direction:   direction,
		MessageID:   messageID,
		ContentType: mimeType,
		Extension:   extension,
		Size:        int64(len(data)),
	})
	if err != nil {
		return nil, err
	}

	return &MediaInfo{
		Data:       data,
		MimeType:   mimeType,
		Extension:  extension,
		Size:       int64(len(data)),
		FileName:   fmt.Sprintf("%s.%s", messageID, extension),
		ObjectPath: objectPath,
	}, nil
}

// uploadToMinIO faz upload dos dados para o MinIO
func (md *MediaDownloader) uploadToMinIO(ctx context.Context, data []byte, opts storage.MediaUploadOptions) (string, error) {
	reader := bytes.NewReader(data)
	return md.minioClient.UploadMedia(ctx, reader, opts)
}

// getExtensionFromMimeType obtém a extensão do arquivo baseada no MIME type
func (md *MediaDownloader) getExtensionFromMimeType(mimeType, defaultExt string) string {
	if mimeType == "" {
		return strings.TrimPrefix(defaultExt, ".")
	}

	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil || len(exts) == 0 {
		return strings.TrimPrefix(defaultExt, ".")
	}

	// Retornar a primeira extensão sem o ponto
	return strings.TrimPrefix(exts[0], ".")
}

// HasMedia verifica se a mensagem contém mídia
func HasMedia(msg *waE2E.Message) bool {
	return msg.ImageMessage != nil ||
		msg.VideoMessage != nil ||
		msg.AudioMessage != nil ||
		msg.DocumentMessage != nil ||
		msg.StickerMessage != nil
}
