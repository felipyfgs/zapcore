package handlers

import (
	"net/http"
	"time"

	"zapcore/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// HealthHandler gerencia as requisições de health check
type HealthHandler struct {
	logger    *logger.Logger
	startTime time.Time
	version   string
}

// NewHealthHandler cria uma nova instância do handler
func NewHealthHandler(zeroLogger zerolog.Logger, version string) *HealthHandler {
	return &HealthHandler{
		logger:    logger.NewFromZerolog(zeroLogger),
		startTime: time.Now(),
		version:   version,
	}
}

// Check verifica a saúde da aplicação
// @Summary Health Check
// @Description Verifica se a aplicação está funcionando corretamente
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *HealthHandler) Check(c *gin.Context) {
	uptime := time.Since(h.startTime)

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   h.version,
		Uptime:    uptime.String(),
	}

	c.JSON(http.StatusOK, response)
}

// Ready verifica se a aplicação está pronta para receber tráfego
// @Summary Readiness Check
// @Description Verifica se a aplicação está pronta para receber requisições
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} ErrorResponse
// @Router /ready [get]
func (h *HealthHandler) Ready(c *gin.Context) {
	// Aqui você pode adicionar verificações específicas:
	// - Conexão com banco de dados
	// - Conexões com serviços externos
	// - Verificação de recursos necessários

	response := HealthResponse{
		Status:    "ready",
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   h.version,
		Uptime:    time.Since(h.startTime).String(),
	}

	c.JSON(http.StatusOK, response)
}

// Live verifica se a aplicação está viva
// @Summary Liveness Check
// @Description Verifica se a aplicação está viva e respondendo
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /live [get]
func (h *HealthHandler) Live(c *gin.Context) {
	response := HealthResponse{
		Status:    "alive",
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   h.version,
		Uptime:    time.Since(h.startTime).String(),
	}

	c.JSON(http.StatusOK, response)
}

