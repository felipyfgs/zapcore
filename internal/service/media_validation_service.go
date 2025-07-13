package service

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"wamex/internal/domain"
)

// MediaValidationService implementa validações robustas para arquivos de mídia
type MediaValidationService struct{}

// NewMediaValidationService cria uma nova instância do serviço de validação
func NewMediaValidationService() *MediaValidationService {
	return &MediaValidationService{}
}

// ValidateFile valida um arquivo multipart completo
func (s *MediaValidationService) ValidateFile(file multipart.File, header *multipart.FileHeader) (*domain.MediaFile, error) {
	// 1. Validar tamanho do arquivo
	if header.Size <= 0 {
		return nil, fmt.Errorf("arquivo vazio")
	}

	// 2. Ler primeiros 512 bytes para detectar tipo MIME real
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("erro ao ler arquivo: %w", err)
	}

	// Reset file pointer para o início
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("erro ao resetar ponteiro do arquivo: %w", err)
	}

	// 3. Detectar tipo MIME real usando magic numbers
	detectedMimeType := http.DetectContentType(buffer[:n])

	// 4. Detectar tipo de mensagem baseado no MIME type
	messageType := domain.DetectMessageTypeFromMime(detectedMimeType)

	// 5. Validar se o tipo MIME é suportado pelo WhatsApp para este tipo de mensagem
	if !domain.IsValidMimeType(messageType, detectedMimeType) {
		return nil, fmt.Errorf("tipo de arquivo não suportado: %s para %s", detectedMimeType, messageType)
	}

	// 6. Validar tamanho máximo para o tipo de mensagem
	maxSize := domain.GetMaxSizeForMessageType(messageType)
	if header.Size > maxSize {
		return nil, fmt.Errorf("arquivo muito grande: %d bytes (máximo: %d bytes para %s)",
			header.Size, maxSize, messageType)
	}

	// 7. Validações específicas por tipo
	if err := s.validateSpecificType(buffer[:n], detectedMimeType, messageType, header.Size); err != nil {
		return nil, err
	}

	// 8. Criar objeto MediaFile
	mediaFile := &domain.MediaFile{
		Filename:    header.Filename,
		MimeType:    detectedMimeType,
		Size:        header.Size,
		MessageType: string(messageType),
	}

	return mediaFile, nil
}

// validateSpecificType executa validações específicas por tipo de mídia
func (s *MediaValidationService) validateSpecificType(buffer []byte, mimeType string, messageType domain.MessageType, size int64) error {
	switch messageType {
	case domain.MessageTypeImage:
		return s.validateImage(buffer, mimeType, size)
	case domain.MessageTypeAudio:
		return s.validateAudio(buffer, mimeType, size)
	case domain.MessageTypeVideo:
		return s.validateVideo(buffer, mimeType, size)
	case domain.MessageTypeDocument:
		return s.validateDocument(buffer, mimeType, size)
	case domain.MessageTypeSticker:
		return s.validateSticker(buffer, mimeType, size)
	default:
		return fmt.Errorf("tipo de mensagem não suportado: %s", messageType)
	}
}

// validateImage valida arquivos de imagem
func (s *MediaValidationService) validateImage(buffer []byte, mimeType string, size int64) error {
	switch mimeType {
	case "image/jpeg":
		// Verificar magic numbers JPEG (FF D8 FF)
		if len(buffer) < 3 || buffer[0] != 0xFF || buffer[1] != 0xD8 || buffer[2] != 0xFF {
			return fmt.Errorf("arquivo JPEG inválido: magic numbers incorretos")
		}
	case "image/png":
		// Verificar magic numbers PNG (89 50 4E 47 0D 0A 1A 0A)
		pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		if len(buffer) < 8 || !bytes.Equal(buffer[:8], pngSignature) {
			return fmt.Errorf("arquivo PNG inválido: magic numbers incorretos")
		}
	}

	// Validar tamanho máximo para imagens (5MB)
	if size > domain.MaxImageSize {
		return fmt.Errorf("imagem muito grande: %d bytes (máximo: %d bytes)", size, domain.MaxImageSize)
	}

	return nil
}

// validateAudio valida arquivos de áudio
func (s *MediaValidationService) validateAudio(buffer []byte, mimeType string, size int64) error {
	switch mimeType {
	case "audio/mpeg":
		// Verificar magic numbers MP3 (ID3 ou FF FB/FF F3/FF F2)
		if len(buffer) < 3 {
			return fmt.Errorf("arquivo MP3 muito pequeno")
		}

		// ID3 tag
		if bytes.Equal(buffer[:3], []byte("ID3")) {
			return nil
		}

		// MPEG frame sync
		if buffer[0] == 0xFF && (buffer[1]&0xE0) == 0xE0 {
			return nil
		}

		return fmt.Errorf("arquivo MP3 inválido: magic numbers incorretos")

	case "audio/ogg":
		// Verificar magic numbers OGG (4F 67 67 53)
		oggSignature := []byte{0x4F, 0x67, 0x67, 0x53}
		if len(buffer) < 4 || !bytes.Equal(buffer[:4], oggSignature) {
			return fmt.Errorf("arquivo OGG inválido: magic numbers incorretos")
		}
	}

	// Validar tamanho máximo para áudio (16MB)
	if size > domain.MaxAudioSize {
		return fmt.Errorf("áudio muito grande: %d bytes (máximo: %d bytes)", size, domain.MaxAudioSize)
	}

	return nil
}

