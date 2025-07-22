package middleware

import (
	"time"
	"zapcore/pkg/logger"

	"github.com/gin-gonic/gin"
)

// LoggingConfig representa a configuração do middleware de logging
type LoggingConfig struct {
	Logger     *logger.Logger
	SkipPaths  []string
	TimeFormat string
}

// DefaultLoggingConfig retorna a configuração padrão do logging
func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Logger:     logger.Get(),
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

		// Log baseado no status code
		status := c.Writer.Status()

		// Preparar log event usando o logger centralizado
		logEvent := config.Logger.WithFields(map[string]interface{}{
			"method":        c.Request.Method,
			"path":          path,
			"status":        status,
			"latency":       latency.String(),
			"ip":            c.ClientIP(),
			"user_agent":    c.Request.UserAgent(),
			"timestamp":     start.Format(config.TimeFormat),
			"response_size": c.Writer.Size(),
		})

		// Adicionar query parameters se existirem
		if c.Request.URL.RawQuery != "" {
			logEvent = logEvent.WithField("query", c.Request.URL.RawQuery)
		}

		// Adicionar erro se existir
		if len(c.Errors) > 0 {
			logEvent = logEvent.WithField("errors", c.Errors.String())
		}

		// Log baseado no status code
		switch {
		case status >= 500:
			logEvent.Error().Msg("HTTP Request")
		case status >= 400:
			logEvent.Warn().Msg("HTTP Request")
		default:
			logEvent.Info().Msg("HTTP Request")
		}
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
