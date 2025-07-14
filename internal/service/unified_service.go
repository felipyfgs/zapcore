package service

import (
	"context"
	"fmt"
	"time"

	"wamex/internal/domain"
	"wamex/internal/repository"
	"wamex/pkg/storage"
)

// UnifiedMediaService integra todas as funcionalidades de mídia em um único service
type UnifiedMediaService struct {
	// Componentes internos
	baseService *MediaService
	validator   *MediaValidationService
	security    *MediaSecurityService
	detector    *AutoTypeDetector

	// Dependências
	mediaRepo   *repository.MediaRepository
	minioClient *storage.MinIOClient
}

// NewUnifiedMediaService cria uma nova instância do serviço unificado
func NewUnifiedMediaService(
	mediaRepo *repository.MediaRepository,
	minioClient *storage.MinIOClient,
) *UnifiedMediaService {
	// Criar componentes internos
	baseService := NewMediaService()
	validator := NewMediaValidationService()
	security := NewMediaSecurityService()
	detector := NewAutoTypeDetector()

	return &UnifiedMediaService{
		baseService: baseService,
		validator:   validator,
		security:    security,
		detector:    detector,
		mediaRepo:   mediaRepo,
		minioClient: minioClient,
	}
}

// ValidateRateLimit verifica rate limiting
func (ums *UnifiedMediaService) ValidateRateLimit(
	identifier string,
	maxRequests int,
	window time.Duration,
) error {
	return ums.security.ValidateRateLimit(identifier, maxRequests, window)
}

// ValidateDomain verifica se domínio está na whitelist
func (ums *UnifiedMediaService) ValidateDomain(url string) error {
	return ums.security.ValidateDomain(url)
}

// DetectFromData detecta tipo de mídia a partir dos dados
func (ums *UnifiedMediaService) DetectFromData(
	data []byte,
	filename string,
) (domain.MessageType, string, error) {
	return ums.detector.DetectFromData(data, filename)
}

// DetectFromBase64 detecta tipo de mídia a partir de base64
func (ums *UnifiedMediaService) DetectFromBase64(
	base64Data string,
	mediaService *MediaService,
) (domain.MessageType, string, []byte, error) {
	return ums.detector.DetectFromBase64(base64Data, mediaService)
}

// DetectFromURL detecta tipo de mídia a partir de URL
func (ums *UnifiedMediaService) DetectFromURL(url string) (domain.MessageType, string, error) {
	return ums.detector.DetectFromURL(url)
}

// --- Métodos do MediaService base ---

// DecodeBase64Media decodifica mídia em base64
func (ums *UnifiedMediaService) DecodeBase64Media(dataURL string) ([]byte, string, error) {
	return ums.baseService.DecodeBase64Media(dataURL)
}

// ValidateMediaType valida tipo MIME para tipo de mensagem
func (ums *UnifiedMediaService) ValidateMediaType(mimeType string, messageType domain.MessageType) error {
	return ums.baseService.ValidateMediaType(mimeType, messageType)
}

// ValidateFileSize valida tamanho do arquivo
func (ums *UnifiedMediaService) ValidateFileSize(data []byte, messageType domain.MessageType) error {
	return ums.baseService.ValidateFileSize(data, messageType)
}

// DetectMimeType detecta tipo MIME dos dados
func (ums *UnifiedMediaService) DetectMimeType(data []byte) string {
	return ums.baseService.DetectMimeType(data)
}

// ProcessMediaForUpload processa mídia completa para upload
func (ums *UnifiedMediaService) ProcessMediaForUpload(
	dataURL string,
	messageType domain.MessageType,
) (*ProcessedMedia, error) {
	return ums.baseService.ProcessMediaForUpload(dataURL, messageType)
}

// --- Métodos de repositório ---

// SaveMediaFile salva arquivo de mídia no banco
func (ums *UnifiedMediaService) SaveMediaFile(ctx context.Context, mediaFile *domain.MediaFile) error {
	return ums.mediaRepo.Create(ctx, mediaFile)
}

// GetMediaFile obtém arquivo de mídia por ID
func (ums *UnifiedMediaService) GetMediaFile(ctx context.Context, id string) (*domain.MediaFile, error) {
	return ums.mediaRepo.GetByID(ctx, id)
}

// ListMediaFiles lista arquivos de mídia
func (ums *UnifiedMediaService) ListMediaFiles(
	ctx context.Context,
	limit, offset int,
	messageType, sessionID, sessionName string,
) ([]domain.MediaFile, int, error) {
	return ums.mediaRepo.List(ctx, limit, offset, messageType, sessionID, sessionName)
}

// DeleteMediaFile remove arquivo de mídia
func (ums *UnifiedMediaService) DeleteMediaFile(ctx context.Context, id string) error {
	return ums.mediaRepo.Delete(ctx, id)
}

// CleanupExpiredFiles remove arquivos expirados
func (ums *UnifiedMediaService) CleanupExpiredFiles(ctx context.Context) (int, error) {
	return ums.mediaRepo.CleanupExpired(ctx)
}

// --- Métodos de storage ---

// UploadToMinIO faz upload para MinIO
func (ums *UnifiedMediaService) UploadToMinIO(
	ctx context.Context,
	bucketName, objectName string,
	data []byte,
	contentType string,
) (string, error) {
	return ums.minioClient.UploadFile(ctx, bucketName, objectName, data, contentType)
}

// DownloadFromMinIO baixa arquivo do MinIO
func (ums *UnifiedMediaService) DownloadFromMinIO(ctx context.Context, bucketName, objectName string) ([]byte, error) {
	return ums.minioClient.DownloadFile(ctx, bucketName, objectName)
}

// GetMinIOFileURL obtém URL do MinIO
func (ums *UnifiedMediaService) GetMinIOFileURL(
	bucketName, objectName string,
) string {
	return ums.minioClient.GetFileURL(bucketName, objectName)
}

// --- Métodos utilitários ---

// GetStats obtém estatísticas de mídia
func (ums *UnifiedMediaService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	return ums.mediaRepo.GetStats(ctx)
}

// HealthCheck verifica saúde do serviço
func (ums *UnifiedMediaService) HealthCheck(ctx context.Context) error {
	// Verificar conexão com banco
	_, err := ums.mediaRepo.GetStats(ctx)
	if err != nil {
		return fmt.Errorf("erro na conexão com banco: %w", err)
	}

	return nil
}
