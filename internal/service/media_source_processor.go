package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"wamex/internal/domain"
	"wamex/internal/repository"
	"wamex/pkg/logger"
	"wamex/pkg/storage"
)

// MediaSourceProcessor processa diferentes fontes de mídia de forma unificada
type MediaSourceProcessor struct {
	mediaRepo         *repository.MediaRepository
	minioClient       *storage.MinIOClient
	mediaService      *MediaService
	validationService *MediaValidationService
	autoDetector      *AutoTypeDetector
}

// NewMediaSourceProcessor cria uma nova instância do processador
func NewMediaSourceProcessor(
	mediaRepo *repository.MediaRepository,
	minioClient *storage.MinIOClient,
	mediaService *MediaService,
	validationService *MediaValidationService,
) *MediaSourceProcessor {
	return &MediaSourceProcessor{
		mediaRepo:         mediaRepo,
		minioClient:       minioClient,
		mediaService:      mediaService,
		validationService: validationService,
		autoDetector:      NewAutoTypeDetector(),
	}
}

// ProcessRequest processa uma requisição de mídia multi-source
func (p *MediaSourceProcessor) ProcessRequest(
	req *domain.SendMediaMessageRequest,
	file multipart.File,
	header *multipart.FileHeader,
) (*domain.ProcessedMedia, error) {
	startTime := time.Now()

	logger.WithComponent("media-source-processor").Info().
		Str("media_id", req.MediaID).
		Bool("has_base64", req.Base64 != "").
		Bool("has_url", req.URL != "").
		Bool("has_file", file != nil).
		Msg("Processando requisição multi-source")

	// Determinar qual fonte foi fornecida
	sourceCount := 0
	var source string

	if req.MediaID != "" {
		sourceCount++
		source = "mediaId"
	}
	if req.Base64 != "" {
		sourceCount++
		source = "base64"
	}
	if req.URL != "" {
		sourceCount++
		source = "url"
	}
	if file != nil {
		sourceCount++
		source = "upload"
	}

	// Validar que apenas uma fonte foi fornecida
	if sourceCount == 0 {
		return nil, fmt.Errorf("nenhuma fonte de mídia fornecida (mediaId, base64, url ou file)")
	}
	if sourceCount > 1 {
		return nil, fmt.Errorf("apenas uma fonte de mídia deve ser fornecida por vez")
	}

	// Processar baseado na fonte
	var processed *domain.ProcessedMedia
	var err error

	switch source {
	case "mediaId":
		processed, err = p.processMinIOSource(req.MediaID)
	case "base64":
		processed, err = p.processBase64Source(req.Base64, req.Filename)
	case "url":
		processed, err = p.processURLSource(req.URL, req.Filename)
	case "upload":
		processed, err = p.processUploadSource(file, header)
	default:
		return nil, fmt.Errorf("fonte de mídia não reconhecida: %s", source)
	}

	if err != nil {
		return nil, fmt.Errorf("erro ao processar fonte %s: %w", source, err)
	}

	// Definir fonte e tempo de processamento
	processed.Source = source
	processed.ProcessingTime = time.Since(startTime)

	// Override do tipo de mensagem se fornecido
	if req.MessageType != "" {
		if domain.MessageType(req.MessageType) != processed.MessageType {
			logger.WithComponent("media-source-processor").Info().
				Str("detected_type", string(processed.MessageType)).
				Str("override_type", req.MessageType).
				Msg("Tipo de mensagem sobrescrito manualmente")
			processed.MessageType = domain.MessageType(req.MessageType)
		}
	}

	// Override do filename se fornecido
	if req.Filename != "" {
		processed.Filename = req.Filename
	}

	logger.WithComponent("media-source-processor").Info().
		Str("source", source).
		Str("message_type", string(processed.MessageType)).
		Str("mime_type", processed.MimeType).
		Int64("size", processed.Size).
		Dur("processing_time", processed.ProcessingTime).
		Msg("Processamento concluído com sucesso")

	return processed, nil
}

