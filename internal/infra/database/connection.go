package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	configs "wamex/internal/infra/config"
	"wamex/pkg/logger"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// DatabaseConfig representa a configuração do banco de dados
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// DatabaseConnection representa uma conexão com o banco de dados
type DatabaseConnection struct {
	DB     *bun.DB
	config *DatabaseConfig
}

// NewDatabaseConnection cria uma nova conexão com o banco de dados
func NewDatabaseConnection(cfg *configs.Config) (*DatabaseConnection, error) {
	dbConfig := &DatabaseConfig{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Name:     cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
	}

	dsn := buildDSN(dbConfig)

	connector := pgdriver.NewConnector(
		pgdriver.WithDSN(dsn),
		pgdriver.WithTimeout(30*time.Second),
	)

	sqldb := sql.OpenDB(connector)
	db := bun.NewDB(sqldb, pgdialect.New())

	// Testar conexão
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.WithComponent("database").Info().
		Str("host", dbConfig.Host).
		Str("port", dbConfig.Port).
		Str("database", dbConfig.Name).
		Msg("📊 Database connected successfully")

	return &DatabaseConnection{
		DB:     db,
		config: dbConfig,
	}, nil
}

// Close fecha a conexão com o banco de dados
func (dc *DatabaseConnection) Close() error {
	if dc.DB != nil {
		return dc.DB.Close()
	}
	return nil
}

// GetDB retorna a instância do banco de dados
func (dc *DatabaseConnection) GetDB() *bun.DB {
	return dc.DB
}

// HealthCheck verifica se a conexão está saudável
func (dc *DatabaseConnection) HealthCheck(ctx context.Context) error {
	return dc.DB.PingContext(ctx)
}

// GetStats retorna estatísticas da conexão
func (dc *DatabaseConnection) GetStats() sql.DBStats {
	return dc.DB.DB.Stats()
}

// buildDSN constrói a string de conexão do PostgreSQL
func buildDSN(cfg *DatabaseConfig) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.SSLMode,
	)
}
