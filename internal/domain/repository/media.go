package domain

import (
	"context"
	"time"
	entity "wamex/internal/domain/entity"
)

// MediaRepository define a interface para operações de mídia no banco de dados
type MediaRepository interface {
	Create(ctx context.Context, mediaFile *entity.MediaFile) error
	GetByID(ctx context.Context, id string) (*entity.MediaFile, error)
	List(ctx context.Context, limit, offset int, messageType, sessionID, sessionName string) ([]entity.MediaFile, int, error)
	Delete(ctx context.Context, id string) error
	GetExpiredFiles(ctx context.Context) ([]entity.MediaFile, error)
	UpdateExpiresAt(ctx context.Context, id string, expiresAt time.Time) error
	GetByMessageType(ctx context.Context, messageType string, limit int) ([]entity.MediaFile, error)
	GetStats(ctx context.Context) (map[string]interface{}, error)
	CleanupExpired(ctx context.Context) (int, error)
}
