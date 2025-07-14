package infra

import (
	"context"
	"fmt"

	"wamex/pkg/logger"
	"wamex/pkg/storage"
)

// StorageConnection representa uma conexão com o sistema de storage
type StorageConnection struct {
	MinIO *storage.MinIOClient
}

// NewStorageConnection cria uma nova conexão com o storage
func NewStorageConnection() (*StorageConnection, error) {
	minioClient, err := storage.InitializeMinIO()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO: %w", err)
	}

	logger.WithComponent("storage").Info().
		Msg("📦 Storage connection initialized successfully")

	return &StorageConnection{
		MinIO: minioClient,
	}, nil
}

// HealthCheck verifica se a conexão com storage está saudável
func (sc *StorageConnection) HealthCheck(ctx context.Context) error {
	// Tenta verificar se um bucket padrão existe para testar conectividade
	err := sc.MinIO.CreateBucketIfNotExists(ctx, "wamex-media", false)
	if err != nil {
		return fmt.Errorf("storage health check failed: %w", err)
	}

	logger.WithComponent("storage").Debug().
		Msg("Storage health check passed")

	return nil
}

// GetMinIOClient retorna o cliente MinIO
func (sc *StorageConnection) GetMinIOClient() *storage.MinIOClient {
	return sc.MinIO
}

// Close fecha a conexão com storage (se necessário)
func (sc *StorageConnection) Close() error {
	// MinIO client não precisa de close explícito
	logger.WithComponent("storage").Info().
		Msg("Storage connection closed")
	return nil
}
