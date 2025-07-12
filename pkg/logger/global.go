package logger

import (
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
		// Inicializa com configuração padrão se não foi configurado
		InitFromEnv()
		globalLogger = New(DefaultConfig())
	}
	return globalLogger
}

// Funções de conveniência para usar o logger global

// Debug registra uma mensagem de debug usando o logger global
func Debug() *Logger {
	return GetGlobal()
}

// Info registra uma mensagem informativa usando o logger global
func Info() *Logger {
	return GetGlobal()
}

// Warn registra uma mensagem de aviso usando o logger global
func Warn() *Logger {
	return GetGlobal()
}

// Error registra uma mensagem de erro usando o logger global
func Error() *Logger {
	return GetGlobal()
}

// Fatal registra uma mensagem fatal usando o logger global
func Fatal() *Logger {
	return GetGlobal()
}

// Panic registra uma mensagem de pânico usando o logger global
func Panic() *Logger {
	return GetGlobal()
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
