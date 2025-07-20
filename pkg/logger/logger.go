package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Logger encapsula o zerolog com configurações padronizadas
type Logger struct {
	logger zerolog.Logger
}

// Config representa as configurações do logger
type Config struct {
	Level         string // debug, info, warn, error, fatal
	Format        string // console, json
	DualOutput    bool   // true para ativar saída dupla (terminal + arquivo)
	ConsoleFormat string // console format para terminal
	FileFormat    string // json format para arquivo
	FilePath      string // caminho do arquivo de log
}

// New cria uma nova instância do logger centralizado
func New(config Config) *Logger {
	// Configurar nível de log
	level := parseLogLevel(config.Level)
	zerolog.SetGlobalLevel(level)

	var zeroLogger zerolog.Logger

	if config.DualOutput {
		// Configurar saída dupla (terminal + arquivo)
		zeroLogger = createDualOutputLogger(config)
	} else {
		// Configurar saída única (compatibilidade com sistema atual)
		if config.Format == "console" {
			// Formato console colorido para desenvolvimento
			zeroLogger = zerolog.New(zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: time.RFC3339,
				NoColor:    false,
			}).With().Timestamp().Caller().Logger()
		} else {
			// Formato JSON para produção
			zeroLogger = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()
		}
	}

	return &Logger{
		logger: zeroLogger,
	}
}

// NewFromZerolog cria um Logger a partir de um zerolog.Logger existente
func NewFromZerolog(zeroLogger zerolog.Logger) *Logger {
	return &Logger{
		logger: zeroLogger,
	}
}

// createDualOutputLogger cria um logger com saída dupla (terminal + arquivo)
func createDualOutputLogger(config Config) zerolog.Logger {
	// Criar writer para console (colorido)
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	}

	// Criar writer para arquivo (JSON)
	fileWriter, err := createFileWriter(config.FilePath)
	if err != nil {
		// Se falhar ao criar arquivo, usar apenas console
		fmt.Printf("Aviso: Falha ao criar arquivo de log (%v), usando apenas console\n", err)
		return zerolog.New(consoleWriter).With().Timestamp().Caller().Logger()
	}

	// Combinar os dois writers
	multiWriter := zerolog.MultiLevelWriter(consoleWriter, fileWriter)

	return zerolog.New(multiWriter).With().Timestamp().Caller().Logger()
}

// createFileWriter cria um writer para arquivo com rotação diária
func createFileWriter(filePath string) (io.Writer, error) {
	// Criar diretório se não existir
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("erro ao criar diretório %s: %w", dir, err)
	}

	// Gerar nome do arquivo com data atual
	now := time.Now()
	fileName := fmt.Sprintf("zapcore-%s.log", now.Format("2006-01-02"))
	fullPath := filepath.Join(dir, fileName)

	// Abrir arquivo para escrita (append)
	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir arquivo %s: %w", fullPath, err)
	}

	return file, nil
}

// parseLogLevel converte string para zerolog.Level
func parseLogLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
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
	case "disabled":
		return zerolog.Disabled
	default:
		return zerolog.InfoLevel
	}
}

// Debug cria um evento de debug
func (l *Logger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

// Info cria um evento de info
func (l *Logger) Info() *zerolog.Event {
	return l.logger.Info()
}

// Warn cria um evento de warning
func (l *Logger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

// Error cria um evento de erro
func (l *Logger) Error() *zerolog.Event {
	return l.logger.Error()
}

// Fatal cria um evento fatal (termina a aplicação)
func (l *Logger) Fatal() *zerolog.Event {
	return l.logger.Fatal()
}

// Panic cria um evento de panic
func (l *Logger) Panic() *zerolog.Event {
	return l.logger.Panic()
}

// With cria um novo logger com campos adicionais
func (l *Logger) With() zerolog.Context {
	return l.logger.With()
}

// GetZerolog retorna o zerolog.Logger interno para compatibilidade
func (l *Logger) GetZerolog() zerolog.Logger {
	return l.logger
}

// WithSessionID adiciona session_id ao contexto do logger
func (l *Logger) WithSessionID(sessionID string) *Logger {
	return &Logger{
		logger: l.logger.With().Str("session_id", sessionID).Logger(),
	}
}

// WithJID adiciona jid ao contexto do logger
func (l *Logger) WithJID(jid string) *Logger {
	return &Logger{
		logger: l.logger.With().Str("jid", jid).Logger(),
	}
}

// WithStatus adiciona status ao contexto do logger
func (l *Logger) WithStatus(status string) *Logger {
	return &Logger{
		logger: l.logger.With().Str("status", status).Logger(),
	}
}

// WithError adiciona erro ao contexto do logger
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		logger: l.logger.With().Err(err).Logger(),
	}
}

// WithField adiciona um campo personalizado ao contexto do logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		logger: l.logger.With().Interface(key, value).Logger(),
	}
}

// WithFields adiciona múltiplos campos ao contexto do logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	ctx := l.logger.With()
	for key, value := range fields {
		ctx = ctx.Interface(key, value)
	}
	return &Logger{
		logger: ctx.Logger(),
	}
}

// Global logger instance
var globalLogger *Logger

// Init inicializa o logger global
func Init(config Config) {
	globalLogger = New(config)
}

// InitFromZerolog inicializa o logger global a partir de um zerolog.Logger
func InitFromZerolog(zeroLogger zerolog.Logger) {
	globalLogger = NewFromZerolog(zeroLogger)
}

// Get retorna o logger global
func Get() *Logger {
	if globalLogger == nil {
		// Fallback para logger padrão se não foi inicializado
		globalLogger = New(Config{
			Level:  "info",
			Format: "console",
		})
	}
	return globalLogger
}

// Funções de conveniência para usar o logger global

// Debug cria um evento de debug no logger global
func Debug() *zerolog.Event {
	return Get().Debug()
}

// Info cria um evento de info no logger global
func Info() *zerolog.Event {
	return Get().Info()
}

// Warn cria um evento de warning no logger global
func Warn() *zerolog.Event {
	return Get().Warn()
}

// Error cria um evento de erro no logger global
func Error() *zerolog.Event {
	return Get().Error()
}

// Fatal cria um evento fatal no logger global
func Fatal() *zerolog.Event {
	return Get().Fatal()
}

// Panic cria um evento de panic no logger global
func Panic() *zerolog.Event {
	return Get().Panic()
}

// WithSessionID adiciona session_id ao logger global
func WithSessionID(sessionID string) *Logger {
	return Get().WithSessionID(sessionID)
}

// WithJID adiciona jid ao logger global
func WithJID(jid string) *Logger {
	return Get().WithJID(jid)
}

// WithStatus adiciona status ao logger global
func WithStatus(status string) *Logger {
	return Get().WithStatus(status)
}

// WithError adiciona erro ao logger global
func WithError(err error) *Logger {
	return Get().WithError(err)
}

// WithField adiciona um campo ao logger global
func WithField(key string, value interface{}) *Logger {
	return Get().WithField(key, value)
}

// WithFields adiciona múltiplos campos ao logger global
func WithFields(fields map[string]interface{}) *Logger {
	return Get().WithFields(fields)
}
