package routes

import (
	"net/http"

	"wamex/internal/handler"
	"wamex/internal/middleware"
	"wamex/internal/service"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// Router configura e retorna todas as rotas da aplicação
func SetupRoutes(sessionHandler *handler.SessionHandler, messageHandler *handler.MessageHandler, mediaHandler *handler.MediaHandler, whatsappService *service.WhatsAppService) chi.Router {
	router := chi.NewRouter()

	// Middleware básicos do Chi
	router.Use(chimiddleware.Logger)
	router.Use(chimiddleware.Recoverer)
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)

	// Middleware customizados para CORS e headers
	router.Use(corsMiddleware)
	router.Use(jsonMiddleware)

	// Cria o middleware de resolução de sessão
	sessionResolver := middleware.NewSessionResolver(whatsappService)

	// Grupo de rotas para gerenciamento de sessões
	router.Route("/sessions", func(r chi.Router) {
		// POST /sessions/add - Cria uma nova sessão do WhatsApp
		r.Post("/add", sessionHandler.CreateSession)

		// GET /sessions/list - Lista todas as sessões ativas e registradas
		r.Get("/list", sessionHandler.ListSessions)

		// Grupo de rotas que precisam do middleware de resolução universal
		r.Route("/", func(sr chi.Router) {
			// Aplica o middleware de resolução universal para todas as rotas com {sessionID}
			sr.Use(sessionResolver.Middleware())

			// GET /sessions/{sessionID}/info - Retorna informações detalhadas de uma sessão específica (aceita ID ou Name)
			sr.Get("/{sessionID}/info", sessionHandler.GetSession)

			// DELETE /sessions/{sessionID}/delete - Remove permanentemente uma sessão existente (aceita ID ou Name)
			sr.Delete("/{sessionID}/delete", sessionHandler.DeleteSession)

			// POST /sessions/{sessionID}/connect - Estabelece a conexão da sessão com o WhatsApp (aceita ID ou Name)
			sr.Post("/{sessionID}/connect", sessionHandler.ConnectSession)

			// POST /sessions/{sessionID}/disconnect - Desconecta a sessão do WhatsApp (aceita ID ou Name)
			sr.Post("/{sessionID}/disconnect", sessionHandler.DisconnectSession)

			// GET /sessions/{sessionID}/status - Consulta o status atual da sessão (aceita ID ou Name)
			sr.Get("/{sessionID}/status", sessionHandler.GetSessionStatus)

			// GET /sessions/{sessionID}/qr - Gera e retorna o QR Code para autenticação (aceita ID ou Name)
			sr.Get("/{sessionID}/qr", sessionHandler.GetQRCode)

			// POST /sessions/{sessionID}/pairphone - Emparelha um telefone com a sessão (aceita ID ou Name)
			sr.Post("/{sessionID}/pairphone", sessionHandler.PairPhone)
		})
	})

	// Grupo de rotas para envio de mensagens
	router.Route("/message", func(r chi.Router) {
		// Aplica o middleware de resolução universal para todas as rotas com {sessionID}
		r.Use(sessionResolver.Middleware())

		// POST /message/{sessionID}/send/text - Envia mensagem de texto (aceita ID ou Name)
		r.Post("/{sessionID}/send/text", messageHandler.SendTextMessage)

		// POST /message/{sessionID}/send/document - Envia mensagem de documento (aceita ID ou Name)
		r.Post("/{sessionID}/send/document", sessionHandler.SendDocumentMessage)

		// POST /message/{sessionID}/send/audio - Envia mensagem de áudio (aceita ID ou Name)
		r.Post("/{sessionID}/send/audio", messageHandler.SendAudioMessage)

		// POST /message/{sessionID}/send/image - Envia mensagem de imagem (aceita ID ou Name)
		r.Post("/{sessionID}/send/image", messageHandler.SendImageMessage)

		// POST /message/{sessionID}/send/sticker - Envia mensagem de sticker (aceita ID ou Name)
		r.Post("/{sessionID}/send/sticker", sessionHandler.SendStickerMessage)

		// POST /message/{sessionID}/send/location - Envia mensagem de localização (aceita ID ou Name)
		r.Post("/{sessionID}/send/location", sessionHandler.SendLocationMessage)

		// POST /message/{sessionID}/send/contact - Envia mensagem de contato (aceita ID ou Name)
		r.Post("/{sessionID}/send/contact", sessionHandler.SendContactMessage)

		// POST /message/{sessionID}/react - Reage a uma mensagem existente (aceita ID ou Name)
		r.Post("/{sessionID}/react", sessionHandler.ReactToMessage)

		// POST /message/{sessionID}/send/video - Envia mensagem de vídeo (aceita ID ou Name)
		r.Post("/{sessionID}/send/video", sessionHandler.SendVideoMessage)

		// POST /message/{sessionID}/edit - Edita uma mensagem existente (aceita ID ou Name)
		r.Post("/{sessionID}/edit", sessionHandler.EditMessage)

		// POST /message/{sessionID}/send/poll - Envia mensagem de enquete para grupos (aceita ID ou Name)
		r.Post("/{sessionID}/send/poll", sessionHandler.SendPollMessage)

		// POST /message/{sessionID}/send/list - Envia mensagem de lista interativa (aceita ID ou Name)
		r.Post("/{sessionID}/send/list", sessionHandler.SendListMessage)

		// POST /message/{sessionID}/send/media - Envia mensagem de mídia já uploadada (aceita ID ou Name)
		r.Post("/{sessionID}/send/media", sessionHandler.SendMediaMessage)
	})

	// Grupo de rotas para gerenciamento de mídia
	router.Route("/media", func(r chi.Router) {
		// POST /media/upload - Upload geral de mídia (detecta tipo automaticamente)
		r.Post("/upload", mediaHandler.UploadMedia)

		// GET /media/list - Listagem de mídias com paginação e filtros
		r.Get("/list", mediaHandler.ListMedia)

		// GET /media/{mediaID}/download - Download de mídia por ID
		r.Get("/{mediaID}/download", mediaHandler.DownloadMedia)

		// DELETE /media/{mediaID} - Deleção de mídia por ID
		r.Delete("/{mediaID}", mediaHandler.DeleteMedia)
	})

	// Rota de health check
	router.Get("/health", healthCheck)

	return router
}

// corsMiddleware adiciona headers CORS
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// jsonMiddleware define o content-type como JSON
func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// healthCheck endpoint para verificar se a API está funcionando
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok", "message": "WAMEX API is running"}`))
}