// processMinIOSource processa mídia do MinIO (fonte atual)
func (p *MediaSourceProcessor) processMinIOSource(mediaID string) (*domain.ProcessedMedia, error) {
	logger.WithComponent("media-source-processor").Debug().
		Str("media_id", mediaID).
		Msg("Processando fonte MinIO")

	// Buscar metadados no banco
	ctx := context.Background()
	mediaFile, err := p.mediaRepo.GetByID(ctx, mediaID)
	if err != nil {
		return nil, fmt.Errorf("mídia não encontrada: %w", err)
	}

	// Baixar dados do MinIO
	data, err := p.minioClient.DownloadFile(ctx, "wamex-media", mediaFile.FilePath)
	if err != nil {
		return nil, fmt.Errorf("erro ao baixar do MinIO: %w", err)
	}

	return &domain.ProcessedMedia{
		Data:        data,
		MimeType:    mediaFile.MimeType,
		MessageType: domain.MessageType(mediaFile.MessageType),
		Size:        mediaFile.Size,
		Filename:    mediaFile.Filename,
	}, nil
}

// processBase64Source processa mídia em base64
func (p *MediaSourceProcessor) processBase64Source(base64Data, filename string) (*domain.ProcessedMedia, error) {
	logger.WithComponent("media-source-processor").Debug().
		Bool("has_filename", filename != "").
		Msg("Processando fonte Base64")

	// Usar detector automático com MediaService
	messageType, mimeType, data, err := p.autoDetector.DetectFromBase64(base64Data, p.mediaService)
	if err != nil {
		return nil, fmt.Errorf("erro na detecção automática: %w", err)
	}

	// Validar tamanho
	if err := p.mediaService.ValidateFileSize(data, messageType); err != nil {
		return nil, err
	}

	// Gerar filename se não fornecido
	if filename == "" {
		filename = p.generateFilename(mimeType)
	}

	return &domain.ProcessedMedia{
		Data:        data,
		MimeType:    mimeType,
		MessageType: messageType,
		Size:        int64(len(data)),
		Filename:    filename,
	}, nil
}

// processURLSource processa mídia de URL externa
func (p *MediaSourceProcessor) processURLSource(url, filename string) (*domain.ProcessedMedia, error) {
	logger.WithComponent("media-source-processor").Debug().
		Str("url", url).
		Msg("Processando fonte URL")

	// Detectar tipo pela URL
	messageType, mimeType, err := p.autoDetector.DetectFromURL(url)
	if err != nil {
		return nil, fmt.Errorf("erro na detecção por URL: %w", err)
	}

	// Download da URL
	data, actualMimeType, err := p.downloadFromURL(url)
	if err != nil {
		return nil, fmt.Errorf("erro no download: %w", err)
	}

	// Usar MIME type real se diferente
	if actualMimeType != "" && actualMimeType != mimeType {
		mimeType = actualMimeType
		messageType = domain.DetectMessageTypeFromMime(mimeType)
	}

	// Validar tamanho
	if err := p.mediaService.ValidateFileSize(data, messageType); err != nil {
		return nil, err
	}

	// Gerar filename se não fornecido
	if filename == "" {
		filename = p.generateFilename(mimeType)
	}

	return &domain.ProcessedMedia{
		Data:        data,
		MimeType:    mimeType,
		MessageType: messageType,
		Size:        int64(len(data)),
		Filename:    filename,
	}, nil
}

// processUploadSource processa upload direto com validações robustas
func (p *MediaSourceProcessor) processUploadSource(file multipart.File, header *multipart.FileHeader) (*domain.ProcessedMedia, error) {
	logger.WithComponent("media-source-processor").Debug().
		Str("filename", header.Filename).
		Int64("size", header.Size).
		Msg("Processando upload direto")

	// Validar tamanho do header primeiro
	maxSize := int64(100 * 1024 * 1024) // 100MB máximo
	if header.Size > maxSize {
		return nil, fmt.Errorf("arquivo muito grande: %d bytes (máximo: %d)", header.Size, maxSize)
	}

	// Validar nome do arquivo
	if header.Filename == "" {
		return nil, fmt.Errorf("nome do arquivo é obrigatório")
	}

	// Validar extensão básica
	if err := p.validateFileExtension(header.Filename); err != nil {
		return nil, fmt.Errorf("extensão não permitida: %w", err)
	}

	// Ler dados do arquivo com limite
	limitReader := io.LimitReader(file, maxSize)
	data, err := io.ReadAll(limitReader)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo: %w", err)
	}

	// Validar que o tamanho real corresponde ao header
	if int64(len(data)) != header.Size {
		logger.WithComponent("media-source-processor").Warn().
			Int64("header_size", header.Size).
			Int("actual_size", len(data)).
			Msg("Tamanho do arquivo difere do header")
	}

	// Detectar tipo automaticamente com validação de magic numbers
	messageType, mimeType, err := p.autoDetector.DetectFromData(data, header.Filename)
	if err != nil {
		return nil, fmt.Errorf("erro na detecção automática: %w", err)
	}

	// Validar magic numbers contra extensão (segurança)
	if err := p.validateMagicNumbers(data, header.Filename, mimeType); err != nil {
		return nil, fmt.Errorf("validação de segurança falhou: %w", err)
	}

	// Validar tamanho para o tipo específico
	if err := p.mediaService.ValidateFileSize(data, messageType); err != nil {
		return nil, err
	}

	logger.WithComponent("media-source-processor").Info().
		Str("filename", header.Filename).
		Str("message_type", string(messageType)).
		Str("mime_type", mimeType).
		Int("size", len(data)).
		Msg("Upload direto processado com sucesso")

	return &domain.ProcessedMedia{
		Data:        data,
		MimeType:    mimeType,
		MessageType: messageType,
		Size:        int64(len(data)),
		Filename:    header.Filename,
	}, nil
}

