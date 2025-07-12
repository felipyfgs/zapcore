package logger

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// HTTPMiddleware cria um middleware de logging para requisições HTTP
func HTTPMiddleware(logger *Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Cria um wrapper para capturar o status code
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Obtém ou gera request ID
			requestID := middleware.GetReqID(r.Context())
			if requestID == "" {
				requestID = generateRequestID()
			}

			// Cria logger específico para esta requisição
			reqLogger := logger.WithRequestID(requestID).WithFields(map[string]interface{}{
				"method":     r.Method,
				"path":       r.URL.Path,
				"remote_ip":  r.RemoteAddr,
				"user_agent": r.UserAgent(),
			})

			// Log da requisição iniciada
			reqLogger.Info().Msg("HTTP request started")

			// Processa a requisição
			next.ServeHTTP(ww, r)

			// Calcula duração
			duration := time.Since(start)

			// Log da requisição finalizada
			logEvent := reqLogger.Info().
				Int("status_code", ww.Status()).
				Dur("duration", duration).
				Int("response_size", ww.BytesWritten())

			// Adiciona nível de log baseado no status code
			switch {
			case ww.Status() >= 500:
				logEvent = reqLogger.Error().
					Int("status_code", ww.Status()).
					Dur("duration", duration).
					Int("response_size", ww.BytesWritten())
			case ww.Status() >= 400:
				logEvent = reqLogger.Warn().
					Int("status_code", ww.Status()).
					Dur("duration", duration).
					Int("response_size", ww.BytesWritten())
			}

			logEvent.Msg("HTTP request completed")
		})
	}
}

// RecoveryMiddleware cria um middleware de recovery com logging
func RecoveryMiddleware(logger *Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID := middleware.GetReqID(r.Context())
					
					logger.WithRequestID(requestID).Error().
						Interface("panic", err).
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Str("remote_ip", r.RemoteAddr).
						Msg("HTTP request panic recovered")

					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// generateRequestID gera um ID único para a requisição
func generateRequestID() string {
	// Implementação simples - em produção, considere usar UUID
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
