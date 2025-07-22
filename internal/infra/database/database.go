package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"zapcore/pkg/logger"

	_ "github.com/lib/pq"
)

// Config representa a configuração do banco de dados
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DB representa a conexão com o banco de dados
type DB struct {
	db     *sql.DB
	config *Config
	logger *logger.Logger
}

// NewDB cria uma nova conexão com o banco de dados
func NewDB(config *Config) (*DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.DBName,
		config.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir conexão com banco de dados: %w", err)
	}

	// Configurar pool de conexões
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Testar conexão
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("erro ao conectar com banco de dados: %w", err)
	}

	logger.Info().Msg("Conexão com banco de dados estabelecida com sucesso")

	return &DB{
		db:     db,
		config: config,
		logger: logger.Get(),
	}, nil
}

// GetDB retorna a instância do sql.DB
func (d *DB) GetDB() *sql.DB {
	return d.db
}

// Ping testa a conexão com o banco
func (d *DB) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return d.db.PingContext(ctx)
}

// Close fecha a conexão com o banco
func (d *DB) Close() error {
	if d.db != nil {
		d.logger.Info().Msg("Fechando conexão com banco de dados")
		return d.db.Close()
	}
	return nil
}

// Stats retorna estatísticas da conexão
func (d *DB) Stats() sql.DBStats {
	return d.db.Stats()
}

// BeginTx inicia uma transação
func (d *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return d.db.BeginTx(ctx, opts)
}

// GetConfig retorna a configuração do banco
func (d *DB) GetConfig() *Config {
	return d.config
}

// GetLogger retorna o logger
func (d *DB) GetLogger() *logger.Logger {
	return d.logger
}

// Health verifica a saúde da conexão
func (d *DB) Health() map[string]interface{} {
	stats := d.db.Stats()

	health := map[string]interface{}{
		"status": "healthy",
		"stats": map[string]interface{}{
			"max_open_connections": stats.MaxOpenConnections,
			"open_connections":     stats.OpenConnections,
			"in_use":               stats.InUse,
			"idle":                 stats.Idle,
			"wait_count":           stats.WaitCount,
			"wait_duration":        stats.WaitDuration.String(),
			"max_idle_closed":      stats.MaxIdleClosed,
			"max_idle_time_closed": stats.MaxIdleTimeClosed,
			"max_lifetime_closed":  stats.MaxLifetimeClosed,
		},
	}

	// Testar conexão
	if err := d.Ping(); err != nil {
		health["status"] = "unhealthy"
		health["error"] = err.Error()
	}

	return health
}
