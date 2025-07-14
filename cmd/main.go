package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wamex/configs"
	"wamex/internal/handler"
	"wamex/internal/repository"
	"wamex/internal/routes"
	"wamex/internal/service"
	"wamex/pkg/logger"
	"wamex/pkg/storage"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func main() {
	// Carregar configurações
	cfg, err := configs.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Inicializar logger
	logger.InitFromEnv()
	logger.GetGlobal().Info().Msg("🚀 Starting WAMEX server...")

	// Inicializar banco de dados
	db, err := initDatabase(cfg)
	if err != nil {
		logger.GetGlobal().Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer db.Close()

	// Inicializar MinIO
	minioClient, err := storage.InitializeMinIO()
	if err != nil {
		logger.GetGlobal().Fatal().Err(err).Msg("Failed to initialize MinIO")
	}

	// Inicializar repositórios
	sessionRepo := repository.NewSessionRepository(db)
	mediaRepo := repository.NewMediaRepository(db)

	// Inicializar services
	whatsappService, err := service.NewWhatsAppService(
		sessionRepo,
		"postgres",
		buildDSN(cfg.Database),
	)
	if err != nil {
		logger.GetGlobal().Fatal().Err(err).Msg("Failed to initialize WhatsApp service")
	}

	// Unified media service disponível para uso futuro
	_ = service.NewUnifiedMediaService(mediaRepo, minioClient)

	// Inicializar handlers
	sessionHandler := handler.NewSessionHandler(whatsappService, mediaRepo)
	messageHandler := handler.NewMessageHandler(whatsappService, mediaRepo)
	mediaHandler := handler.NewMediaHandler(mediaRepo, minioClient)

	// Configurar rotas
	router := routes.SetupRoutes(sessionHandler, messageHandler, mediaHandler, whatsappService)

	// Configurar servidor HTTP
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// Canal para capturar sinais do sistema
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Iniciar servidor em goroutine
	go func() {
		logger.GetGlobal().Info().
			Str("host", cfg.Server.Host).
			Str("port", cfg.Server.Port).
			Msg("🌐 Server starting...")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.GetGlobal().Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	logger.GetGlobal().Info().Msg("✅ Server started successfully. Press Ctrl+C to stop.")

	// Aguardar sinal de parada
	<-quit
	logger.GetGlobal().Info().Msg("🛑 Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.GetGlobal().Fatal().Err(err).Msg("Server forced to shutdown")
	}

	logger.GetGlobal().Info().Msg("✅ Server stopped gracefully")
}

// initDatabase inicializa a conexão com o banco de dados
func initDatabase(cfg *configs.Config) (*bun.DB, error) {
	dsn := buildDSN(cfg.Database)

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

	logger.GetGlobal().Info().
		Str("host", cfg.Database.Host).
		Str("port", cfg.Database.Port).
		Str("database", cfg.Database.Name).
		Msg("📊 Database connected successfully")

	return db, nil
}

// buildDSN constrói a string de conexão do PostgreSQL
func buildDSN(dbCfg configs.DatabaseConfig) string {
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
