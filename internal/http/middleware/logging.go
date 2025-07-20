package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// LoggingConfig representa a configuração do middleware de logging
type LoggingConfig struct {
	Logger     zerolog.Logger
	SkipPaths  []string
	TimeFormat string
}

// DefaultLoggingConfig retorna a configuração padrão do logging
func DefaultLoggingConfig(logger zerolog.Logger) LoggingConfig {
	return LoggingConfig{
		Logger:     logger,
		SkipPaths:  []string{"/health", "/ready", "/live"},
		TimeFormat: time.RFC3339,
	}
}

// Logging middleware para log estruturado das requisições HTTP
func Logging(config LoggingConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Verificar se deve pular o log para este path
		path := c.Request.URL.Path
		for _, skipPath := range config.SkipPaths {
			if path == skipPath {
				c.Next()
				return
			}
		}

		// Capturar tempo de início
		start := time.Now()

		// Processar requisição
		c.Next()

		// Calcular latência
		latency := time.Since(start)

		// Preparar campos do log
		fields := map[string]interface{}{
			"method":     c.Request.Method,
			"path":       path,
			"status":     c.Writer.Status(),
			"latency":    latency.String(),
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
			"timestamp":  start.Format(config.TimeFormat),
		}

		// Adicionar query parameters se existirem
		if c.Request.URL.RawQuery != "" {
			fields["query"] = c.Request.URL.RawQuery
		}

		// Adicionar tamanho da resposta
		fields["response_size"] = c.Writer.Size()

		// Adicionar erro se existir
		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}

		// Log baseado no status code
		status := c.Writer.Status()

		// Preparar log event
		var event *zerolog.Event
		switch {
		case status >= 500:
			event = config.Logger.Error()
		case status >= 400:
			event = config.Logger.Warn()
		default:
			event = config.Logger.Info()
		}

		// Adicionar campos e enviar log
		event.
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", status).
			Dur("latency", latency).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Time("timestamp", start).
			Int("response_size", c.Writer.Size()).
			Msg("HTTP Request")
	}
}

// RequestID middleware para adicionar ID único a cada requisição
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// generateRequestID gera um ID único para a requisição
func generateRequestID() string {
	// Implementação simples usando timestamp
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

// randomString gera uma string aleatória
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

