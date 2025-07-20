package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"zapcore/internal/app/config"
	"zapcore/internal/http/handlers"
	"zapcore/internal/http/router"
	"zapcore/internal/infra/database"
	"zapcore/internal/infra/repository"
	"zapcore/internal/infra/whatsapp"
	"zapcore/internal/usecases/session"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Server representa o servidor HTTP da aplicação
type Server struct {
	config       *config.Config
	httpServer   *http.Server
	logger       zerolog.Logger
	db           *database.DB
	storeManager *whatsapp.StoreManager
}

// New cria uma nova instância do servidor
func New(cfg *config.Config) (*Server, error) {
	// Configurar logger
	logger := setupLogger(cfg)

	// Configurar modo do Gin
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Conectar ao banco de dados
	dbConfig := &database.Config{
		Host:            cfg.Database.Host,
		Port:            5432, // Converter string para int
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		DBName:          cfg.Database.Name,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	}

	db, err := database.NewDB(dbConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar com banco de dados: %w", err)
	}

	// Inicializar store manager do WhatsApp
	storeManager, err := whatsapp.NewStoreManager(db.GetDB(), logger)
	if err != nil {
		return nil, fmt.Errorf("erro ao inicializar store manager do WhatsApp: %w", err)
	}

	server := &Server{
		config:       cfg,
		logger:       logger,
		db:           db,
		storeManager: storeManager,
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

// setupLogger configura o logger baseado na configuração
func setupLogger(cfg *config.Config) zerolog.Logger {
	// Configurar nível de log
	level, err := zerolog.ParseLevel(cfg.Log.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Configurar formato
	var logger zerolog.Logger
	if cfg.Log.Format == "console" || cfg.IsDevelopment() {
		// Formato console para desenvolvimento
		logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}).With().Timestamp().Logger()
	} else {
		// Formato JSON para produção
		logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}

	return logger
}

// setupRoutes configura todas as rotas da aplicação
func (s *Server) setupRoutes() *gin.Engine {
	// Criar repositórios
	sessionRepo := repository.NewSessionRepository(s.db.GetDB(), s.logger)

	// Criar event handler para WhatsApp
	eventHandler := whatsapp.NewSessionEventHandler(sessionRepo, s.logger)

	// Criar cliente WhatsApp
	whatsappClient := whatsapp.NewWhatsAppClient(s.storeManager.GetContainer(), sessionRepo, s.logger, eventHandler)

	// Criar use cases
	createUseCase := session.NewCreateUseCase(sessionRepo, s.logger)
	connectUseCase := session.NewConnectUseCase(sessionRepo, whatsappClient, s.logger)
	disconnectUseCase := session.NewDisconnectUseCase(sessionRepo, whatsappClient, s.logger)
	listUseCase := session.NewListUseCase(sessionRepo, s.logger)
	getStatusUseCase := session.NewGetStatusUseCase(sessionRepo, whatsappClient, s.logger)

	// Criar handlers
	healthHandler := handlers.NewHealthHandler(s.logger, "1.0.0")
	sessionHandler := handlers.NewSessionHandler(
		createUseCase,
		connectUseCase,
		disconnectUseCase,
		listUseCase,
		getStatusUseCase,
		s.logger,
	)

	// Configurar router
	routerConfig := router.Config{
		Logger:          s.logger,
		APIKey:          s.config.Auth.APIKey,
		RateLimitReqs:   s.config.RateLimit.Requests,
		RateLimitWindow: s.config.RateLimit.Window.String(),
		CORSOrigins:     s.config.CORS.AllowedOrigins,
		CORSMethods:     s.config.CORS.AllowedMethods,
		CORSHeaders:     s.config.CORS.AllowedHeaders,
	}

	// Criar router com todos os handlers
	r := router.NewRouter(routerConfig, sessionHandler, healthHandler)
	return r.Setup()
}

// Start inicia o servidor HTTP
func (s *Server) Start() error {
	s.logger.Info().
		Str("address", s.config.GetServerAddress()).
		Str("env", s.config.Server.Env).
		Msg("Iniciando servidor HTTP")

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

	s.logger.Info().
		Str("address", s.config.GetServerAddress()).
		Msg("Servidor HTTP iniciado com sucesso")

	// Aguardar sinal de parada
	<-quit
	s.logger.Info().Msg("Recebido sinal de parada, iniciando shutdown graceful...")

	return s.Stop()
}

// Stop para o servidor graciosamente
func (s *Server) Stop() error {
	// Criar contexto com timeout para shutdown
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout.Shutdown)
	defer cancel()

	s.logger.Info().
		Dur("timeout", s.config.Timeout.Shutdown).
		Msg("Iniciando shutdown do servidor HTTP")

	// Parar servidor HTTP
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error().Err(err).Msg("Erro durante shutdown do servidor HTTP")
		return err
	}

	// Fechar store manager do WhatsApp
	if s.storeManager != nil {
		if err := s.storeManager.Close(); err != nil {
			s.logger.Error().Err(err).Msg("Erro ao fechar store manager do WhatsApp")
		}
	}

	// Fechar conexão com banco de dados
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.Error().Err(err).Msg("Erro ao fechar conexão com banco de dados")
		}
	}

	s.logger.Info().Msg("Servidor parado com sucesso")
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

	// Verificar conexão com banco de dados
	if s.db != nil {
		if err := s.db.Ping(); err != nil {
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
func (s *Server) GetLogger() zerolog.Logger {
	return s.logger
}

// GetDB retorna a instância do banco de dados
func (s *Server) GetDB() *database.DB {
	return s.db
}

// connectActiveSessionsOnStartup reconecta sessões ativas automaticamente
func (s *Server) connectActiveSessionsOnStartup() error {
	// Criar repositórios e cliente WhatsApp
	sessionRepo := repository.NewSessionRepository(s.db.GetDB(), s.logger)
	eventHandler := whatsapp.NewSessionEventHandler(sessionRepo, s.logger)
	whatsappClient := whatsapp.NewWhatsAppClient(s.storeManager.GetContainer(), sessionRepo, s.logger, eventHandler)

	// Chamar ConnectOnStartup
	ctx := context.Background()
	return whatsappClient.ConnectOnStartup(ctx)
}
