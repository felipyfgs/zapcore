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
	"zapcore/internal/usecases/message"
	"zapcore/internal/usecases/session"
	"zapcore/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Server representa o servidor HTTP da aplicação
type Server struct {
	config         *config.Config
	httpServer     *http.Server
	logger         *logger.Logger
	db             *database.DB
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

	db, err := database.NewDB(dbConfig, appLogger.GetZerolog())
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar com banco de dados: %w", err)
	}

	// Inicializar store manager do WhatsApp
	storeManager, err := whatsapp.NewStoreManager(db.GetDB(), appLogger.GetZerolog())
	if err != nil {
		return nil, fmt.Errorf("erro ao inicializar store manager do WhatsApp: %w", err)
	}

	// Criar repositórios e cliente WhatsApp (singleton)
	sessionRepo := repository.NewSessionRepository(db.GetDB(), appLogger.GetZerolog())
	eventHandler := whatsapp.NewSessionEventHandler(sessionRepo, appLogger.GetZerolog())
	whatsappClient := whatsapp.NewWhatsAppClient(storeManager.GetContainer(), sessionRepo, appLogger.GetZerolog(), eventHandler)

	server := &Server{
		config:         cfg,
		logger:         appLogger,
		db:             db,
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

// setupRoutes configura todas as rotas da aplicação
func (s *Server) setupRoutes() *gin.Engine {
	// Usar repositórios já criados
	sessionRepo := repository.NewSessionRepository(s.db.GetDB(), s.logger.GetZerolog())

	// Usar cliente WhatsApp singleton
	whatsappClient := s.whatsappClient

	// Criar repositórios de mensagem
	messageRepo := repository.NewMessageRepository(s.db.GetDB(), s.logger.GetZerolog())

	// Criar use cases de sessão
	createUseCase := session.NewCreateUseCase(sessionRepo, s.logger.GetZerolog())
	connectUseCase := session.NewConnectUseCase(sessionRepo, whatsappClient, s.logger.GetZerolog())
	disconnectUseCase := session.NewDisconnectUseCase(sessionRepo, whatsappClient, s.logger.GetZerolog())
	listUseCase := session.NewListUseCase(sessionRepo, s.logger.GetZerolog())
	getStatusUseCase := session.NewGetStatusUseCase(sessionRepo, whatsappClient, s.logger.GetZerolog())

	// Criar use cases de mensagem
	sendTextUseCase := message.NewSendTextUseCase(messageRepo, sessionRepo, whatsappClient, s.logger.GetZerolog())
	sendMediaUseCase := message.NewSendMediaUseCase(messageRepo, sessionRepo, whatsappClient, s.logger.GetZerolog())

	// Criar handlers
	healthHandler := handlers.NewHealthHandler(s.logger.GetZerolog(), "1.0.0")
	sessionHandler := handlers.NewSessionHandler(
		createUseCase,
		connectUseCase,
		disconnectUseCase,
		listUseCase,
		getStatusUseCase,
		s.logger.GetZerolog(),
	)
	messageHandler := handlers.NewMessageHandler(
		sendTextUseCase,
		sendMediaUseCase,
		s.logger.GetZerolog(),
	)

	// Configurar router
	routerConfig := router.Config{
		Logger:          s.logger.GetZerolog(),
		APIKey:          s.config.Auth.APIKey,
		RateLimitReqs:   s.config.RateLimit.Requests,
		RateLimitWindow: s.config.RateLimit.Window.String(),
		CORSOrigins:     s.config.CORS.AllowedOrigins,
		CORSMethods:     s.config.CORS.AllowedMethods,
		CORSHeaders:     s.config.CORS.AllowedHeaders,
	}

	// Criar router com todos os handlers
	r := router.NewRouter(routerConfig, sessionHandler, messageHandler, healthHandler)
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
func (s *Server) GetLogger() *logger.Logger {
	return s.logger
}

// GetDB retorna a instância do banco de dados
func (s *Server) GetDB() *database.DB {
	return s.db
}

// connectActiveSessionsOnStartup reconecta sessões ativas automaticamente
func (s *Server) connectActiveSessionsOnStartup() error {
	// Usar cliente WhatsApp singleton
	ctx := context.Background()
	return s.whatsappClient.ConnectOnStartup(ctx)
}
