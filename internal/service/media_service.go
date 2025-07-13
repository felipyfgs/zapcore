package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"strings"

	"wamex/internal/domain"
	"wamex/pkg/logger"

	"github.com/nfnt/resize"
	"github.com/vincent-petithory/dataurl"
	"go.mau.fi/whatsmeow"
)

// MediaService gerencia operações de mídia
type MediaService struct{}

// NewMediaService cria uma nova instância do MediaService
func NewMediaService() *MediaService {
	return &MediaService{}
}

// DecodeBase64Media decodifica dados base64 de mídia
func (ms *MediaService) DecodeBase64Media(dataURL string) ([]byte, string, error) {
	logger.WithComponent("media").Debug().
		Str("data_url_prefix", dataURL[:min(50, len(dataURL))]).
		Msg("Decodificando mídia base64")

	// Verifica se é um data URL válido
	if !strings.HasPrefix(dataURL, "data:") {
		return nil, "", fmt.Errorf("formato de data URL inválido")
	}

	// Decodifica o data URL
	decoded, err := dataurl.DecodeString(dataURL)
	if err != nil {
		logger.WithComponent("media").Error().
			Err(err).
			Msg("Erro ao decodificar data URL")
		return nil, "", fmt.Errorf("erro ao decodificar base64: %w", err)
	}

	mimeType := decoded.MediaType.ContentType()
	data := decoded.Data

	logger.WithComponent("media").Info().
		Str("mime_type", mimeType).
		Int("data_size", len(data)).
		Msg("Mídia decodificada com sucesso")

	return data, mimeType, nil
}

// ValidateMediaType valida se o tipo MIME é suportado para o tipo de mensagem
func (ms *MediaService) ValidateMediaType(mimeType string, messageType domain.MessageType) error {
	if !domain.IsValidMimeType(messageType, mimeType) {
		logger.WithComponent("media").Warn().
			Str("mime_type", mimeType).
			Str("message_type", string(messageType)).
			Msg("Tipo MIME não suportado")
		return fmt.Errorf("tipo MIME %s não suportado para %s", mimeType, messageType)
	}
	return nil
}

// ValidateFileSize valida se o tamanho do arquivo está dentro do limite
func (ms *MediaService) ValidateFileSize(data []byte, messageType domain.MessageType) error {
	maxSize := domain.GetMaxFileSize(messageType)
	if maxSize == 0 {
		return fmt.Errorf("tipo de mensagem não suportado: %s", messageType)
	}

	if int64(len(data)) > maxSize {
		logger.WithComponent("media").Warn().
			Int("file_size", len(data)).
			Int64("max_size", maxSize).
			Str("message_type", string(messageType)).
			Msg("Arquivo excede tamanho máximo")
		return fmt.Errorf("arquivo muito grande: %d bytes (máximo: %d bytes)", len(data), maxSize)
	}

	return nil
}

// GenerateImageThumbnail gera thumbnail para imagens
func (ms *MediaService) GenerateImageThumbnail(imageData []byte, mimeType string) ([]byte, error) {
	logger.WithComponent("media").Debug().
		Str("mime_type", mimeType).
		Int("original_size", len(imageData)).
		Msg("Gerando thumbnail de imagem")

	// Decodifica a imagem baseado no tipo MIME
	var img image.Image
	var err error

	reader := bytes.NewReader(imageData)

	switch mimeType {
	case domain.MimeTypeImageJPEG:
		img, err = jpeg.Decode(reader)
	case domain.MimeTypeImagePNG:
		img, err = png.Decode(reader)
	default:
		// Tenta detectar automaticamente
		img, _, err = image.Decode(bytes.NewReader(imageData))
	}

	if err != nil {
		logger.WithComponent("media").Error().
			Err(err).
			Str("mime_type", mimeType).
			Msg("Erro ao decodificar imagem para thumbnail")
		return nil, fmt.Errorf("erro ao decodificar imagem: %w", err)
	}

	// Redimensiona para thumbnail (máximo 320x320 mantendo proporção)
	thumbnail := resize.Thumbnail(320, 320, img, resize.Lanczos3)

	// Codifica o thumbnail como JPEG
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, thumbnail, &jpeg.Options{Quality: 80})
	if err != nil {
		logger.WithComponent("media").Error().
			Err(err).
			Msg("Erro ao codificar thumbnail")
		return nil, fmt.Errorf("erro ao codificar thumbnail: %w", err)
	}

	thumbnailData := buf.Bytes()

	logger.WithComponent("media").Info().
		Int("thumbnail_size", len(thumbnailData)).
		Msg("Thumbnail gerado com sucesso")

	return thumbnailData, nil
}

