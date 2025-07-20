package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"zapcore/pkg/logger"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
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
func NewDB(config *Config, zeroLogger zerolog.Logger) (*DB, error) {
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
		logger: logger.NewFromZerolog(zeroLogger),
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

// ExecContext executa uma query sem retorno
func (d *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return d.db.ExecContext(ctx, query, args...)
}

// QueryContext executa uma query com retorno
func (d *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return d.db.QueryContext(ctx, query, args...)
}

// QueryRowContext executa uma query que retorna uma única linha
func (d *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return d.db.QueryRowContext(ctx, query, args...)
}

// PrepareContext prepara uma statement
func (d *DB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return d.db.PrepareContext(ctx, query)
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
