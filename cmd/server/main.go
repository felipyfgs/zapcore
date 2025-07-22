package main

import (
	"fmt"
	"runtime"
	"time"

	"zapcore/internal/app/config"
	"zapcore/internal/app/server"
	"zapcore/pkg/logger"

	"github.com/fatih/color"
)

// printStartupInfo exibe informações básicas de inicialização
func printStartupInfo() {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("%s ZAPCORE - WhatsApp API Server %s\n",
		green("🚀"), cyan("v1.0.0"))
	fmt.Printf("Go: %s | Sistema: %s | Iniciado: %s\n",
		runtime.Version(),
		runtime.GOOS+"/"+runtime.GOARCH,
		time.Now().Format("15:04:05"))
	fmt.Println()
}

func main() {
	// Exibir informações de inicialização
	printStartupInfo()

	// Carregar configurações
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("❌ Erro ao carregar configurações: %v\n", err)
		return
	}

	// Inicializar logger centralizado
	logger.Init(logger.Config{
		Level:         cfg.Log.Level,
		Format:        cfg.Log.Format,
		DualOutput:    cfg.Log.DualOutput,
		ConsoleFormat: cfg.Log.ConsoleFormat,
		FileFormat:    cfg.Log.FileFormat,
		FilePath:      cfg.Log.FilePath,
	})

	// A partir daqui, usar apenas o logger centralizado
	logger.WithFields(map[string]interface{}{
		"component": "main",
		"phase":     "startup",
	}).Info().Msg("📋 Config carregada")

	// Validar configurações
	logger.WithFields(map[string]interface{}{
		"component": "main",
		"phase":     "validation",
	}).Info().Msg("🔍 Validando config")
	if err := cfg.Validate(); err != nil {
		logger.WithFields(map[string]interface{}{
			"component": "main",
			"phase":     "validation",
		}).Fatal().Err(err).Msg("❌ Configuração inválida")
	}

	// Criar servidor
	logger.WithFields(map[string]interface{}{
		"component": "main",
		"phase":     "initialization",
	}).Info().Msg("🏗️ Criando componentes")
	srv, err := server.New(cfg)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"component": "main",
			"phase":     "initialization",
		}).Fatal().Err(err).Msg("❌ Erro ao criar servidor")
	}

	// Iniciar servidor
	logger.WithFields(map[string]interface{}{
		"component": "main",
		"phase":     "startup",
	}).Info().Msg("🚀 Iniciando server")
	if err := srv.Start(); err != nil {
		logger.WithFields(map[string]interface{}{
			"component": "main",
			"phase":     "startup",
		}).Fatal().Err(err).Msg("❌ Erro ao iniciar servidor")
	}
}
