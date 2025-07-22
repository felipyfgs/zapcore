package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"zapcore/internal/app/config"
	"zapcore/internal/domain/chat"
	"zapcore/internal/domain/contact"
	"zapcore/internal/domain/message"
	"zapcore/internal/domain/session"
	"zapcore/internal/http/handlers"
	"zapcore/internal/http/router"
	"zapcore/internal/infra/database"
	"zapcore/internal/infra/repository"
	"zapcore/internal/infra/storage"
	"zapcore/internal/infra/whatsapp"
	messageUseCase "zapcore/internal/usecases/message"
	sessionUseCase "zapcore/internal/usecases/session"
	"zapcore/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// BunDB representa a conexão com o banco de dados usando Bun ORM
type BunDB struct {
	db     *bun.DB
	config *database.Config
	logger *logger.Logger
}

// NewBunDB cria uma nova conexão com o banco de dados usando Bun ORM
func NewBunDB(cfg *config.Config) (*BunDB, error) {
	// Converter configuração para formato do database
	port, err := strconv.Atoi(cfg.Database.Port)
	if err != nil {
		return nil, fmt.Errorf("erro ao converter porta do banco: %w", err)
	}

	// Criar DSN para PostgreSQL
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	// Criar conexão SQL
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	// Configurar pool de conexões
	sqldb.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqldb.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqldb.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// Criar instância Bun
	db := bun.NewDB(sqldb, pgdialect.New())

	// Logs do Bun desabilitados - usando logger centralizado

	// Testar conexão
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("erro ao conectar com banco de dados: %w", err)
	}

	bunDB := &BunDB{
		db:     db,
		logger: logger.Get(),
	}

	// Executar auto-migration
	if err := bunDB.AutoMigrate(ctx); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("erro ao executar auto-migration: %w", err)
	}

	logger.WithFields(map[string]interface{}{
		"component": "database",
		"driver":    "postgresql",
		"status":    "connected",
	}).Info().Msg("🗄️ Conexão PostgreSQL OK")
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
		d.logger.Info().Msg("Fechando banco Bun")
		return d.db.Close()
	}
	return nil
}

// AutoMigrate executa as migrations automáticas baseadas nas structs
func (d *BunDB) AutoMigrate(ctx context.Context) error {
	d.logger.WithFields(map[string]interface{}{
		"component": "database",
		"operation": "migration",
	}).Info().Msg("🔄 Executando auto-migration")

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

	d.logger.WithFields(map[string]interface{}{
		"component": "database",
		"operation": "migration",
		"status":    "completed",
	}).Info().Msg("✅ Auto-migration concluída")
	return nil
}

// Server representa o servidor HTTP da aplicação
type Server struct {
	config         *config.Config
	httpServer     *http.Server
	logger         *logger.Logger
	bunDB          *BunDB
	storeManager   *whatsapp.StoreManager
	whatsappClient *whatsapp.WhatsAppClient // Singleton instance
}

// New cria uma nova instância do servidor
func New(cfg *config.Config) (*Server, error) {
	// Usar logger global já inicializado
	appLogger := logger.Get()

	// Configurar modo do Gin
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Conectar ao banco de dados usando Bun ORM
	bunDB, err := NewBunDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar com banco de dados Bun: %w", err)
	}

	// Inicializar store manager do WhatsApp
	storeManager, err := whatsapp.NewStoreManager(bunDB.GetSQLDB(), appLogger.GetZerolog())
	if err != nil {
		return nil, fmt.Errorf("erro ao inicializar store manager do WhatsApp: %w", err)
	}

	// Criar repositórios Bun
	sessionRepo := repository.NewSessionRepository(bunDB.GetDB())
	messageRepo := repository.NewMessageRepository(bunDB.GetDB())
	chatRepo := repository.NewChatRepository(bunDB.GetDB())
	contactRepo := repository.NewContactRepository(bunDB.GetDB())

	// Criar cliente MinIO se habilitado
	var minioClient *storage.MinIOClient
	if cfg.MinIO.Enabled {
		var err error
		minioClient, err = storage.NewMinIOClient(&cfg.MinIO)
		if err != nil {
			return nil, fmt.Errorf("erro ao criar cliente MinIO: %w", err)
		}
	}

	// Criar handlers de eventos (MediaDownloader será configurado dinamicamente)
	sessionHandler := whatsapp.NewSessionEventHandler(sessionRepo)
	storageHandler := whatsapp.NewStorageHandler(messageRepo, chatRepo, contactRepo, nil)
	compositeHandler := whatsapp.NewCompositeEventHandler(sessionHandler, storageHandler)

	// Criar cliente WhatsApp (singleton)
	whatsappClient := whatsapp.NewWhatsAppClient(storeManager.GetContainer(), sessionRepo, compositeHandler, minioClient)

	server := &Server{
		config:         cfg,
		logger:         appLogger,
		bunDB:          bunDB,
		storeManager:   storeManager,
		whatsappClient: whatsappClient,
	}

	// Configurar rotas
	engine := server.setupRoutes()

	// Configurar servidor HTTP
	server.httpServer = &http.Server{
		Addr:         cfg.GetServerAddress(),
		Handler:      engine,
		ReadTimeout:  cfg.Timeout.Request,
		WriteTimeout: cfg.Timeout.Request,
		IdleTimeout:  cfg.Timeout.Request * 2,
	}

	return server, nil
}

