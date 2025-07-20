package main

import (
	"zapcore/internal/app/config"
	"zapcore/internal/app/server"
	"zapcore/pkg/logger"
)

func main() {
	// Carregar configurações
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("Erro ao carregar configurações")
	}

	// Inicializar logger centralizado PRIMEIRO
	logger.Init(logger.Config{
		Level:         cfg.Log.Level,
		Format:        cfg.Log.Format,
		DualOutput:    cfg.Log.DualOutput,
		ConsoleFormat: cfg.Log.ConsoleFormat,
		FileFormat:    cfg.Log.FileFormat,
		FilePath:      cfg.Log.FilePath,
	})

	if cfg.Log.DualOutput {
		logger.Info().
			Str("level", cfg.Log.Level).
			Str("console_format", cfg.Log.ConsoleFormat).
			Str("file_format", cfg.Log.FileFormat).
			Str("file_path", cfg.Log.FilePath).
			Bool("dual_output", cfg.Log.DualOutput).
			Msg("Logger centralizado inicializado com saída dupla")
	} else {
		logger.Info().
			Str("level", cfg.Log.Level).
			Str("format", cfg.Log.Format).
			Msg("Logger centralizado inicializado")
	}

	// Validar configurações
	if err := cfg.Validate(); err != nil {
		logger.Fatal().Err(err).Msg("Configuração inválida")
	}

	// Criar servidor
	srv, err := server.New(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Erro ao criar servidor")
	}

	// Iniciar servidor
	if err := srv.Start(); err != nil {
		logger.Fatal().Err(err).Msg("Erro ao iniciar servidor")
	}
}