// UploadMediaToWhatsApp faz upload da mídia para o WhatsApp
func (ms *MediaService) UploadMediaToWhatsApp(client *whatsmeow.Client, data []byte, mediaType whatsmeow.MediaType) (*whatsmeow.UploadResponse, error) {
	logger.WithComponent("media").Debug().
		Str("media_type", string(mediaType)).
		Int("data_size", len(data)).
		Msg("Fazendo upload de mídia para WhatsApp")

	// Faz o upload
	uploaded, err := client.Upload(context.Background(), data, mediaType)
	if err != nil {
		logger.WithComponent("media").Error().
			Err(err).
			Str("media_type", string(mediaType)).
			Msg("Erro ao fazer upload para WhatsApp")
		return nil, fmt.Errorf("erro ao fazer upload para WhatsApp: %w", err)
	}

	logger.WithComponent("media").Info().
		Str("media_type", string(mediaType)).
		Str("url", uploaded.URL).
		Str("direct_path", uploaded.DirectPath).
		Msg("Upload para WhatsApp realizado com sucesso")

	return &uploaded, nil
}

// DetectMimeType detecta o tipo MIME de dados binários
func (ms *MediaService) DetectMimeType(data []byte) string {
	mimeType := http.DetectContentType(data)

	logger.WithComponent("media").Debug().
		Str("detected_mime", mimeType).
		Int("data_size", len(data)).
		Msg("Tipo MIME detectado")

	return mimeType
}

// GetWhatsAppMediaType converte tipo MIME para tipo de mídia do WhatsApp
func (ms *MediaService) GetWhatsAppMediaType(mimeType string) whatsmeow.MediaType {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return whatsmeow.MediaImage
	case strings.HasPrefix(mimeType, "audio/"):
		return whatsmeow.MediaAudio
	case strings.HasPrefix(mimeType, "video/"):
		return whatsmeow.MediaVideo
	default:
		return whatsmeow.MediaDocument
	}
}

// GetImageDimensions obtém as dimensões de uma imagem
func (ms *MediaService) GetImageDimensions(imageData []byte) (string, error) {
	reader := bytes.NewReader(imageData)
	config, _, err := image.DecodeConfig(reader)
	if err != nil {
		return "", fmt.Errorf("erro ao obter dimensões da imagem: %w", err)
	}

	dimensions := fmt.Sprintf("%dx%d", config.Width, config.Height)

	logger.WithComponent("media").Debug().
		Str("dimensions", dimensions).
		Msg("Dimensões da imagem obtidas")

	return dimensions, nil
}

// ProcessMediaForUpload processa mídia completa para upload (compatibilidade com base64)
func (ms *MediaService) ProcessMediaForUpload(dataURL string, messageType domain.MessageType) (*ProcessedMedia, error) {
	// Decodifica base64
	data, mimeType, err := ms.DecodeBase64Media(dataURL)
	if err != nil {
		return nil, err
	}

	// Se MIME type não foi fornecido, detecta automaticamente
	if mimeType == "" {
		mimeType = ms.DetectMimeType(data)
	}

	// Valida tipo MIME
	if err := ms.ValidateMediaType(mimeType, messageType); err != nil {
		return nil, err
	}

	// Valida tamanho
	if err := ms.ValidateFileSize(data, messageType); err != nil {
		return nil, err
	}

	processed := &ProcessedMedia{
		Data:     data,
		MimeType: mimeType,
		Size:     int64(len(data)),
	}

	// Gera thumbnail para imagens
	if messageType == domain.MessageTypeImage {
		thumbnail, err := ms.GenerateImageThumbnail(data, mimeType)
		if err != nil {
			logger.WithComponent("media").Warn().
				Err(err).
				Msg("Erro ao gerar thumbnail, continuando sem thumbnail")
		} else {
			processed.Thumbnail = thumbnail
		}

		// Obtém dimensões
		if dimensions, err := ms.GetImageDimensions(data); err == nil {
			processed.Dimensions = dimensions
		}
	}

	return processed, nil
}

// ProcessMediaFromSource processa mídia de qualquer fonte suportada
func (ms *MediaService) ProcessMediaFromSource(sourceResult *MediaSourceResult, messageType domain.MessageType) (*ProcessedMedia, error) {
	logger.WithComponent("media").Debug().
		Str("source", sourceResult.Source).
		Str("mime_type", sourceResult.MimeType).
		Int64("size", sourceResult.Size).
		Str("message_type", string(messageType)).
		Msg("Processando mídia de fonte externa")

	processed := &ProcessedMedia{
		Data:     sourceResult.Data,
		MimeType: sourceResult.MimeType,
		Size:     sourceResult.Size,
		Filename: sourceResult.Filename,
	}

	// Gera thumbnail para imagens
	if messageType == domain.MessageTypeImage {
		thumbnail, err := ms.GenerateImageThumbnail(sourceResult.Data, sourceResult.MimeType)
		if err != nil {
			logger.WithComponent("media").Warn().
				Err(err).
				Msg("Erro ao gerar thumbnail, continuando sem thumbnail")
		} else {
			processed.Thumbnail = thumbnail
		}

		// Obtém dimensões
		if dimensions, err := ms.GetImageDimensions(sourceResult.Data); err == nil {
			processed.Dimensions = dimensions
		}
	}

	return processed, nil
}

// ProcessedMedia representa mídia processada pronta para upload
type ProcessedMedia struct {
	Data       []byte
	MimeType   string
	Size       int64
	Filename   string
	Thumbnail  []byte
	Dimensions string
}

// Função auxiliar para min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
