package router

import (
	"time"

	"zapcore/internal/http/handlers"
	"zapcore/internal/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Config representa a configuração do router
type Config struct {
	Logger          zerolog.Logger
	APIKey          string
	RateLimitReqs   int
	RateLimitWindow string
	CORSOrigins     []string
	CORSMethods     []string
	CORSHeaders     []string
}

// Router representa o router principal da aplicação
type Router struct {
	config         Config
	sessionHandler *handlers.SessionHandler
	messageHandler *handlers.MessageHandler
	healthHandler  *handlers.HealthHandler
}

// NewRouter cria uma nova instância do router
func NewRouter(
	config Config,
	sessionHandler *handlers.SessionHandler,
	messageHandler *handlers.MessageHandler,
	healthHandler *handlers.HealthHandler,
) *Router {
	return &Router{
		config:         config,
		sessionHandler: sessionHandler,
		messageHandler: messageHandler,
		healthHandler:  healthHandler,
	}
}

// Setup configura todas as rotas da aplicação
func (r *Router) Setup() *gin.Engine {
	// Configurar modo do Gin baseado no ambiente
	gin.SetMode(gin.ReleaseMode)

	// Criar engine do Gin
	engine := gin.New()

	// Middlewares globais
	r.setupMiddlewares(engine)

	// Rotas públicas (sem autenticação)
	r.setupPublicRoutes(engine)

	// Rotas protegidas (com autenticação)
	r.setupProtectedRoutes(engine)

	return engine
}

// setupMiddlewares configura os middlewares globais
func (r *Router) setupMiddlewares(engine *gin.Engine) {
	// Recovery middleware
	engine.Use(gin.Recovery())

	// Request ID middleware
	engine.Use(middleware.RequestID())

	// Logging middleware
	loggingConfig := middleware.DefaultLoggingConfig(r.config.Logger)
	engine.Use(middleware.Logging(loggingConfig))

	// CORS middleware
	corsConfig := middleware.CORSConfig{
		AllowedOrigins: r.config.CORSOrigins,
		AllowedMethods: r.config.CORSMethods,
		AllowedHeaders: r.config.CORSHeaders,
		MaxAge:         86400,
	}
	engine.Use(middleware.CORS(corsConfig))

	// Rate limiting middleware
	rateLimitConfig := middleware.DefaultRateLimitConfig(
		r.config.RateLimitReqs,
		parseDuration(r.config.RateLimitWindow),
		r.config.Logger,
	)
	engine.Use(middleware.RateLimit(rateLimitConfig))
}

// setupPublicRoutes configura as rotas públicas
func (r *Router) setupPublicRoutes(engine *gin.Engine) {
	// Health check routes
	engine.GET("/health", r.healthHandler.Check)
	engine.GET("/ready", r.healthHandler.Ready)
	engine.GET("/live", r.healthHandler.Live)

	// Root route
	engine.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"service": "ZapCore WhatsApp API",
			"version": "1.0.0",
			"status":  "running",
		})
	})
}

// setupProtectedRoutes configura as rotas protegidas
func (r *Router) setupProtectedRoutes(engine *gin.Engine) {
	// Middleware de autenticação para rotas protegidas
	authConfig := middleware.DefaultAuthConfig(r.config.APIKey, r.config.Logger)
	protected := engine.Group("/", middleware.APIKeyAuth(authConfig))

	// Rotas de sessões
	r.setupSessionRoutes(protected)

	// Rotas de mensagens
	r.setupMessageRoutes(protected)
}

// setupSessionRoutes configura as rotas de sessões
func (r *Router) setupSessionRoutes(group *gin.RouterGroup) {
	sessions := group.Group("/sessions")
	{
		// Gerenciamento de sessões
		sessions.POST("/add", r.sessionHandler.Create)
		sessions.GET("/list", r.sessionHandler.List)
		sessions.GET("/:sessionID", r.sessionHandler.GetStatus)
		// sessions.DELETE("/:sessionID", r.sessionHandler.Delete) // TODO: Implementar

		// Controle de conexão (aceita UUID ou nome da sessão)
		sessions.POST("/:sessionID/connect", r.sessionHandler.Connect)
		sessions.POST("/:sessionID/logout", r.sessionHandler.Disconnect)
		sessions.GET("/:sessionID/status", r.sessionHandler.GetStatus)

		// QR Code e emparelhamento - TODO: Implementar
		// sessions.GET("/:sessionID/qr", r.sessionHandler.GetQRCode)
		// sessions.POST("/:sessionID/pairphone", r.sessionHandler.PairPhone)

		// Configurações - TODO: Implementar
		// sessions.POST("/:sessionID/proxy/set", r.sessionHandler.SetProxy)
		// sessions.POST("/:sessionID/webhook/set", r.sessionHandler.SetWebhook)
	}
}

// setupMessageRoutes configura as rotas de mensagens
func (r *Router) setupMessageRoutes(group *gin.RouterGroup) {
	r.config.Logger.Debug().Msg("Configurando rotas de mensagens")

	messages := group.Group("/messages")
	{
		// Rotas de envio de mensagens por sessão
		sessionMessages := messages.Group("/:sessionID/send")
		{
			// Mensagem de texto
			r.config.Logger.Debug().Msg("Registrando rota POST /messages/:sessionID/send/text")
			sessionMessages.POST("/text", r.messageHandler.SendText)

			// Envio de mídia
			sessionMessages.POST("/image", r.messageHandler.SendImage)
			sessionMessages.POST("/video", r.messageHandler.SendVideo)
			sessionMessages.POST("/audio", r.messageHandler.SendAudio)
			sessionMessages.POST("/document", r.messageHandler.SendDocument)
			sessionMessages.POST("/sticker", r.messageHandler.SendSticker)

			// TODO: Implementar outros tipos de mensagem
			// sessionMessages.POST("/location", r.messageHandler.SendLocation)
			// sessionMessages.POST("/contact", r.messageHandler.SendContact)
			// sessionMessages.POST("/buttons", r.messageHandler.SendButtons)
			// sessionMessages.POST("/list", r.messageHandler.SendList)
			// sessionMessages.POST("/poll", r.messageHandler.SendPoll)
		}

		// TODO: Implementar gerenciamento de mensagens
		// sessionMessages.GET("/", r.messageHandler.GetMessages)
		// sessionMessages.GET("/:messageID", r.messageHandler.GetMessage)
		// sessionMessages.POST("/:messageID/read", r.messageHandler.MarkAsRead)
		// sessionMessages.PUT("/:messageID/edit", r.messageHandler.EditMessage)
	}
}

// parseDuration converte string de duração para time.Duration
func parseDuration(duration string) time.Duration {
	// Implementação simples - em produção usar time.ParseDuration
	switch duration {
	case "60s":
		return 60 * time.Second
	case "1m":
		return 1 * time.Minute
	case "5m":
		return 5 * time.Minute
	default:
		return 60 * time.Second
	}
}
