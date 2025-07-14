package infra

import (
	"context"
	"fmt"

	"wamex/configs"
	"wamex/internal/repository"
	"wamex/internal/service"
	"wamex/pkg/logger"
)

// Infrastructure representa toda a infraestrutura da aplicação
type Infrastructure struct {
	Database *DatabaseConnection
	Storage  *StorageConnection

	// Repositories
	SessionRepo *repository.SessionRepository
	MediaRepo   *repository.MediaRepository

	// Services
	WhatsAppService *service.WhatsAppService
	UnifiedService  *service.UnifiedMediaService
}

// NewInfrastructure cria uma nova instância da infraestrutura
func NewInfrastructure(cfg *configs.Config) (*Infrastructure, error) {
	logger.WithComponent("infra").Info().
		Msg("🏗️ Initializing infrastructure...")

	// Inicializar conexão com banco de dados
	dbConn, err := NewDatabaseConnection(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Inicializar conexão com storage
	storageConn, err := NewStorageConnection()
	if err != nil {
		dbConn.Close()
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Inicializar repositories
	sessionRepo := repository.NewSessionRepository(dbConn.GetDB())
	mediaRepo := repository.NewMediaRepository(dbConn.GetDB())

	// Inicializar services
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	whatsappService, err := service.NewWhatsAppService(
		sessionRepo,
		"postgres",
		dsn,
	)
	if err != nil {
		dbConn.Close()
		storageConn.Close()
		return nil, fmt.Errorf("failed to initialize WhatsApp service: %w", err)
	}

	unifiedService := service.NewUnifiedMediaService(mediaRepo, storageConn.GetMinIOClient())

	logger.WithComponent("infra").Info().
		Msg("✅ Infrastructure initialized successfully")

	return &Infrastructure{
		Database:        dbConn,
		Storage:         storageConn,
		SessionRepo:     sessionRepo,
		MediaRepo:       mediaRepo,
		WhatsAppService: whatsappService,
		UnifiedService:  unifiedService,
	}, nil
}

// Close fecha todas as conexões da infraestrutura
func (i *Infrastructure) Close() error {
	logger.WithComponent("infra").Info().
		Msg("🔄 Closing infrastructure connections...")

	var errors []error

	// Fechar conexão com storage
	if err := i.Storage.Close(); err != nil {
		errors = append(errors, fmt.Errorf("storage close error: %w", err))
	}

	// Fechar conexão com banco de dados
	if err := i.Database.Close(); err != nil {
		errors = append(errors, fmt.Errorf("database close error: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("infrastructure close errors: %v", errors)
	}

	logger.WithComponent("infra").Info().
		Msg("✅ Infrastructure closed successfully")

	return nil
}

// HealthCheck verifica a saúde de toda a infraestrutura
func (i *Infrastructure) HealthCheck(ctx context.Context) error {
	logger.WithComponent("infra").Debug().
		Msg("🔍 Running infrastructure health check...")

	// Verificar banco de dados
	if err := i.Database.HealthCheck(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	// Verificar storage
	if err := i.Storage.HealthCheck(ctx); err != nil {
		return fmt.Errorf("storage health check failed: %w", err)
	}

	// Verificar unified service
	if err := i.UnifiedService.HealthCheck(ctx); err != nil {
		return fmt.Errorf("unified service health check failed: %w", err)
	}

	logger.WithComponent("infra").Debug().
		Msg("✅ Infrastructure health check passed")

	return nil
}

// GetDatabaseStats retorna estatísticas do banco de dados
func (i *Infrastructure) GetDatabaseStats() map[string]interface{} {
	stats := i.Database.GetStats()
	return map[string]interface{}{
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}
}
