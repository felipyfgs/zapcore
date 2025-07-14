package database

import (
	"context"
	"time"

	entity "wamex/internal/domain/entity"

	"github.com/uptrace/bun"
)

// MediaRepository implementa operações de banco de dados para arquivos de mídia
type MediaRepository struct {
	db *bun.DB
}

// NewMediaRepository cria uma nova instância do repositório de mídia
func NewMediaRepository(db *bun.DB) *MediaRepository {
	return &MediaRepository{
		db: db,
	}
}

// Create salva um novo arquivo de mídia no banco de dados
func (r *MediaRepository) Create(ctx context.Context, mediaFile *entity.MediaFile) error {
	// Define timestamps
	now := time.Now()
	mediaFile.CreatedAt = now
	mediaFile.ExpiresAt = now.Add(entity.DefaultMediaTTL)

	_, err := r.db.NewInsert().
		Model(mediaFile).
		Exec(ctx)

	return err
}

// GetByID busca um arquivo de mídia por ID
func (r *MediaRepository) GetByID(ctx context.Context, id string) (*entity.MediaFile, error) {
	mediaFile := &entity.MediaFile{}

	err := r.db.NewSelect().
		Model(mediaFile).
		Where("id = ?", id).
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	return mediaFile, nil
}

// List retorna uma lista paginada de arquivos de mídia
func (r *MediaRepository) List(ctx context.Context, limit, offset int, messageType, sessionID, sessionName string) ([]entity.MediaFile, int, error) {
	var mediaFiles []entity.MediaFile

	query := r.db.NewSelect().
		Model(&mediaFiles).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset)

	// Filtro por tipo de mensagem se fornecido
	if messageType != "" {
		query = query.Where("message_type = ?", messageType)
	}

	// Filtro por ID da sessão se fornecido
	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}

	// Filtro por nome da sessão se fornecido
	if sessionName != "" {
		query = query.Where("session_name = ?", sessionName)
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Conta total de registros
	countQuery := r.db.NewSelect().
		Model((*entity.MediaFile)(nil))

	if messageType != "" {
		countQuery = countQuery.Where("message_type = ?", messageType)
	}

	if sessionID != "" {
		countQuery = countQuery.Where("session_id = ?", sessionID)
	}

	if sessionName != "" {
		countQuery = countQuery.Where("session_name = ?", sessionName)
	}

	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return mediaFiles, total, nil
}

// Delete remove um arquivo de mídia do banco de dados
func (r *MediaRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().
		Model((*entity.MediaFile)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// GetExpiredFiles retorna arquivos que expiraram e devem ser removidos
func (r *MediaRepository) GetExpiredFiles(ctx context.Context) ([]entity.MediaFile, error) {
	var expiredFiles []entity.MediaFile

	err := r.db.NewSelect().
		Model(&expiredFiles).
		Where("expires_at < ?", time.Now()).
		Scan(ctx)

	return expiredFiles, err
}

// UpdateExpiresAt atualiza a data de expiração de um arquivo
func (r *MediaRepository) UpdateExpiresAt(ctx context.Context, id string, expiresAt time.Time) error {
	_, err := r.db.NewUpdate().
		Model((*entity.MediaFile)(nil)).
		Set("expires_at = ?", expiresAt).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// GetByMessageType retorna arquivos filtrados por tipo de mensagem
func (r *MediaRepository) GetByMessageType(ctx context.Context, messageType string, limit int) ([]entity.MediaFile, error) {
	var mediaFiles []entity.MediaFile

	err := r.db.NewSelect().
		Model(&mediaFiles).
		Where("message_type = ?", messageType).
		Order("created_at DESC").
		Limit(limit).
		Scan(ctx)

	return mediaFiles, err
}

// GetStats retorna estatísticas de uso de mídia
func (r *MediaRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total de arquivos
	totalFiles, err := r.db.NewSelect().
		Model((*entity.MediaFile)(nil)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats["total_files"] = totalFiles

	// Total de tamanho em bytes
	var totalSize int64
	err = r.db.NewSelect().
		Model((*entity.MediaFile)(nil)).
		ColumnExpr("COALESCE(SUM(size), 0)").
		Scan(ctx, &totalSize)
	if err != nil {
		return nil, err
	}
	stats["total_size_bytes"] = totalSize

	// Arquivos por tipo
	type TypeCount struct {
		MessageType string `bun:"message_type"`
		Count       int    `bun:"count"`
	}

	var typeCounts []TypeCount
	err = r.db.NewSelect().
		Model((*entity.MediaFile)(nil)).
		Column("message_type").
		ColumnExpr("COUNT(*) as count").
		Group("message_type").
		Scan(ctx, &typeCounts)
	if err != nil {
		return nil, err
	}

	typeStats := make(map[string]int)
	for _, tc := range typeCounts {
		typeStats[tc.MessageType] = tc.Count
	}
	stats["files_by_type"] = typeStats

	// Arquivos criados nas últimas 24 horas
	recentFiles, err := r.db.NewSelect().
		Model((*entity.MediaFile)(nil)).
		Where("created_at > ?", time.Now().Add(-24*time.Hour)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats["recent_files_24h"] = recentFiles

	return stats, nil
}

// CleanupExpired remove arquivos expirados do banco de dados
func (r *MediaRepository) CleanupExpired(ctx context.Context) (int, error) {
	result, err := r.db.NewDelete().
		Model((*entity.MediaFile)(nil)).
		Where("expires_at < ?", time.Now()).
		Exec(ctx)

	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}
