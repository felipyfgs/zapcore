package logger

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

// LogLevel representa os níveis de log disponíveis
type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	FatalLevel LogLevel = "fatal"
	PanicLevel LogLevel = "panic"
)

// LogFormat representa os formatos de log disponíveis
type LogFormat string

const (
	JSONFormat    LogFormat = "json"
	ConsoleFormat LogFormat = "console"
)

// Config representa a configuração do logger
type Config struct {
	Level      LogLevel  `json:"level"`
	Format     LogFormat `json:"format"`
	Output     io.Writer `json:"-"`
	TimeFormat string    `json:"time_format"`
	Caller     bool      `json:"caller"`
	Stack      bool      `json:"stack"`
}

// DefaultConfig retorna a configuração padrão do logger
func DefaultConfig() *Config {
	return &Config{
		Level:      InfoLevel,
		Format:     JSONFormat,
		Output:     os.Stdout,
		TimeFormat: time.RFC3339,
		Caller:     true,
		Stack:      true,
	}
}

// Logger representa o logger da aplicação
type Logger struct {
	logger zerolog.Logger
	config *Config
}

// New cria uma nova instância do logger
func New(config *Config) *Logger {
	if config == nil {
		config = DefaultConfig()
	}

	// Configura o zerolog para usar stack traces do pkg/errors
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	// Configura o nível de log
	level := parseLogLevel(config.Level)
	zerolog.SetGlobalLevel(level)

	// Configura o formato de tempo
	zerolog.TimeFieldFormat = config.TimeFormat

	// Cria o logger base
	var logger zerolog.Logger

	// Configura o output baseado no formato
	switch config.Format {
	case ConsoleFormat:
		output := zerolog.ConsoleWriter{
			Out:        config.Output,
			TimeFormat: "15:04:05",
			NoColor:    false,
		}
		logger = zerolog.New(output)
	default:
		logger = zerolog.New(config.Output)
	}

	// Adiciona timestamp
	logger = logger.With().Timestamp().Logger()

	// Adiciona caller se habilitado
	if config.Caller {
		logger = logger.With().Caller().Logger()
	}

	return &Logger{
		logger: logger,
		config: config,
	}
}

// Init inicializa o logger global da aplicação
func Init(config *Config) {
	logger := New(config)
	log.Logger = logger.logger
}

// InitFromEnv inicializa o logger baseado em variáveis de ambiente
func InitFromEnv() {
	config := &Config{
		Level:      LogLevel(getEnv("LOG_LEVEL", string(InfoLevel))),
		Format:     LogFormat(getEnv("LOG_FORMAT", string(JSONFormat))),
		Output:     os.Stdout,
		TimeFormat: getEnv("LOG_TIME_FORMAT", time.RFC3339),
		Caller:     getEnvAsBool("LOG_CALLER", true),
		Stack:      getEnvAsBool("LOG_STACK", true),
	}

	// Configura output para stderr em desenvolvimento
	if getEnv("ENVIRONMENT", "development") == "development" {
		config.Output = os.Stderr
		config.Format = ConsoleFormat
	}

	Init(config)
}

// Debug registra uma mensagem de debug
func (l *Logger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

// Info registra uma mensagem informativa
func (l *Logger) Info() *zerolog.Event {
	return l.logger.Info()
}

// Warn registra uma mensagem de aviso
func (l *Logger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

// Error registra uma mensagem de erro
func (l *Logger) Error() *zerolog.Event {
	return l.logger.Error()
}

// Fatal registra uma mensagem fatal e encerra a aplicação
func (l *Logger) Fatal() *zerolog.Event {
	return l.logger.Fatal()
}

// Panic registra uma mensagem de pânico
func (l *Logger) Panic() *zerolog.Event {
	return l.logger.Panic()
}

// With cria um novo logger com campos adicionais
func (l *Logger) With() zerolog.Context {
	return l.logger.With()
}

// WithFields cria um novo logger com múltiplos campos
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	event := l.logger.With()
	for key, value := range fields {
		event = event.Interface(key, value)
	}
	
	return &Logger{
		logger: event.Logger(),
		config: l.config,
	}
}

// WithComponent cria um logger com componente específico
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		logger: l.logger.With().Str("component", component).Logger(),
		config: l.config,
	}
}

// WithSession cria um logger com ID de sessão
func (l *Logger) WithSession(sessionID string) *Logger {
	return &Logger{
		logger: l.logger.With().Str("session_id", sessionID).Logger(),
		config: l.config,
	}
}

// WithRequestID cria um logger com ID de requisição
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		logger: l.logger.With().Str("request_id", requestID).Logger(),
		config: l.config,
	}
}

// GetLevel retorna o nível atual do logger
func (l *Logger) GetLevel() LogLevel {
	return l.config.Level
}

// SetLevel define o nível do logger
func (l *Logger) SetLevel(level LogLevel) {
	l.config.Level = level
	zerolog.SetGlobalLevel(parseLogLevel(level))
}

// parseLogLevel converte string para zerolog.Level
func parseLogLevel(level LogLevel) zerolog.Level {
	switch strings.ToLower(string(level)) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

// getEnv obtém variável de ambiente com valor padrão
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsBool obtém variável de ambiente como boolean
func getEnvAsBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	
	switch strings.ToLower(value) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultValue
	}
}
