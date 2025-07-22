package middleware

import (
	"net/http"
	"sync"
	"time"
	"zapcore/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RateLimitConfig representa a configuração do rate limiting
type RateLimitConfig struct {
	Requests  int                       // Número de requests permitidos
	Window    time.Duration             // Janela de tempo
	KeyFunc   func(*gin.Context) string // Função para extrair a chave (IP, API Key, etc)
	SkipPaths []string                  // Paths que devem ser ignorados
	Logger    *logger.Logger
}

// DefaultRateLimitConfig retorna a configuração padrão do rate limiting
func DefaultRateLimitConfig(requests int, window time.Duration) RateLimitConfig {
	return RateLimitConfig{
		Requests:  requests,
		Window:    window,
		KeyFunc:   func(c *gin.Context) string { return c.ClientIP() },
		SkipPaths: []string{"/health", "/ready", "/live"},
		Logger:    logger.Get(),
	}
}

// rateLimitEntry representa uma entrada no rate limiter
type rateLimitEntry struct {
	count     int
	resetTime time.Time
	mutex     sync.Mutex
}

// rateLimiter implementa um rate limiter em memória
type rateLimiter struct {
	entries map[string]*rateLimitEntry
	mutex   sync.RWMutex
	config  RateLimitConfig
}

// newRateLimiter cria um novo rate limiter
func newRateLimiter(config RateLimitConfig) *rateLimiter {
	rl := &rateLimiter{
		entries: make(map[string]*rateLimitEntry),
		config:  config,
	}

	// Iniciar limpeza periódica
	go rl.cleanup()

	return rl
}

// isAllowed verifica se a requisição é permitida
func (rl *rateLimiter) isAllowed(key string) bool {
	now := time.Now()

	rl.mutex.RLock()
	entry, exists := rl.entries[key]
	rl.mutex.RUnlock()

	if !exists {
		// Primeira requisição para esta chave
		rl.mutex.Lock()
		rl.entries[key] = &rateLimitEntry{
			count:     1,
			resetTime: now.Add(rl.config.Window),
		}
		rl.mutex.Unlock()
		return true
	}

	entry.mutex.Lock()
	defer entry.mutex.Unlock()

	// Verificar se a janela expirou
	if now.After(entry.resetTime) {
		entry.count = 1
		entry.resetTime = now.Add(rl.config.Window)
		return true
	}

	// Verificar se ainda há requests disponíveis
	if entry.count >= rl.config.Requests {
		return false
	}

	entry.count++
	return true
}

// cleanup remove entradas expiradas periodicamente
func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.Window)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		rl.mutex.Lock()

		for key, entry := range rl.entries {
			entry.mutex.Lock()
			if now.After(entry.resetTime) {
				delete(rl.entries, key)
			}
			entry.mutex.Unlock()
		}

		rl.mutex.Unlock()
	}
}

// RateLimit middleware para rate limiting
func RateLimit(config RateLimitConfig) gin.HandlerFunc {
	limiter := newRateLimiter(config)

	return func(c *gin.Context) {
		// Verificar se deve pular o rate limiting para este path
		path := c.Request.URL.Path
		for _, skipPath := range config.SkipPaths {
			if path == skipPath {
				c.Next()
				return
			}
		}

		// Extrair chave para rate limiting
		key := config.KeyFunc(c)

		// Verificar se a requisição é permitida
		if !limiter.isAllowed(key) {
			config.Logger.Warn().
				Str("key", key).
				Str("path", path).
				Str("method", c.Request.Method).
				Int("requests", config.Requests).
				Str("window", config.Window.String()).
				Msg("Rate limit excedido")

			c.Header("X-RateLimit-Limit", string(rune(config.Requests)))
			c.Header("X-RateLimit-Window", config.Window.String())

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Muitas requisições. Tente novamente mais tarde.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// APIKeyRateLimit cria um rate limiter baseado em API Key
func APIKeyRateLimit(requests int, window time.Duration) gin.HandlerFunc {
	config := RateLimitConfig{
		Requests: requests,
		Window:   window,
		KeyFunc: func(c *gin.Context) string {
			// Tentar extrair API Key
			apiKey := c.GetHeader("X-API-Key")
			if apiKey == "" {
				apiKey = c.GetHeader("Authorization")
			}
			if apiKey == "" {
				apiKey = c.Query("api_key")
			}

			// Se não tiver API Key, usar IP
			if apiKey == "" {
				return c.ClientIP()
			}

			return "api:" + apiKey
		},
		SkipPaths: []string{"/health", "/ready", "/live"},
		Logger:    logger.Get(),
	}

	return RateLimit(config)
}
