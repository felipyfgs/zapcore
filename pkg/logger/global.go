package logger

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Instância global do logger para facilitar o uso
var globalLogger *Logger

// SetGlobal define o logger global da aplicação
func SetGlobal(logger *Logger) {
	globalLogger = logger
	log.Logger = logger.logger
}

// GetGlobal retorna o logger global
func GetGlobal() *Logger {
	if globalLogger == nil {
		// Inicializa apenas com InitFromEnv() que já configura o logger global
		InitFromEnv()
		// Cria uma instância wrapper para manter compatibilidade
		globalLogger = &Logger{
			logger: log.Logger,
			config: DefaultConfig(),
		}
	}
	return globalLogger
}

// Funções de conveniência para usar o logger global

// Debug registra uma mensagem de debug usando o logger global
func Debug() *zerolog.Event {
	return GetGlobal().Debug()
}

// Info registra uma mensagem informativa usando o logger global
func Info() *zerolog.Event {
	return GetGlobal().Info()
}

// Warn registra uma mensagem de aviso usando o logger global
func Warn() *zerolog.Event {
	return GetGlobal().Warn()
}

// Error registra uma mensagem de erro usando o logger global
func Error() *zerolog.Event {
	return GetGlobal().Error()
}

// Fatal registra uma mensagem fatal usando o logger global
func Fatal() *zerolog.Event {
	return GetGlobal().Fatal()
}

// Panic registra uma mensagem de pânico usando o logger global
func Panic() *zerolog.Event {
	return GetGlobal().Panic()
}

// WithComponent cria um logger com componente específico usando o logger global
func WithComponent(component string) *Logger {
	return GetGlobal().WithComponent(component)
}

// WithSession cria um logger com ID de sessão usando o logger global
func WithSession(sessionID string) *Logger {
	return GetGlobal().WithSession(sessionID)
}

// WithRequestID cria um logger com ID de requisição usando o logger global
func WithRequestID(requestID string) *Logger {
	return GetGlobal().WithRequestID(requestID)
}

// WithFields cria um logger com múltiplos campos usando o logger global
func WithFields(fields map[string]interface{}) *Logger {
	return GetGlobal().WithFields(fields)
}

// Funções de conveniência para componentes específicos

// Main cria um logger para o componente principal
func Main() *zerolog.Event {
	return log.Info().Str("component", "main")
}

// Database cria um logger para operações de banco de dados
func Database() *zerolog.Event {
	return log.Info().Str("component", "database")
}

// WhatsApp cria um logger para o serviço WhatsApp
func WhatsApp() *zerolog.Event {
	return log.Info().Str("component", "whatsapp")
}

// HTTP cria um logger para requisições HTTP
func HTTP() *zerolog.Event {
	return log.Info().Str("component", "http")
}

// Service cria um logger para serviços gerais
func Service(serviceName string) *zerolog.Event {
	return log.Info().Str("component", serviceName)
}
