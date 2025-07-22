package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"
	"zapcore/pkg/logger"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"zapcore/internal/app/config"
)

// MinIOClient representa o cliente MinIO para armazenamento de mídia
type MinIOClient struct {
	client        *minio.Client
	defaultBucket string
	logger        *logger.Logger
}

// NewMinIOClient cria uma nova instância do cliente MinIO
func NewMinIOClient(cfg *config.MinIOConfig) (*MinIOClient, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("MinIO está desabilitado na configuração")
	}

	// Criar cliente MinIO
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao criar cliente MinIO: %w", err)
	}

	minioClient := &MinIOClient{
		client:        client,
		defaultBucket: cfg.DefaultBucket,
		logger:        logger.Get().WithField("component", "minio"),
	}

	// Verificar conexão e criar bucket se necessário
	if err := minioClient.ensureBucket(context.Background()); err != nil {
		return nil, fmt.Errorf("erro ao verificar/criar bucket: %w", err)
	}

	minioClient.logger.WithFields(map[string]interface{}{
		"component": "storage",
		"provider":  "minio",
		"endpoint":  cfg.Endpoint,
		"bucket":    cfg.DefaultBucket,
		"ssl":       cfg.UseSSL,
		"status":    "initialized",
	}).Info().Msg("📦 Cliente MinIO OK")

	return minioClient, nil
}

// ensureBucket verifica se o bucket existe e cria se necessário
func (m *MinIOClient) ensureBucket(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.defaultBucket)
	if err != nil {
		return fmt.Errorf("erro ao verificar existência do bucket: %w", err)
	}

	if !exists {
		err = m.client.MakeBucket(ctx, m.defaultBucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("erro ao criar bucket: %w", err)
		}
		m.logger.Info().Str("bucket", m.defaultBucket).Msg("Bucket criado com sucesso")
	}

	return nil
}

// MediaUploadOptions opções para upload de mídia
type MediaUploadOptions struct {
	SessionID   uuid.UUID
	ChatJID     string
	Direction   string // "inbound" ou "outbound"
	MessageID   string
	ContentType string
	Extension   string
	Size        int64
}

// UploadMedia faz upload de mídia para o MinIO seguindo a estrutura de paths
func (m *MinIOClient) UploadMedia(ctx context.Context, reader io.Reader, opts MediaUploadOptions) (string, error) {
	uploadStart := time.Now()

	// Construir path seguindo o padrão: {sessionID}/{chatJID}/{direction}/{messageID}.{extension}
	objectPath := m.buildMediaPath(opts)

	m.logger.Debug().
		Str("session_id", opts.SessionID.String()).
		Str("message_id", opts.MessageID).
		Str("object_path", objectPath).
		Int64("size", opts.Size).
		Str("content_type", opts.ContentType).
		Msg("🚀 Upload MinIO")

	// Fazer upload
	uploadInfo, err := m.client.PutObject(ctx, m.defaultBucket, objectPath, reader, opts.Size, minio.PutObjectOptions{
		ContentType: opts.ContentType,
		UserMetadata: map[string]string{
			"session-id": opts.SessionID.String(),
			"chat-jid":   opts.ChatJID,
			"direction":  opts.Direction,
			"message-id": opts.MessageID,
		},
	})
	if err != nil {
		m.logger.Error().
			Err(err).
			Str("session_id", opts.SessionID.String()).
			Str("message_id", opts.MessageID).
			Str("object_path", objectPath).
			Str("bucket", m.defaultBucket).
			Msg("❌ Erro upload MinIO")
		return "", fmt.Errorf("erro ao fazer upload da mídia: %w", err)
	}

	uploadDuration := time.Since(uploadStart)

	m.logger.Info().
		Str("object_path", objectPath).
		Str("session_id", opts.SessionID.String()).
		Str("message_id", opts.MessageID).
		Int64("size", uploadInfo.Size).
		Str("etag", uploadInfo.ETag).
		Dur("upload_duration", uploadDuration).
		Msg("✅ Upload MinIO OK")

	return objectPath, nil
}

// buildMediaPath constrói o path da mídia seguindo o padrão definido
func (m *MinIOClient) buildMediaPath(opts MediaUploadOptions) string {
	// Construir path: {sessionID}/{chatJID}/{direction}/{messageID}.{extension}
	// Usando chatJID real sem sanitização, pois MinIO suporta @ e .
	return filepath.Join(
		opts.SessionID.String(),
		opts.ChatJID,
		opts.Direction,
		fmt.Sprintf("%s.%s", opts.MessageID, opts.Extension),
	)
}

// GetMediaURL retorna a URL para acessar a mídia
func (m *MinIOClient) GetMediaURL(ctx context.Context, objectPath string) (string, error) {
	// Gerar URL pré-assinada válida por 24 horas
	url, err := m.client.PresignedGetObject(ctx, m.defaultBucket, objectPath, 24*60*60, nil)
	if err != nil {
		return "", fmt.Errorf("erro ao gerar URL da mídia: %w", err)
	}

	return url.String(), nil
}

// DeleteMedia remove mídia do MinIO
func (m *MinIOClient) DeleteMedia(ctx context.Context, objectPath string) error {
	err := m.client.RemoveObject(ctx, m.defaultBucket, objectPath, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("erro ao remover mídia: %w", err)
	}

	m.logger.Info().Str("object_path", objectPath).Msg("Mídia removida do MinIO com sucesso")
	return nil
}

// GetMediaInfo retorna informações sobre a mídia
func (m *MinIOClient) GetMediaInfo(ctx context.Context, objectPath string) (*minio.ObjectInfo, error) {
	info, err := m.client.StatObject(ctx, m.defaultBucket, objectPath, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("erro ao obter informações da mídia: %w", err)
	}

	return &info, nil
}

// HealthCheck verifica se o MinIO está acessível
func (m *MinIOClient) HealthCheck(ctx context.Context) error {
	_, err := m.client.BucketExists(ctx, m.defaultBucket)
	if err != nil {
		return fmt.Errorf("MinIO não está acessível: %w", err)
	}
	return nil
}
