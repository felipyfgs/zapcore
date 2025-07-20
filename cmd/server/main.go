package main

import (
	"log"

	"zapcore/internal/app/config"
	"zapcore/internal/app/server"
)

func main() {
	// Carregar configurações
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Erro ao carregar configurações: %v", err)
	}

	// Validar configurações
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuração inválida: %v", err)
	}

	// Criar servidor
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Erro ao criar servidor: %v", err)
	}

	// Iniciar servidor
	if err := srv.Start(); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}
