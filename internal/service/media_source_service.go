package service

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wamex/internal/domain"
	"wamex/pkg/logger"
	"wamex/pkg/storage"
)

// MediaSourceService gerencia múltiplas fontes de mídia
type MediaSourceService struct {
	minioClient *storage.MinIOClient
	projectRoot string
}

// NewMediaSourceService cria uma nova instância do MediaSourceService
func NewMediaSourceService(minioClient *storage.MinIOClient, projectRoot string) *MediaSourceService {
	return &MediaSourceService{
		minioClient: minioClient,
		projectRoot: projectRoot,
	}
}

// MediaSourceRequest representa uma requisição de mídia com múltiplas fontes
type MediaSourceRequest struct {
	// Apenas uma dessas opções deve ser fornecida
	Base64   string // data:image/png;base64,...
	FilePath string // assets/image.png
	URL      string // https://example.com/image.png
	MinioID  string // media/2025/01/02/image_123.png

	// Metadados opcionais
	MimeType string
	Filename string
}

// MediaSourceResult resultado do processamento da fonte de mídia
type MediaSourceResult struct {
	Data     []byte
	MimeType string
	Filename string
	Size     int64
	Source   string // "base64", "file", "url", "minio"
}

// ProcessMediaSource processa mídia de qualquer fonte suportada
func (ms *MediaSourceService) ProcessMediaSource(req MediaSourceRequest, messageType domain.MessageType) (*MediaSourceResult, error) {
	// Valida que apenas uma fonte foi fornecida
	sources := []string{}
	if req.Base64 != "" {
		sources = append(sources, "base64")
	}
	if req.FilePath != "" {
		sources = append(sources, "filePath")
	}
	if req.URL != "" {
		sources = append(sources, "url")
	}
	if req.MinioID != "" {
		sources = append(sources, "minioId")
	}

	if len(sources) == 0 {
		return nil, fmt.Errorf("nenhuma fonte de mídia fornecida")
	}
	if len(sources) > 1 {
		return nil, fmt.Errorf("apenas uma fonte de mídia deve ser fornecida, recebido: %v", sources)
	}

	var result *MediaSourceResult
	var err error

	// Processa baseado na fonte fornecida
	switch sources[0] {
	case "base64":
		result, err = ms.processBase64(req.Base64)
	case "filePath":
		result, err = ms.processFilePath(req.FilePath)
	case "url":
		result, err = ms.processURL(req.URL)
	case "minioId":
		result, err = ms.processMinioID(req.MinioID)
	default:
		return nil, fmt.Errorf("fonte de mídia não suportada: %s", sources[0])
	}

	if err != nil {
		return nil, err
	}

	// Aplica metadados fornecidos
	if req.MimeType != "" {
		result.MimeType = req.MimeType
	}
	if req.Filename != "" {
		result.Filename = req.Filename
	}

	// Detecta MIME type se não fornecido
	if result.MimeType == "" {
		result.MimeType = http.DetectContentType(result.Data)
	}

	// Valida tipo MIME para o tipo de mensagem
	if !domain.IsValidMimeType(messageType, result.MimeType) {
		return nil, fmt.Errorf("tipo MIME %s não suportado para %s", result.MimeType, messageType)
	}

	// Valida tamanho do arquivo
	maxSize := domain.GetMaxFileSize(messageType)
	if maxSize > 0 && result.Size > maxSize {
		return nil, fmt.Errorf("arquivo muito grande: %d bytes (máximo: %d bytes)", result.Size, maxSize)
	}

	logger.WithComponent("media-source").Info().
		Str("source", result.Source).
		Str("mime_type", result.MimeType).
		Int64("size", result.Size).
		Str("filename", result.Filename).
		Str("message_type", string(messageType)).
		Msg("Mídia processada com sucesso")

	return result, nil
}

// processBase64 processa mídia em formato base64
func (ms *MediaSourceService) processBase64(dataURL string) (*MediaSourceResult, error) {
	logger.WithComponent("media-source").Debug().
		Str("source", "base64").
		Msg("Processando mídia base64")

	mediaService := NewMediaService()
	data, mimeType, err := mediaService.DecodeBase64Media(dataURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao processar base64: %w", err)
	}

	return &MediaSourceResult{
		Data:     data,
		MimeType: mimeType,
		Size:     int64(len(data)),
		Source:   "base64",
	}, nil
}

