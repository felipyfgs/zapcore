package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"zapcore/internal/domain/chat"
	"zapcore/internal/domain/contact"
	"zapcore/internal/domain/message"
	"zapcore/internal/domain/session"
	"zapcore/pkg/logger"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// BunLoggerAdapter adapta nosso logger centralizado para o formato do Bun
type BunLoggerAdapter struct {
	logger *logger.Logger
}

// NewBunLoggerAdapter cria um novo adapter de logger para o Bun
func NewBunLoggerAdapter() *BunLoggerAdapter {
	return &BunLoggerAdapter{
		logger: logger.Get().WithField("component", "bun"),
	}
}

// Printf implementa a interface do Bun logger
func (b *BunLoggerAdapter) Printf(format string, v ...interface{}) {
	// Extrair informações da query se possível
	msg := fmt.Sprintf(format, v...)

	// Log estruturado para queries SQL
	b.logger.Debug().
		Str("type", "sql_query").
		Msg(msg)
}

// Write implementa io.Writer para integração com bundebug
func (b *BunLoggerAdapter) Write(p []byte) (n int, err error) {
	// Converter bytes para string e logar
	msg := string(p)

	// Log estruturado para queries SQL
	b.logger.Debug().
		Str("type", "sql_query").
		Msg(msg)

	return len(p), nil
}

// BunQueryHook é nosso hook personalizado para capturar queries do Bun
type BunQueryHook struct {
	logger *BunLoggerAdapter
}

// BeforeQuery é chamado antes da execução da query
func (h *BunQueryHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	return ctx
}

// AfterQuery é chamado após a execução da query
func (h *BunQueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	duration := time.Since(event.StartTime)

	h.logger.logger.Debug().
		Str("type", "sql_query").
		Str("operation", event.Operation()).
		Dur("duration", duration).
		Str("query", event.Query).
		Msg("SQL Query executed")
}

// BunDB representa a conexão com o banco de dados usando Bun ORM
type BunDB struct {
	db     *bun.DB
	config *Config
	logger *logger.Logger
}

// SilentLogger implementa a interface logging do pgdriver mas não faz nada
type SilentLogger struct{}

func (s *SilentLogger) Printf(ctx context.Context, format string, v ...interface{}) {
	// Não faz nada - silencia os logs
}

// NewBunDB cria uma nova conexão com o banco de dados usando Bun ORM
func NewBunDB(config *Config) (*BunDB, error) {
	// Desabilitar logs do pgdriver completamente
	pgdriver.Logger = &SilentLogger{}

	// Criar DSN para PostgreSQL
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.DBName,
		config.SSLMode,
	)

	// Criar conexão SQL
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	// Configurar pool de conexões
	sqldb.SetMaxOpenConns(config.MaxOpenConns)
	sqldb.SetMaxIdleConns(config.MaxIdleConns)
	sqldb.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Criar instância Bun com configuração para PostgreSQL
	dialect := pgdialect.New()
	db := bun.NewDB(sqldb, dialect)

	// Manter o filtro ativo permanentemente para interceptar logs do Bun

	// NÃO adicionar nenhum hook de query para manter logs silenciosos
	// Os logs do Bun vêm do bundebug que não está sendo adicionado

	// Adicionar nosso logger centralizado apenas se necessário para desenvolvimento
	if config.Host == "localhost" || config.Host == "127.0.0.1" {
		bunLogger := NewBunLoggerAdapter()
		// Adicionar nosso próprio hook de query
		db.AddQueryHook(&BunQueryHook{logger: bunLogger})
	}

	// Testar conexão
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("erro ao conectar com banco de dados: %w", err)
	}

	bunDB := &BunDB{
		db:     db,
		config: config,
		logger: logger.Get(),
	}

	// Executar auto-migration
	if err := bunDB.AutoMigrate(ctx); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("erro ao executar auto-migration: %w", err)
	}

	bunDB.logger.Info().Msg("Conexão com banco de dados Bun estabelecida com sucesso")
	return bunDB, nil
}

// GetDB retorna a instância do Bun DB
func (d *BunDB) GetDB() *bun.DB {
	return d.db
}

// GetSQLDB retorna a instância do sql.DB subjacente
func (d *BunDB) GetSQLDB() *sql.DB {
	return d.db.DB
}

// Ping testa a conexão com o banco
func (d *BunDB) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return d.db.PingContext(ctx)
}

// Close fecha a conexão com o banco
func (d *BunDB) Close() error {
	if d.db != nil {
		d.logger.Info().Msg("Fechando conexão com banco de dados Bun")
		return d.db.Close()
	}
	return nil
}

// Stats retorna estatísticas da conexão
func (d *BunDB) Stats() sql.DBStats {
	return d.db.DB.Stats()
}

// BeginTx inicia uma transação
func (d *BunDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (bun.Tx, error) {
	return d.db.BeginTx(ctx, opts)
}

// AutoMigrate executa as migrations automáticas baseadas nas structs
func (d *BunDB) AutoMigrate(ctx context.Context) error {
	d.logger.Info().Msg("Iniciando auto-migration do banco de dados")

	// Registrar modelos
	models := []interface{}{
		(*session.Session)(nil),
		(*message.Message)(nil),
		(*chat.Chat)(nil),
		(*contact.Contact)(nil),
	}

	// Criar tabelas para cada modelo usando apenas Bun ORM
	for _, model := range models {
		if _, err := d.db.NewCreateTable().Model(model).IfNotExists().Exec(ctx); err != nil {
			return fmt.Errorf("erro ao criar tabela para modelo %T: %w", model, err)
		}
	}

	d.logger.Info().Msg("Auto-migration concluída com sucesso")
	return nil
}

// GetConfig retorna a configuração do banco
func (d *BunDB) GetConfig() *Config {
	return d.config
}

// GetLogger retorna o logger
func (d *BunDB) GetLogger() *logger.Logger {
	return d.logger
}