// downloadFromURL faz download seguro de uma URL externa
func (p *MediaSourceProcessor) downloadFromURL(url string) ([]byte, string, error) {
	// Validar URL
	if err := p.validateURL(url); err != nil {
		return nil, "", fmt.Errorf("URL inválida: %w", err)
	}

	logger.WithComponent("media-source-processor").Info().
		Str("url", url).
		Msg("Iniciando download de URL externa")

	// Cliente HTTP com timeout e configurações de segurança
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Limitar redirecionamentos
			if len(via) >= 5 {
				return fmt.Errorf("muitos redirecionamentos")
			}
			// Validar URL de redirecionamento
			return p.validateURL(req.URL.String())
		},
	}

	// Criar request com headers apropriados
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("erro ao criar request: %w", err)
	}

	// Headers de segurança
	req.Header.Set("User-Agent", "WAMEX-Media-Downloader/1.0")
	req.Header.Set("Accept", "image/*,audio/*,video/*,application/pdf,application/msword,application/vnd.*")

	// Executar request
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("erro no download: %w", err)
	}
	defer resp.Body.Close()

	// Validar status
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("status HTTP inválido: %d %s", resp.StatusCode, resp.Status)
	}

	// Validar Content-Length se presente
	if resp.ContentLength > 0 {
		maxSize := int64(100 * 1024 * 1024) // 100MB máximo
		if resp.ContentLength > maxSize {
			return nil, "", fmt.Errorf("arquivo muito grande: %d bytes (máximo: %d)", resp.ContentLength, maxSize)
		}
	}

	// Ler dados com limite
	limitReader := io.LimitReader(resp.Body, 100*1024*1024) // 100MB máximo
	data, err := io.ReadAll(limitReader)
	if err != nil {
		return nil, "", fmt.Errorf("erro ao ler resposta: %w", err)
	}

	// Detectar MIME type real
	mimeType := resp.Header.Get("Content-Type")
	if mimeType != "" {
		// Limpar parâmetros extras (charset, etc)
		if idx := strings.Index(mimeType, ";"); idx != -1 {
			mimeType = strings.TrimSpace(mimeType[:idx])
		}
	}

	// Usar magic numbers se Content-Type não estiver presente ou for genérico
	if mimeType == "" || mimeType == "application/octet-stream" {
		detectedMime := http.DetectContentType(data)
		if detectedMime != "application/octet-stream" {
			mimeType = detectedMime
		}
	}

	logger.WithComponent("media-source-processor").Info().
		Str("url", url).
		Str("mime_type", mimeType).
		Int("data_size", len(data)).
		Int("status_code", resp.StatusCode).
		Msg("Download concluído com sucesso")

	return data, mimeType, nil
}

// generateFilename gera um nome de arquivo baseado no MIME type
func (p *MediaSourceProcessor) generateFilename(mimeType string) string {
	timestamp := time.Now().Format("20060102_150405")

	extensions := map[string]string{
		"image/jpeg":      ".jpg",
		"image/png":       ".png",
		"image/gif":       ".gif",
		"image/webp":      ".webp",
		"audio/mpeg":      ".mp3",
		"audio/ogg":       ".ogg",
		"video/mp4":       ".mp4",
		"video/3gpp":      ".3gp",
		"application/pdf": ".pdf",
	}

	ext := extensions[mimeType]
	if ext == "" {
		ext = ".bin"
	}

	return fmt.Sprintf("media_%s%s", timestamp, ext)
}

