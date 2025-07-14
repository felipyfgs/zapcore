package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wamex/pkg/logger"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOClient wrapper para cliente MinIO
type MinIOClient struct {
	client   *minio.Client
	endpoint string
	useSSL   bool
}

// MinIOConfig configurações para MinIO
type MinIOConfig struct {
	Endpoint         string
	AccessKeyID      string
	SecretAccessKey  string
	UseSSL           bool
	BucketMedia      string
	BucketTemp       string
	BucketThumbnails string
}

// NewMinIOClient cria uma nova instância do cliente MinIO
func NewMinIOClient(config MinIOConfig) (*MinIOClient, error) {
	logger.WithComponent("minio").Info().
		Str("endpoint", config.Endpoint).
		Bool("use_ssl", config.UseSSL).
		Msg("Inicializando cliente MinIO")

	// Cria o cliente MinIO
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Secure: config.UseSSL,
	})
	if err != nil {
		logger.WithComponent("minio").Error().
			Err(err).
			Str("endpoint", config.Endpoint).
			Msg("Erro ao criar cliente MinIO")
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	minioClient := &MinIOClient{
		client:   client,
		endpoint: config.Endpoint,
		useSSL:   config.UseSSL,
	}

	// Testa a conexão
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.ListBuckets(ctx)
	if err != nil {
		logger.WithComponent("minio").Error().
			Err(err).
			Msg("Erro ao testar conexão com MinIO")
		return nil, fmt.Errorf("failed to connect to MinIO: %w", err)
	}

	logger.WithComponent("minio").Info().
		Msg("Cliente MinIO inicializado com sucesso")

	return minioClient, nil
}

// CreateBucketIfNotExists cria um bucket se ele não existir
func (mc *MinIOClient) CreateBucketIfNotExists(ctx context.Context, bucketName string, makePublic bool) error {
	// Verifica se o bucket existe
	exists, err := mc.client.BucketExists(ctx, bucketName)
	if err != nil {
		logger.WithComponent("minio").Error().
			Err(err).
			Str("bucket", bucketName).
			Msg("Erro ao verificar existência do bucket")
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		// Cria o bucket
		err = mc.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			logger.WithComponent("minio").Error().
				Err(err).
				Str("bucket", bucketName).
				Msg("Erro ao criar bucket")
			return fmt.Errorf("failed to create bucket: %w", err)
		}

		logger.WithComponent("minio").Info().
			Str("bucket", bucketName).
			Msg("Bucket criado com sucesso")

		// Define política pública se solicitado
		if makePublic {
			policy := fmt.Sprintf(`{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Effect": "Allow",
						"Principal": {"AWS": ["*"]},
						"Action": ["s3:GetObject"],
						"Resource": ["arn:aws:s3:::%s/*"]
					}
				]
			}`, bucketName)

			err = mc.client.SetBucketPolicy(ctx, bucketName, policy)
			if err != nil {
				logger.WithComponent("minio").Warn().
					Err(err).
					Str("bucket", bucketName).
					Msg("Erro ao definir política pública do bucket")
			} else {
				logger.WithComponent("minio").Info().
					Str("bucket", bucketName).
					Msg("Política pública definida para o bucket")
			}
		}
	} else {
		logger.WithComponent("minio").Debug().
			Str("bucket", bucketName).
			Msg("Bucket já existe")
	}

	return nil
}

// UploadFile faz upload de um arquivo para o MinIO
func (mc *MinIOClient) UploadFile(ctx context.Context, bucketName, objectName string, data []byte, contentType string) (string, error) {
	logger.WithComponent("minio").Debug().
		Str("bucket", bucketName).
		Str("object", objectName).
		Str("content_type", contentType).
		Int("size", len(data)).
		Msg("Fazendo upload de arquivo para MinIO")

	// Faz o upload
	reader := bytes.NewReader(data)
	_, err := mc.client.PutObject(ctx, bucketName, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		logger.WithComponent("minio").Error().
			Err(err).
			Str("bucket", bucketName).
			Str("object", objectName).
			Msg("Erro ao fazer upload para MinIO")
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Gera URL de acesso
	var url string
	if mc.useSSL {
		url = fmt.Sprintf("https://%s/%s/%s", mc.endpoint, bucketName, objectName)
	} else {
		url = fmt.Sprintf("http://%s/%s/%s", mc.endpoint, bucketName, objectName)
	}

	logger.WithComponent("minio").Info().
		Str("bucket", bucketName).
		Str("object", objectName).
		Str("url", url).
		Msg("Upload realizado com sucesso")

	return url, nil
}

// GenerateObjectName gera um nome único para o objeto
func (mc *MinIOClient) GenerateObjectName(prefix, extension string) string {
	timestamp := time.Now().Format("2006/01/02")
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), generateRandomString(8))

	if extension != "" && !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}

	return filepath.Join(prefix, timestamp, filename+extension)
}

