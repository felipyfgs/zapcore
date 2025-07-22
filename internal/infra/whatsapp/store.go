package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"zapcore/pkg/logger"

	"github.com/rs/zerolog"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// WhatsAppLoggerAdapter adapta nosso logger centralizado para o formato do whatsmeow
type WhatsAppLoggerAdapter struct {
	logger *logger.Logger
}

// NewWhatsAppLoggerAdapter cria um novo adapter de logger para o whatsmeow
func NewWhatsAppLoggerAdapter(logger *logger.Logger) waLog.Logger {
	return &WhatsAppLoggerAdapter{
		logger: logger,
	}
}

// Errorf implementa waLog.Logger
func (w *WhatsAppLoggerAdapter) Errorf(msg string, args ...interface{}) {
	w.logger.Error().Msgf(msg, args...)
}

// Warnf implementa waLog.Logger
func (w *WhatsAppLoggerAdapter) Warnf(msg string, args ...interface{}) {
	w.logger.Warn().Msgf(msg, args...)
}

// Infof implementa waLog.Logger
func (w *WhatsAppLoggerAdapter) Infof(msg string, args ...interface{}) {
	w.logger.Info().Msgf(msg, args...)
}

// Debugf implementa waLog.Logger
func (w *WhatsAppLoggerAdapter) Debugf(msg string, args ...interface{}) {
	w.logger.Debug().Msgf(msg, args...)
}

// Sub implementa waLog.Logger
func (w *WhatsAppLoggerAdapter) Sub(module string) waLog.Logger {
	return &WhatsAppLoggerAdapter{
		logger: w.logger.WithField("module", module),
	}
}

// StoreManager gerencia o store do whatsmeow
type StoreManager struct {
	container *sqlstore.Container
	logger    *logger.Logger
}

// NewStoreManager cria um novo gerenciador de store
func NewStoreManager(db *sql.DB, zeroLogger zerolog.Logger) (*StoreManager, error) {
	// Criar logger adapter para o whatsmeow que usa nosso logger centralizado
	waLogger := NewWhatsAppLoggerAdapter(logger.NewFromZerolog(zeroLogger))

	// Criar container do sqlstore
	container := sqlstore.NewWithDB(db, "postgres", waLogger)

	// Executar upgrade para criar as tabelas do whatsmeow
	ctx := context.Background()
	err := container.Upgrade(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer upgrade do banco whatsmeow: %w", err)
	}

	logger.WithFields(map[string]interface{}{
		"component": "whatsapp",
		"operation": "store_init",
		"status":    "completed",
	}).Info().Msg("📱 Tabelas WhatsApp OK")

	return &StoreManager{
		container: container,
		logger:    logger.NewFromZerolog(zeroLogger),
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
