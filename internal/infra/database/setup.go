package database

import (
	"context"
	"fmt"

	configs "wamex/internal/infra/config"
	"wamex/pkg/logger"
)

// Infrastructure representa toda a infraestrutura da aplicação
type Infrastructure struct {
	Database *DatabaseConnection

	// Repositories
	SessionRepo *SessionRepository
	MediaRepo   *MediaRepository
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

	// Storage será inicializado na próxima tarefa

	// Inicializar repositories
	sessionRepo := NewSessionRepository(dbConn.GetDB())
	mediaRepo := NewMediaRepository(dbConn.GetDB())

	// Database connection initialized successfully

	logger.WithComponent("infra").Info().
		Msg("✅ Infrastructure initialized successfully")

	return &Infrastructure{
		Database:    dbConn,
		SessionRepo: sessionRepo,
		MediaRepo:   mediaRepo,
	}, nil
}

// Close fecha todas as conexões da infraestrutura
func (i *Infrastructure) Close() error {
	logger.WithComponent("infra").Info().
		Msg("🔄 Closing infrastructure connections...")

	var errors []error

	// Storage será fechado na próxima tarefa

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

	// Storage health check será implementado na próxima tarefa

	// Unified service health check será implementado na próxima tarefa

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