// validateURL valida se uma URL é segura para download
func (p *MediaSourceProcessor) validateURL(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("URL malformada: %w", err)
	}

	// Validar esquema
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("esquema não suportado: %s (apenas http/https)", parsedURL.Scheme)
	}

	// Validar host
	if parsedURL.Host == "" {
		return fmt.Errorf("host vazio")
	}

	// Blacklist de IPs locais/privados (segurança básica)
	host := strings.ToLower(parsedURL.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" ||
		strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "10.") ||
		strings.HasPrefix(host, "172.16.") || strings.HasPrefix(host, "172.17.") ||
		strings.HasPrefix(host, "172.18.") || strings.HasPrefix(host, "172.19.") ||
		strings.HasPrefix(host, "172.2") || strings.HasPrefix(host, "172.30.") ||
		strings.HasPrefix(host, "172.31.") {
		return fmt.Errorf("acesso a redes privadas não permitido")
	}

	// Whitelist básica de domínios conhecidos (pode ser expandida)
	allowedDomains := []string{
		"imgur.com", "i.imgur.com",
		"github.com", "raw.githubusercontent.com",
		"dropbox.com", "dl.dropboxusercontent.com",
		"drive.google.com", "docs.google.com",
		"onedrive.live.com",
		"amazonaws.com", "s3.amazonaws.com",
		"cloudfront.net",
		"cdn.discordapp.com", "media.discordapp.net",
		"telegram.org", "t.me",
	}

	// Verificar se o domínio está na whitelist ou é um subdomínio permitido
	domainAllowed := false
	for _, allowedDomain := range allowedDomains {
		if host == allowedDomain || strings.HasSuffix(host, "."+allowedDomain) {
			domainAllowed = true
			break
		}
	}

	if !domainAllowed {
		logger.WithComponent("media-source-processor").Warn().
			Str("host", host).
			Msg("Domínio não está na whitelist")
		// Por enquanto, apenas log de warning - pode ser configurado para bloquear
	}

	return nil
}

// validateFileExtension valida se a extensão do arquivo é permitida
func (p *MediaSourceProcessor) validateFileExtension(filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))

	allowedExtensions := []string{
		// Imagens
		".jpg", ".jpeg", ".png", ".gif", ".webp",
		// Áudio
		".mp3", ".ogg", ".aac", ".amr", ".wav",
		// Vídeo
		".mp4", ".3gp",
		// Documentos
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt",
	}

	for _, allowed := range allowedExtensions {
		if ext == allowed {
			return nil
		}
	}

	return fmt.Errorf("extensão %s não permitida", ext)
}

// validateMagicNumbers valida se os magic numbers correspondem à extensão
func (p *MediaSourceProcessor) validateMagicNumbers(data []byte, filename, mimeType string) error {
	if len(data) < 4 {
		return fmt.Errorf("arquivo muito pequeno para validação")
	}

	ext := strings.ToLower(filepath.Ext(filename))

	// Verificações básicas de magic numbers
	magicChecks := map[string][]byte{
		".jpg":  {0xFF, 0xD8, 0xFF},
		".jpeg": {0xFF, 0xD8, 0xFF},
		".png":  {0x89, 0x50, 0x4E, 0x47},
		".gif":  {0x47, 0x49, 0x46},
		".pdf":  {0x25, 0x50, 0x44, 0x46},
		".mp3":  {0x49, 0x44, 0x33}, // ID3
		".mp4":  {0x00, 0x00, 0x00}, // Pode variar, verificação mais complexa necessária
	}

	if expectedMagic, exists := magicChecks[ext]; exists {
		if len(data) >= len(expectedMagic) {
			match := true
			for i, b := range expectedMagic {
				if data[i] != b {
					match = false
					break
				}
			}

			if !match {
				logger.WithComponent("media-source-processor").Warn().
					Str("filename", filename).
					Str("extension", ext).
					Str("mime_type", mimeType).
					Msg("Magic numbers não correspondem à extensão")
				// Por enquanto apenas warning, pode ser configurado para bloquear
			}
		}
	}

	return nil
}
