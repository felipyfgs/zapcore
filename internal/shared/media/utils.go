package media

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DecodeBase64Media decodifica dados de mídia em base64
func DecodeBase64Media(base64Data string) ([]byte, string, error) {
	// Verificar se tem o prefixo data:
	if !strings.HasPrefix(base64Data, "data:") {
		return nil, "", fmt.Errorf("base64 deve começar com 'data:'")
	}

	// Extrair mime type e dados
	parts := strings.SplitN(base64Data, ",", 2)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("formato base64 inválido")
	}

	// Extrair mime type do header
	header := parts[0]
	mimeType := ""
	if strings.Contains(header, ";") {
		mimeType = strings.Split(strings.TrimPrefix(header, "data:"), ";")[0]
	}

	// Decodificar base64
	data, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, "", fmt.Errorf("erro ao decodificar base64: %w", err)
	}

	return data, mimeType, nil
}

// DetectMimeType detecta o tipo MIME de dados binários
func DetectMimeType(data []byte) string {
	// Usar http.DetectContentType para detectar o tipo MIME
	return http.DetectContentType(data)
}

// DetectMimeTypeFromReader detecta o tipo MIME de um Reader
func DetectMimeTypeFromReader(reader io.Reader) (string, io.Reader, error) {
	// Ler os primeiros 512 bytes para detectar o tipo
	buffer := make([]byte, 512)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return "", nil, fmt.Errorf("erro ao ler dados para detectar MIME: %w", err)
	}

	// Detectar tipo MIME
	mimeType := http.DetectContentType(buffer[:n])

	// Criar um novo reader que inclui os bytes já lidos
	newReader := io.MultiReader(bytes.NewReader(buffer[:n]), reader)

	return mimeType, newReader, nil
}

// ValidateImageFormat valida se os dados são de uma imagem válida
func ValidateImageFormat(data []byte) error {
	mimeType := DetectMimeType(data)
	
	validImageTypes := []string{
		"image/jpeg",
		"image/png", 
		"image/gif",
		"image/webp",
	}

	for _, validType := range validImageTypes {
		if mimeType == validType {
			return nil
		}
	}

	return fmt.Errorf("formato de imagem inválido: %s", mimeType)
}

// ValidateVideoFormat valida se os dados são de um vídeo válido
func ValidateVideoFormat(data []byte) error {
	mimeType := DetectMimeType(data)
	
	validVideoTypes := []string{
		"video/mp4",
		"video/avi",
		"video/quicktime", // .mov
		"video/x-msvideo", // .avi
		"video/webm",
	}

	for _, validType := range validVideoTypes {
		if mimeType == validType {
			return nil
		}
	}

	return fmt.Errorf("formato de vídeo inválido: %s", mimeType)
}

// ValidateAudioFormat valida se os dados são de um áudio válido
func ValidateAudioFormat(data []byte) error {
	mimeType := DetectMimeType(data)
	
	validAudioTypes := []string{
		"audio/mpeg",
		"audio/wav",
		"audio/ogg",
		"audio/aac",
		"audio/mp4", // m4a
	}

	for _, validType := range validAudioTypes {
		if mimeType == validType {
			return nil
		}
	}

	return fmt.Errorf("formato de áudio inválido: %s", mimeType)
}

// GenerateFileName gera um nome de arquivo baseado no tipo MIME
func GenerateFileName(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		ext := strings.TrimPrefix(mimeType, "image/")
		if ext == "jpeg" {
			ext = "jpg"
		}
		return fmt.Sprintf("image.%s", ext)
	case strings.HasPrefix(mimeType, "video/"):
		ext := strings.TrimPrefix(mimeType, "video/")
		if ext == "quicktime" {
			ext = "mov"
		}
		return fmt.Sprintf("video.%s", ext)
	case strings.HasPrefix(mimeType, "audio/"):
		ext := strings.TrimPrefix(mimeType, "audio/")
		if ext == "mpeg" {
			ext = "mp3"
		}
		return fmt.Sprintf("audio.%s", ext)
	case strings.HasPrefix(mimeType, "application/"):
		ext := strings.TrimPrefix(mimeType, "application/")
		return fmt.Sprintf("document.%s", ext)
	default:
		return "file.bin"
	}
}

// GetFileSizeFromReader obtém o tamanho de um Reader sem consumi-lo
func GetFileSizeFromReader(reader io.Reader) (int64, io.Reader, error) {
	// Ler todos os dados para calcular o tamanho
	data, err := io.ReadAll(reader)
	if err != nil {
		return 0, nil, fmt.Errorf("erro ao ler dados: %w", err)
	}

	size := int64(len(data))
	
	// Criar um novo reader com os dados
	newReader := bytes.NewReader(data)

	return size, newReader, nil
}

// ValidateFileSize valida se o tamanho do arquivo está dentro dos limites
func ValidateFileSize(size int64, mediaType string) error {
	var maxSize int64

	switch mediaType {
	case "image":
		maxSize = 16 * 1024 * 1024 // 16MB
	case "video":
		maxSize = 64 * 1024 * 1024 // 64MB
	case "audio":
		maxSize = 16 * 1024 * 1024 // 16MB
	case "document":
		maxSize = 100 * 1024 * 1024 // 100MB
	case "sticker":
		maxSize = 500 * 1024 // 500KB
	default:
		return fmt.Errorf("tipo de mídia não suportado: %s", mediaType)
	}

	if size > maxSize {
		return fmt.Errorf("arquivo muito grande: %d bytes (máximo: %d bytes)", size, maxSize)
	}

	if size == 0 {
		return fmt.Errorf("arquivo vazio")
	}

	return nil
}

// CreateThumbnail cria um thumbnail para imagens (implementação básica)
// TODO: Implementar geração real de thumbnails usando uma biblioteca de imagem
func CreateThumbnail(imageData []byte) ([]byte, error) {
	// Por enquanto, retorna a imagem original
	// Em uma implementação real, usaríamos uma biblioteca como imaging ou resize
	return imageData, nil
}

// IsValidURL verifica se uma string é uma URL válida
func IsValidURL(urlStr string) bool {
	return strings.HasPrefix(urlStr, "http://") || strings.HasPrefix(urlStr, "https://")
}

// SanitizeFileName remove caracteres inválidos de nomes de arquivo
func SanitizeFileName(fileName string) string {
	// Remover caracteres perigosos
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	
	sanitized := fileName
	for _, char := range invalidChars {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}
	
	// Limitar tamanho
	if len(sanitized) > 255 {
		sanitized = sanitized[:255]
	}
	
	return sanitized
}
