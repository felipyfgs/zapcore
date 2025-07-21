package media

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MediaProcessor processa diferentes tipos de mídia
type MediaProcessor struct {
	client *http.Client
}

// NewMediaProcessor cria um novo processador de mídia
func NewMediaProcessor() *MediaProcessor {
	return &MediaProcessor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ProcessBase64Media processa mídia em formato base64
func (p *MediaProcessor) ProcessBase64Media(base64Data string) (*ProcessedMedia, error) {
	data, mimeType, err := DecodeBase64Media(base64Data)
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar base64: %w", err)
	}

	return &ProcessedMedia{
		Data:     data,
		MimeType: mimeType,
		Size:     int64(len(data)),
		FileName: GenerateFileName(mimeType),
		Reader:   bytes.NewReader(data),
	}, nil
}

// ProcessURLMedia processa mídia de uma URL
func (p *MediaProcessor) ProcessURLMedia(url string) (*ProcessedMedia, error) {
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erro ao baixar mídia da URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erro HTTP ao baixar mídia: %d", resp.StatusCode)
	}

	// Ler dados
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler dados da URL: %w", err)
	}

	// Detectar tipo MIME
	mimeType := DetectMimeType(data)
	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		mimeType = contentType
	}

	return &ProcessedMedia{
		Data:     data,
		MimeType: mimeType,
		Size:     int64(len(data)),
		FileName: GenerateFileName(mimeType),
		Reader:   bytes.NewReader(data),
	}, nil
}

// ProcessReaderMedia processa mídia de um Reader
func (p *MediaProcessor) ProcessReaderMedia(reader io.Reader, fileName, mimeType string) (*ProcessedMedia, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler dados do reader: %w", err)
	}

	// Detectar tipo MIME se não fornecido
	if mimeType == "" {
		mimeType = DetectMimeType(data)
	}

	// Gerar nome de arquivo se não fornecido
	if fileName == "" {
		fileName = GenerateFileName(mimeType)
	} else {
		fileName = SanitizeFileName(fileName)
	}

	return &ProcessedMedia{
		Data:     data,
		MimeType: mimeType,
		Size:     int64(len(data)),
		FileName: fileName,
		Reader:   bytes.NewReader(data),
	}, nil
}

// ValidateMedia valida se a mídia está em formato correto
func (p *MediaProcessor) ValidateMedia(media *ProcessedMedia, mediaType string) error {
	// Validar tamanho
	if err := ValidateFileSize(media.Size, mediaType); err != nil {
		return err
	}

	// Validar formato baseado no tipo
	switch mediaType {
	case "image":
		return ValidateImageFormat(media.Data)
	case "video":
		return ValidateVideoFormat(media.Data)
	case "audio":
		return ValidateAudioFormat(media.Data)
	case "document":
		// Documentos podem ter vários formatos, validação básica
		if media.Size == 0 {
			return fmt.Errorf("documento vazio")
		}
		return nil
	case "sticker":
		// Stickers devem ser imagens
		return ValidateImageFormat(media.Data)
	default:
		return fmt.Errorf("tipo de mídia não suportado: %s", mediaType)
	}
}

// ProcessedMedia representa mídia processada
type ProcessedMedia struct {
	Data      []byte    `json:"-"`
	MimeType  string    `json:"mime_type"`
	Size      int64     `json:"size"`
	FileName  string    `json:"file_name"`
	Reader    io.Reader `json:"-"`
	Thumbnail []byte    `json:"-,omitempty"`
}

// GetReader retorna um novo Reader para os dados
func (pm *ProcessedMedia) GetReader() io.Reader {
	return bytes.NewReader(pm.Data)
}

// CreateThumbnail cria um thumbnail se aplicável
func (pm *ProcessedMedia) CreateThumbnail() error {
	// Só criar thumbnail para imagens
	if !pm.IsImage() {
		return nil
	}

	thumbnail, err := CreateThumbnail(pm.Data)
	if err != nil {
		return fmt.Errorf("erro ao criar thumbnail: %w", err)
	}

	pm.Thumbnail = thumbnail
	return nil
}

// IsImage verifica se a mídia é uma imagem
func (pm *ProcessedMedia) IsImage() bool {
	return pm.MimeType != "" && pm.MimeType[:6] == "image/"
}

// IsVideo verifica se a mídia é um vídeo
func (pm *ProcessedMedia) IsVideo() bool {
	return pm.MimeType != "" && pm.MimeType[:6] == "video/"
}

// IsAudio verifica se a mídia é um áudio
func (pm *ProcessedMedia) IsAudio() bool {
	return pm.MimeType != "" && pm.MimeType[:6] == "audio/"
}

// IsDocument verifica se a mídia é um documento
func (pm *ProcessedMedia) IsDocument() bool {
	return pm.MimeType != "" && pm.MimeType[:12] == "application/"
}

// GetSizeFormatted retorna o tamanho formatado
func (pm *ProcessedMedia) GetSizeFormatted() string {
	const unit = 1024
	if pm.Size < unit {
		return fmt.Sprintf("%d B", pm.Size)
	}
	div, exp := int64(unit), 0
	for n := pm.Size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(pm.Size)/float64(div), "KMGTPE"[exp])
}

// MediaValidationError representa um erro de validação de mídia
type MediaValidationError struct {
	Field   string
	Message string
}

func (e *MediaValidationError) Error() string {
	return fmt.Sprintf("validação de mídia falhou em %s: %s", e.Field, e.Message)
}

// NewMediaValidationError cria um novo erro de validação
func NewMediaValidationError(field, message string) *MediaValidationError {
	return &MediaValidationError{
		Field:   field,
		Message: message,
	}
}

// MediaProcessingError representa um erro de processamento de mídia
type MediaProcessingError struct {
	Operation string
	Message   string
	Cause     error
}

func (e *MediaProcessingError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("erro ao %s: %s (causa: %v)", e.Operation, e.Message, e.Cause)
	}
	return fmt.Sprintf("erro ao %s: %s", e.Operation, e.Message)
}

func (e *MediaProcessingError) Unwrap() error {
	return e.Cause
}

// NewMediaProcessingError cria um novo erro de processamento
func NewMediaProcessingError(operation, message string, cause error) *MediaProcessingError {
	return &MediaProcessingError{
		Operation: operation,
		Message:   message,
		Cause:     cause,
	}
}
