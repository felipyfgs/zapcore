package middleware

import (
	"net/http"
	"strings"
	"zapcore/pkg/logger"

	"github.com/gin-gonic/gin"
)

// AuthConfig representa a configuração do middleware de autenticação
type AuthConfig struct {
	APIKey     string
	HeaderName string
	QueryParam string
	SkipPaths  []string
	Logger     *logger.Logger
}

// DefaultAuthConfig retorna a configuração padrão de autenticação
func DefaultAuthConfig(apiKey string) AuthConfig {
	return AuthConfig{
		APIKey:     apiKey,
		HeaderName: "Authorization",
		QueryParam: "api_key",
		SkipPaths:  []string{"/health", "/ready", "/live"},
		Logger:     logger.Get(),
	}
}

// APIKeyAuth middleware para autenticação via API Key
func APIKeyAuth(config AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Verificar se deve pular a autenticação para este path
		path := c.Request.URL.Path
		for _, skipPath := range config.SkipPaths {
			if path == skipPath {
				c.Next()
				return
			}
		}

		// Extrair API Key da requisição
		apiKey := extractAPIKey(c, config)

		if apiKey == "" {
			config.Logger.Warn().
				Str("path", path).
				Str("method", c.Request.Method).
				Str("ip", c.ClientIP()).
				Msg("API Key não fornecida")

			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "API Key é obrigatória",
			})
			c.Abort()
			return
		}

		// Validar API Key
		if !isValidAPIKey(apiKey, config.APIKey) {
			config.Logger.Warn().
				Str("path", path).
				Str("method", c.Request.Method).
				Str("ip", c.ClientIP()).
				Str("api_key", maskAPIKey(apiKey)).
				Msg("API Key inválida")

			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "API Key inválida",
			})
			c.Abort()
			return
		}

		// API Key válida, continuar
		config.Logger.Debug().
			Str("path", path).
			Str("method", c.Request.Method).
			Str("ip", c.ClientIP()).
			Msg("Autenticação bem-sucedida")

		c.Next()
	}
}

// extractAPIKey extrai a API Key da requisição
func extractAPIKey(c *gin.Context, config AuthConfig) string {
	// Tentar extrair do header Authorization
	authHeader := c.GetHeader(config.HeaderName)
	if authHeader != "" {
		// Suportar formato "Bearer <api_key>" ou apenas "<api_key>"
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		return authHeader
	}

	// Tentar extrair do header X-API-Key
	if apiKey := c.GetHeader("X-API-Key"); apiKey != "" {
		return apiKey
	}

	// Tentar extrair do query parameter
	if apiKey := c.Query(config.QueryParam); apiKey != "" {
		return apiKey
	}

	return ""
}

// isValidAPIKey valida se a API Key é válida
func isValidAPIKey(providedKey, validKey string) bool {
	// Comparação simples e segura
	if len(providedKey) != len(validKey) {
		return false
	}

	// Comparação byte a byte para evitar timing attacks
	result := byte(0)
	for i := 0; i < len(providedKey); i++ {
		result |= providedKey[i] ^ validKey[i]
	}

	return result == 0
}

// maskAPIKey mascara a API Key para logs (mostra apenas os primeiros e últimos caracteres)
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "****"
	}

	return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}

// RequireAuth é um middleware mais simples que apenas verifica se a API Key está presente
func RequireAuth(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extrair API Key
		providedKey := c.GetHeader("X-API-Key")
		if providedKey == "" {
			providedKey = c.GetHeader("Authorization")
			if strings.HasPrefix(providedKey, "Bearer ") {
				providedKey = strings.TrimPrefix(providedKey, "Bearer ")
			}
		}
		if providedKey == "" {
			providedKey = c.Query("api_key")
		}

		// Validar
		if providedKey != apiKey {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "API Key inválida ou não fornecida",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