// processFilePath processa arquivo local
func (ms *MediaSourceService) processFilePath(filePath string) (*MediaSourceResult, error) {
	logger.WithComponent("media-source").Debug().
		Str("source", "file").
		Str("file_path", filePath).
		Str("project_root", ms.projectRoot).
		Msg("Processando arquivo local")

	// Constrói caminho absoluto relativo ao projeto
	fullPath := filepath.Join(ms.projectRoot, filePath)

	logger.WithComponent("media-source").Debug().
		Str("full_path", fullPath).
		Msg("Caminho completo do arquivo")

	// Verifica se o arquivo existe
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		logger.WithComponent("media-source").Error().
			Str("file_path", filePath).
			Str("full_path", fullPath).
			Msg("Arquivo não encontrado")
		return nil, fmt.Errorf("arquivo não encontrado: %s", filePath)
	}

	// Lê o arquivo
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo %s: %w", filePath, err)
	}

	// Extrai nome do arquivo
	filename := filepath.Base(filePath)

	return &MediaSourceResult{
		Data:     data,
		Filename: filename,
		Size:     int64(len(data)),
		Source:   "file",
	}, nil
}

// processURL processa URL externa
func (ms *MediaSourceService) processURL(url string) (*MediaSourceResult, error) {
	logger.WithComponent("media-source").Debug().
		Str("source", "url").
		Str("url", url).
		Msg("Processando URL externa")

	// Cria cliente HTTP com timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Faz o download
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer download da URL %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erro HTTP %d ao fazer download da URL %s", resp.StatusCode, url)
	}

	// Lê o conteúdo
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler conteúdo da URL %s: %w", url, err)
	}

	// Extrai nome do arquivo da URL
	filename := filepath.Base(url)
	if strings.Contains(filename, "?") {
		filename = strings.Split(filename, "?")[0]
	}

	// Obtém MIME type do header se disponível
	mimeType := resp.Header.Get("Content-Type")
	if strings.Contains(mimeType, ";") {
		mimeType = strings.Split(mimeType, ";")[0]
	}

	return &MediaSourceResult{
		Data:     data,
		MimeType: mimeType,
		Filename: filename,
		Size:     int64(len(data)),
		Source:   "url",
	}, nil
}

// processMinioID processa mídia já armazenada no MinIO
func (ms *MediaSourceService) processMinioID(minioID string) (*MediaSourceResult, error) {
	logger.WithComponent("media-source").Debug().
		Str("source", "minio").
		Str("minio_id", minioID).
		Msg("Processando mídia do MinIO")

	// Se o minioID é uma URL completa do MinIO, usa diretamente
	var downloadURL string
	if strings.HasPrefix(minioID, "https://minio.resolvecert.com/") {
		downloadURL = minioID
		logger.WithComponent("media-source").Debug().
			Str("download_url", downloadURL).
			Msg("Usando URL completa do MinIO")
	} else {
		// Se é apenas um ID, constrói a URL base (implementação futura)
		return nil, fmt.Errorf("formato de MinIO ID não suportado ainda: %s. Use URL completa do MinIO", minioID)
	}

	// Faz o download da URL do MinIO
	client := &http.Client{
		Timeout: 60 * time.Second, // Timeout maior para arquivos grandes
	}

	logger.WithComponent("media-source").Debug().
		Str("url", downloadURL).
		Msg("Fazendo download do MinIO")

	resp, err := client.Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer download do MinIO %s: %w", downloadURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erro HTTP %d ao fazer download do MinIO %s", resp.StatusCode, downloadURL)
	}

	// Lê o conteúdo
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler conteúdo do MinIO %s: %w", downloadURL, err)
	}

	// Extrai nome do arquivo da URL
	filename := "minio-file"
	if strings.Contains(downloadURL, "/") {
		parts := strings.Split(downloadURL, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i] != "" && !strings.Contains(parts[i], "?") {
				filename = parts[i]
				break
			}
		}
	}

	// Remove parâmetros de query do filename se existirem
	if strings.Contains(filename, "?") {
		filename = strings.Split(filename, "?")[0]
	}

	// Obtém MIME type do header se disponível
	mimeType := resp.Header.Get("Content-Type")
	if strings.Contains(mimeType, ";") {
		mimeType = strings.Split(mimeType, ";")[0]
	}

	logger.WithComponent("media-source").Info().
		Str("filename", filename).
		Str("mime_type", mimeType).
		Int64("size", int64(len(data))).
		Msg("Download do MinIO concluído com sucesso")

	return &MediaSourceResult{
		Data:     data,
		MimeType: mimeType,
		Filename: filename,
		Size:     int64(len(data)),
		Source:   "minio",
	}, nil
}

// GetProjectRoot retorna o diretório raiz do projeto
func GetProjectRoot() string {
	// Tenta encontrar o diretório raiz baseado na presença de go.mod
	dir, err := os.Getwd()
	if err != nil {
		logger.WithComponent("media-source").Error().
			Err(err).
			Msg("Erro ao obter diretório atual")
		return "."
	}

	logger.WithComponent("media-source").Debug().
		Str("current_dir", dir).
		Msg("Diretório atual")

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			logger.WithComponent("media-source").Debug().
				Str("project_root", dir).
				Msg("Diretório raiz do projeto encontrado")
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	logger.WithComponent("media-source").Warn().
		Msg("go.mod não encontrado, usando diretório atual")
	return "."
}