// validateVideo valida arquivos de vídeo
func (s *MediaValidationService) validateVideo(buffer []byte, mimeType string, size int64) error {
	switch mimeType {
	case "video/mp4":
		// Verificar magic numbers MP4 (procurar por "ftyp" nos primeiros bytes)
		if len(buffer) < 8 {
			return fmt.Errorf("arquivo MP4 muito pequeno")
		}

		// Procurar por "ftyp" que indica container MP4
		if !bytes.Contains(buffer[:32], []byte("ftyp")) {
			return fmt.Errorf("arquivo MP4 inválido: assinatura ftyp não encontrada")
		}

	case "video/3gp":
		// Verificar magic numbers 3GP (similar ao MP4, mas com "ftyp3g")
		if len(buffer) < 8 {
			return fmt.Errorf("arquivo 3GP muito pequeno")
		}

		if !bytes.Contains(buffer[:32], []byte("ftyp")) {
			return fmt.Errorf("arquivo 3GP inválido: assinatura ftyp não encontrada")
		}
	}

	// Validar tamanho máximo para vídeo (100MB - mesmo limite de documentos)
	if size > domain.MaxDocumentSize {
		return fmt.Errorf("vídeo muito grande: %d bytes (máximo: %d bytes)", size, domain.MaxDocumentSize)
	}

	return nil
}

// validateDocument valida arquivos de documento
func (s *MediaValidationService) validateDocument(buffer []byte, mimeType string, size int64) error {
	switch mimeType {
	case "application/pdf":
		// Verificar magic numbers PDF (%PDF)
		if len(buffer) < 4 || !bytes.Equal(buffer[:4], []byte("%PDF")) {
			return fmt.Errorf("arquivo PDF inválido: magic numbers incorretos")
		}

	case "text/plain":
		// Para texto simples, verificar se contém apenas caracteres válidos
		// Permitir UTF-8 válido
		if !s.isValidUTF8Text(buffer) {
			return fmt.Errorf("arquivo de texto contém caracteres inválidos")
		}

	case "application/msword":
		// Verificar magic numbers DOC (D0 CF 11 E0 A1 B1 1A E1)
		docSignature := []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
		if len(buffer) < 8 || !bytes.Equal(buffer[:8], docSignature) {
			return fmt.Errorf("arquivo DOC inválido: magic numbers incorretos")
		}

	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		// DOCX é um arquivo ZIP, verificar magic numbers ZIP (50 4B 03 04)
		zipSignature := []byte{0x50, 0x4B, 0x03, 0x04}
		if len(buffer) < 4 || !bytes.Equal(buffer[:4], zipSignature) {
			return fmt.Errorf("arquivo DOCX inválido: magic numbers incorretos")
		}
	}

	// Validar tamanho máximo para documentos (100MB)
	if size > domain.MaxDocumentSize {
		return fmt.Errorf("documento muito grande: %d bytes (máximo: %d bytes)", size, domain.MaxDocumentSize)
	}

	return nil
}

// validateSticker valida arquivos de sticker
func (s *MediaValidationService) validateSticker(buffer []byte, mimeType string, size int64) error {
	if mimeType != "image/webp" {
		return fmt.Errorf("sticker deve ser do tipo WebP")
	}

	// Verificar magic numbers WebP (52 49 46 46 ... 57 45 42 50)
	if len(buffer) < 12 {
		return fmt.Errorf("arquivo WebP muito pequeno")
	}

	// RIFF signature
	if !bytes.Equal(buffer[:4], []byte("RIFF")) {
		return fmt.Errorf("arquivo WebP inválido: assinatura RIFF incorreta")
	}

	// WEBP signature
	if !bytes.Equal(buffer[8:12], []byte("WEBP")) {
		return fmt.Errorf("arquivo WebP inválido: assinatura WEBP incorreta")
	}

	// Validar tamanho máximo para stickers (500KB)
	if size > domain.MaxStickerSize {
		return fmt.Errorf("sticker muito grande: %d bytes (máximo: %d bytes)", size, domain.MaxStickerSize)
	}

	return nil
}

// isValidUTF8Text verifica se o buffer contém texto UTF-8 válido
func (s *MediaValidationService) isValidUTF8Text(buffer []byte) bool {
	// Converter para string e verificar se é UTF-8 válido
	text := string(buffer)

	// Verificar se contém apenas caracteres imprimíveis e espaços em branco
	for _, r := range text {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}

	return true
}

// GetFileExtension retorna a extensão apropriada para um tipo MIME
func (s *MediaValidationService) GetFileExtension(mimeType string) string {
	extensions := map[string]string{
		"image/jpeg":         ".jpg",
		"image/png":          ".png",
		"image/gif":          ".gif",
		"image/webp":         ".webp",
		"audio/mpeg":         ".mp3",
		"audio/ogg":          ".ogg",
		"audio/wav":          ".wav",
		"audio/aac":          ".aac",
		"audio/mp4":          ".m4a",
		"audio/amr":          ".amr",
		"video/mp4":          ".mp4",
		"video/3gp":          ".3gp",
		"application/pdf":    ".pdf",
		"text/plain":         ".txt",
		"application/msword": ".doc",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
		"application/vnd.ms-excel": ".xls",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         ".xlsx",
		"application/vnd.ms-powerpoint":                                             ".ppt",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": ".pptx",
	}

	if ext, exists := extensions[mimeType]; exists {
		return ext
	}

	return ".bin"
}

// SanitizeFilename remove caracteres perigosos do nome do arquivo
func (s *MediaValidationService) SanitizeFilename(filename string) string {
	// Remove caracteres perigosos
	dangerous := []string{"..", "/", "\\", ":", "*", "?", "\"", "<", ">", "|"}

	sanitized := filename
	for _, char := range dangerous {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}

	// Limita o tamanho do nome do arquivo
	if len(sanitized) > 255 {
		sanitized = sanitized[:255]
	}

	return sanitized
}
