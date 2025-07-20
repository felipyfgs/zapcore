package whatsapp

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/zerolog"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// StoreManager gerencia o store do whatsmeow
type StoreManager struct {
	container *sqlstore.Container
	logger    zerolog.Logger
}

// NewStoreManager cria um novo gerenciador de store
func NewStoreManager(db *sql.DB, logger zerolog.Logger) (*StoreManager, error) {
	// Criar logger para o whatsmeow
	waLogger := waLog.Stdout("WhatsApp", "INFO", true)
	
	// Criar container do sqlstore
	container := sqlstore.NewWithDB(db, "postgres", waLogger)
	
	// Executar upgrade para criar as tabelas do whatsmeow
	ctx := context.Background()
	err := container.Upgrade(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer upgrade do banco whatsmeow: %w", err)
	}
	
	logger.Info().Msg("Tabelas do whatsmeow inicializadas com sucesso")
	
	return &StoreManager{
		container: container,
		logger:    logger,
	}, nil
}

// GetContainer retorna o container do sqlstore
func (sm *StoreManager) GetContainer() *sqlstore.Container {
	return sm.container
}

// Close fecha o container
func (sm *StoreManager) Close() error {
	if sm.container != nil {
		return sm.container.Close()
	}
	return nil
}