// DeleteFile remove um arquivo do MinIO
func (mc *MinIOClient) DeleteFile(ctx context.Context, bucketName, objectName string) error {
	logger.WithComponent("minio").Debug().
		Str("bucket", bucketName).
		Str("object", objectName).
		Msg("Removendo arquivo do MinIO")

	err := mc.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		logger.WithComponent("minio").Error().
			Err(err).
			Str("bucket", bucketName).
			Str("object", objectName).
			Msg("Erro ao remover arquivo do MinIO")
		return fmt.Errorf("failed to delete file: %w", err)
	}

	logger.WithComponent("minio").Info().
		Str("bucket", bucketName).
		Str("object", objectName).
		Msg("Arquivo removido com sucesso")

	return nil
}

// DownloadFile baixa um arquivo do MinIO
func (mc *MinIOClient) DownloadFile(ctx context.Context, bucketName, objectName string) ([]byte, error) {
	logger.WithComponent("minio").Debug().
		Str("bucket", bucketName).
		Str("object", objectName).
		Msg("Baixando arquivo do MinIO")

	// Obtém o objeto
	object, err := mc.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		logger.WithComponent("minio").Error().
			Err(err).
			Str("bucket", bucketName).
			Str("object", objectName).
			Msg("Erro ao obter objeto do MinIO")
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer object.Close()

	// Lê todos os dados
	data, err := io.ReadAll(object)
	if err != nil {
		logger.WithComponent("minio").Error().
			Err(err).
			Str("bucket", bucketName).
			Str("object", objectName).
			Msg("Erro ao ler dados do objeto")
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}

	logger.WithComponent("minio").Info().
		Str("bucket", bucketName).
		Str("object", objectName).
		Int("size", len(data)).
		Msg("Arquivo baixado com sucesso")

	return data, nil
}

// GetFileURL obtém a URL de um arquivo
func (mc *MinIOClient) GetFileURL(bucketName, objectName string) string {
	if mc.useSSL {
		return fmt.Sprintf("https://%s/%s/%s", mc.endpoint, bucketName, objectName)
	}
	return fmt.Sprintf("http://%s/%s/%s", mc.endpoint, bucketName, objectName)
}

// generateRandomString gera uma string aleatória
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}

// InitializeMinIO inicializa o MinIO com buckets padrão
func InitializeMinIO() (*MinIOClient, error) {
	config := MinIOConfig{
		Endpoint:         getEnv("MINIO_ENDPOINT", "localhost:9000"),
		AccessKeyID:      getEnv("MINIO_USER", "wamex"),
		SecretAccessKey:  getEnv("MINIO_PASSWORD", "wamex123456"),
		UseSSL:           getEnv("MINIO_USE_SSL", "false") == "true",
		BucketMedia:      getEnv("MINIO_BUCKET_MEDIA", "wamex-media"),
		BucketTemp:       getEnv("MINIO_BUCKET_TEMP", "wamex-temp"),
		BucketThumbnails: getEnv("MINIO_BUCKET_THUMBNAILS", "wamex-thumbnails"),
	}

	client, err := NewMinIOClient(config)
	if err != nil {
		return nil, err
	}

	// Cria buckets necessários
	ctx := context.Background()

	// Bucket para mídias (público)
	if err := client.CreateBucketIfNotExists(ctx, config.BucketMedia, true); err != nil {
		return nil, fmt.Errorf("failed to create media bucket: %w", err)
	}

	// Bucket para thumbnails (público)
	if err := client.CreateBucketIfNotExists(ctx, config.BucketThumbnails, true); err != nil {
		return nil, fmt.Errorf("failed to create thumbnails bucket: %w", err)
	}

	// Bucket para arquivos temporários (privado)
	if err := client.CreateBucketIfNotExists(ctx, config.BucketTemp, false); err != nil {
		return nil, fmt.Errorf("failed to create temp bucket: %w", err)
	}

	logger.WithComponent("minio").Info().
		Str("media_bucket", config.BucketMedia).
		Str("thumbnails_bucket", config.BucketThumbnails).
		Str("temp_bucket", config.BucketTemp).
		Msg("MinIO inicializado com todos os buckets")

	return client, nil
}

// getEnv obtém variável de ambiente com valor padrão
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