// setupRoutes configura todas as rotas da aplicação usando o router completo
func (s *Server) setupRoutes() *gin.Engine {
	// Criar repositórios
	sessionRepo := repository.NewSessionRepository(s.bunDB.GetDB())
	messageRepo := repository.NewMessageRepository(s.bunDB.GetDB())

	// Criar use cases
	sendTextUseCase := messageUseCase.NewSendTextUseCase(messageRepo, sessionRepo, s.whatsappClient)
	sendMediaUseCase := messageUseCase.NewSendMediaUseCase(messageRepo, sessionRepo, s.whatsappClient)

	createSessionUseCase := sessionUseCase.NewCreateUseCase(sessionRepo)
	connectSessionUseCase := sessionUseCase.NewConnectUseCase(sessionRepo, s.whatsappClient)
	disconnectSessionUseCase := sessionUseCase.NewDisconnectUseCase(sessionRepo, s.whatsappClient)
	listSessionUseCase := sessionUseCase.NewListUseCase(sessionRepo)
	getStatusSessionUseCase := sessionUseCase.NewGetStatusUseCase(sessionRepo, s.whatsappClient)

	// Criar handlers
	messageHandler := handlers.NewMessageHandler(sendTextUseCase, sendMediaUseCase)
	sessionHandler := handlers.NewSessionHandler(
		createSessionUseCase,
		connectSessionUseCase,
		disconnectSessionUseCase,
		listSessionUseCase,
		getStatusSessionUseCase,
	)
	healthHandler := handlers.NewHealthHandler("1.0.0")

	// Configurar router
	routerConfig := router.Config{
		APIKey:          s.config.Auth.APIKey,
		RateLimitReqs:   s.config.RateLimit.Requests,
		RateLimitWindow: s.config.RateLimit.Window.String(),
		CORSOrigins:     s.config.CORS.AllowedOrigins,
		CORSMethods:     s.config.CORS.AllowedMethods,
		CORSHeaders:     s.config.CORS.AllowedHeaders,
	}

	appRouter := router.NewRouter(routerConfig, sessionHandler, messageHandler, healthHandler)
	return appRouter.Setup()
}

// Start inicia o servidor HTTP
func (s *Server) Start() error {
	s.logger.WithFields(map[string]interface{}{
		"component": "server",
		"protocol":  "http",
		"address":   s.config.GetServerAddress(),
		"env":       s.config.Server.Env,
	}).Info().Msg("🌐 Iniciando HTTP server")

	// Reconectar sessões ativas automaticamente
	if err := s.connectActiveSessionsOnStartup(); err != nil {
		s.logger.Error().Err(err).Msg("Erro ao reconectar sessões ativas")
		// Não retornar erro aqui para não impedir o servidor de iniciar
	}

	// Canal para capturar sinais do sistema
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Iniciar servidor em goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal().Err(err).Msg("Erro ao iniciar servidor HTTP")
		}
	}()

	s.logger.WithFields(map[string]interface{}{
		"component": "server",
		"protocol":  "http",
		"address":   s.config.GetServerAddress(),
		"env":       s.config.Server.Env,
		"status":    "running",
	}).Info().Msg("🚀 Server iniciado!")

	// Aguardar sinal de parada
	<-quit
	s.logger.Info().Msg("Iniciando shutdown graceful...")

	return s.Stop()
}

// Stop para o servidor graciosamente
func (s *Server) Stop() error {
	// Criar contexto com timeout para shutdown
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout.Shutdown)
	defer cancel()

	s.logger.Info().
		Dur("timeout", s.config.Timeout.Shutdown).
		Msg("Shutdown HTTP server")

	// Parar servidor HTTP
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error().Err(err).Msg("Erro shutdown HTTP")
		return err
	}

	// Fechar store manager do WhatsApp
	if s.storeManager != nil {
		if err := s.storeManager.Close(); err != nil {
			s.logger.Error().Err(err).Msg("Erro fechar WhatsApp store")
		}
	}

	// Fechar conexão com banco de dados Bun
	if s.bunDB != nil {
		if err := s.bunDB.Close(); err != nil {
			s.logger.Error().Err(err).Msg("Erro fechar banco")
		}
	}

	s.logger.Info().Msg("Server parado")
	return nil
}

// Health retorna o status de saúde do servidor
func (s *Server) Health() map[string]interface{} {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
		"env":       s.config.Server.Env,
	}

	// Verificar conexão com banco de dados Bun
	if s.bunDB != nil {
		if err := s.bunDB.Ping(); err != nil {
			health["database"] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
			health["status"] = "degraded"
		} else {
			health["database"] = map[string]interface{}{
				"status": "healthy",
			}
		}
	}

	return health
}

// GetConfig retorna a configuração do servidor
func (s *Server) GetConfig() *config.Config {
	return s.config
}

// GetLogger retorna o logger do servidor
func (s *Server) GetLogger() *logger.Logger {
	return s.logger
}

// GetBunDB retorna a instância do banco de dados Bun
func (s *Server) GetBunDB() *BunDB {
	return s.bunDB
}

// connectActiveSessionsOnStartup reconecta sessões ativas automaticamente
func (s *Server) connectActiveSessionsOnStartup() error {
	if s.whatsappClient == nil {
		s.logger.Info().Msg("WhatsApp client não inicializado, pulando reconexão automática")
		return nil
	}

	ctx := context.Background()
	return s.whatsappClient.ConnectOnStartup(ctx)
}
