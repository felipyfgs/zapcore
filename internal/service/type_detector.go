package service

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"wamex/internal/domain"
	"wamex/pkg/logger"
)

// AutoTypeDetector implementa detecção automática de tipos de mídia
type AutoTypeDetector struct{}

// NewAutoTypeDetector cria uma nova instância do detector
func NewAutoTypeDetector() *AutoTypeDetector {
	return &AutoTypeDetector{}
}

// DetectFromData detecta o tipo de mídia a partir dos dados binários e nome do arquivo
func (d *AutoTypeDetector) DetectFromData(data []byte, filename string) (domain.MessageType, string, error) {
	if len(data) == 0 {
		return "", "", fmt.Errorf("dados vazios fornecidos")
	}

	logger.WithComponent("auto-type-detector").Debug().
		Int("data_size", len(data)).
		Str("filename", filename).
		Msg("Iniciando detecção automática de tipo")

	// 1. Detectar MIME type usando magic numbers (primeiros 512 bytes)
	mimeType := http.DetectContentType(data)

	logger.WithComponent("auto-type-detector").Debug().
		Str("detected_mime", mimeType).
		Msg("MIME type detectado via magic numbers")

	// 2. Se não conseguiu detectar ou detectou genérico, tentar pela extensão
	if mimeType == "application/octet-stream" || mimeType == "text/plain" {
		if filename != "" {
			mimeFromExt := d.detectMimeFromExtension(filename)
			if mimeFromExt != "" {
				mimeType = mimeFromExt
				logger.WithComponent("auto-type-detector").Debug().
					Str("mime_from_extension", mimeType).
					Msg("MIME type corrigido pela extensão")
			}
		}
	}

	// 3. Detectar tipo de mensagem baseado no MIME type
	messageType := domain.DetectMessageTypeFromMime(mimeType)

	// 4. Validar compatibilidade com WhatsApp
	if err := d.ValidateForWhatsApp(messageType, mimeType); err != nil {
		return "", "", fmt.Errorf("tipo não suportado pelo WhatsApp: %w", err)
	}

	// 5. Verificar se é sticker (caso especial)
	if d.isSticker(data, mimeType, filename) {
		messageType = domain.MessageTypeSticker
		logger.WithComponent("auto-type-detector").Debug().
			Msg("Detectado como sticker")
	}

	logger.WithComponent("auto-type-detector").Info().
		Str("message_type", string(messageType)).
		Str("mime_type", mimeType).
		Str("filename", filename).
		Msg("Detecção automática concluída")

	return messageType, mimeType, nil
}

// ValidateForWhatsApp valida se o tipo é suportado pelo WhatsApp
func (d *AutoTypeDetector) ValidateForWhatsApp(messageType domain.MessageType, mimeType string) error {
	if !domain.IsValidMimeType(messageType, mimeType) {
		return fmt.Errorf("tipo MIME %s não suportado para %s", mimeType, messageType)
	}
	return nil
}

// detectMimeFromExtension detecta MIME type pela extensão do arquivo
func (d *AutoTypeDetector) detectMimeFromExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	mimeMap := map[string]string{
		// Imagens
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",

		// Áudio
		".mp3": "audio/mpeg",
		".ogg": "audio/ogg",
		".aac": "audio/aac",
		".amr": "audio/amr",
		".wav": "audio/wav",

		// Vídeo
		".mp4": "video/mp4",
		".3gp": "video/3gpp",

		// Documentos
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".txt":  "text/plain",
	}

	if mime, exists := mimeMap[ext]; exists {
		return mime
	}

	return ""
}

// isSticker verifica se a mídia deve ser tratada como sticker
func (d *AutoTypeDetector) isSticker(data []byte, mimeType, filename string) bool {
	// Stickers devem ser WebP
	if mimeType != "image/webp" {
		return false
	}

	// Verificar se o nome sugere sticker
	if strings.Contains(strings.ToLower(filename), "sticker") {
		return true
	}

	// Verificar dimensões típicas de sticker (512x512 ou similar)
	// Por enquanto, assumimos que WebP pequenos são stickers
	if len(data) < 100*1024 { // Menos de 100KB
		return true
	}

	return false
}

// DetectFromBase64 detecta tipo a partir de data URL base64
func (d *AutoTypeDetector) DetectFromBase64(dataURL string, mediaService *MediaService) (domain.MessageType, string, []byte, error) {
	// Validar formato básico
	if !strings.HasPrefix(dataURL, "data:") {
		return "", "", nil, fmt.Errorf("formato de data URL inválido")
	}

	logger.WithComponent("auto-type-detector").Debug().
		Msg("Processando data URL base64")

	// Usar MediaService existente para decodificar
	data, mimeType, err := mediaService.DecodeBase64Media(dataURL)
	if err != nil {
		return "", "", nil, fmt.Errorf("erro ao decodificar base64: %w", err)
	}

	// Se MIME type não foi detectado corretamente, usar magic numbers
	if mimeType == "application/octet-stream" || mimeType == "" {
		detectedMime := http.DetectContentType(data)
		if detectedMime != "application/octet-stream" {
			mimeType = detectedMime
			logger.WithComponent("auto-type-detector").Debug().
				Str("original_mime", mimeType).
				Str("detected_mime", detectedMime).
				Msg("MIME type corrigido via magic numbers")
		}
	}

	// Detectar tipo de mensagem
	messageType := domain.DetectMessageTypeFromMime(mimeType)

	// Verificar se é sticker
	if d.isSticker(data, mimeType, "") {
		messageType = domain.MessageTypeSticker
	}

	// Validar compatibilidade com WhatsApp
	if err := d.ValidateForWhatsApp(messageType, mimeType); err != nil {
		return "", "", nil, fmt.Errorf("tipo não suportado pelo WhatsApp: %w", err)
	}

	logger.WithComponent("auto-type-detector").Info().
		Str("message_type", string(messageType)).
		Str("mime_type", mimeType).
		Int("data_size", len(data)).
		Msg("Base64 processado com sucesso")

	return messageType, mimeType, data, nil
}

// DetectFromURL detecta tipo a partir de URL (baseado na extensão)
func (d *AutoTypeDetector) DetectFromURL(url string) (domain.MessageType, string, error) {
	// Extrair nome do arquivo da URL
	filename := filepath.Base(url)

	// Detectar MIME pela extensão
	mimeType := d.detectMimeFromExtension(filename)
	if mimeType == "" {
		// Default para documento se não conseguir detectar
		mimeType = "application/octet-stream"
	}

	// Detectar tipo de mensagem
	messageType := domain.DetectMessageTypeFromMime(mimeType)

	// Validar
	if err := d.ValidateForWhatsApp(messageType, mimeType); err != nil {
		return "", "", fmt.Errorf("tipo não suportado: %w", err)
	}

	logger.WithComponent("auto-type-detector").Info().
		Str("url", url).
		Str("filename", filename).
		Str("message_type", string(messageType)).
		Str("mime_type", mimeType).
		Msg("Tipo detectado a partir de URL")

	return messageType, mimeType, nil
}
