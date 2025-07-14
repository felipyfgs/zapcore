package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	configs "wamex/internal/infra/config"
	"wamex/internal/infra/database"
	"wamex/internal/infra/storage"
	"wamex/internal/infra/whatsapp"
	"wamex/internal/transport/http/handler"
	"wamex/internal/transport/http/router"
	"wamex/pkg/logger"
)

// App representa a aplicação
type App struct {
	config *configs.Config
	server *http.Server
}

// New cria uma nova instância da aplicação
func New() (*App, error) {
	// Carregar configurações
	cfg, err := configs.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &App{
		config: cfg,
	}, nil
}

// Run executa a aplicação
func (a *App) Run() error {
	// Inicializar logger
	logger.InitFromEnv()
	logger.GetGlobal().Info().Msg("🚀 Starting WAMEX server...")

	// Inicializar banco de dados
	dbConn, err := database.NewDatabaseConnection(a.config)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer dbConn.Close()
	db := dbConn.GetDB()

	// Inicializar MinIO
	minioClient, err := storage.InitializeMinIO()
	if err != nil {
		return fmt.Errorf("failed to initialize MinIO: %w", err)
	}

	// Inicializar repositórios
	sessionRepo := database.NewSessionRepository(db)
	mediaRepo := database.NewMediaRepository(db)

	// Inicializar services
	whatsappService, err := whatsapp.NewWhatsAppService(
		sessionRepo,
		"postgres",
		a.buildDSN(a.config.Database),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize WhatsApp service: %w", err)
	}

	// Unified media service disponível para uso futuro
	// _ = media.NewUnifiedMediaService(mediaRepo, minioClient)

	// Inicializar handlers
	sessionHandler := handler.NewSessionHandler(whatsappService, mediaRepo)
	messageHandler := handler.NewMessageHandler(whatsappService, mediaRepo)
	mediaHandler := handler.NewMediaHandler(mediaRepo, minioClient)

	// Configurar rotas
	httpRouter := router.SetupRoutes(sessionHandler, messageHandler, mediaHandler, whatsappService)

	// Configurar servidor HTTP
	a.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%s", a.config.Server.Host, a.config.Server.Port),
		Handler:      httpRouter,
		ReadTimeout:  time.Duration(a.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(a.config.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(a.config.Server.IdleTimeout) * time.Second,
	}

	// Canal para capturar sinais do sistema
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Iniciar servidor em goroutine
	go func() {
		logger.GetGlobal().Info().
			Str("host", a.config.Server.Host).
			Str("port", a.config.Server.Port).
			Msg("🌐 Server starting...")

		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.GetGlobal().Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	logger.GetGlobal().Info().Msg("✅ Server started successfully. Press Ctrl+C to stop.")

	// Aguardar sinal de parada
	<-sigChan

	logger.GetGlobal().Info().Msg("🛑 Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		logger.GetGlobal().Error().Err(err).Msg("Server forced to shutdown")
		return err
	}

	logger.GetGlobal().Info().Msg("✅ Server stopped gracefully")
	return nil
}

// buildDSN constrói a string de conexão do PostgreSQL
func (a *App) buildDSN(dbCfg configs.DatabaseConfig) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbCfg.User,
		dbCfg.Password,
		dbCfg.Host,
		dbCfg.Port,
		dbCfg.Name,
		dbCfg.SSLMode,
	)
}
