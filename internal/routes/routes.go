package routes

import (
	"net/http"

	"wamex/internal/handler"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Router configura e retorna todas as rotas da aplicação
func SetupRoutes(sessionHandler *handler.SessionHandler) chi.Router {
	router := chi.NewRouter()

	// Middleware básicos do Chi
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)

	// Middleware customizados para CORS e headers
	router.Use(corsMiddleware)
	router.Use(jsonMiddleware)

	// Grupo de rotas para sessões
	router.Route("/sessions", func(r chi.Router) {
		// POST /sessions/add - Cria uma nova sessão do WhatsApp
		r.Post("/add", sessionHandler.CreateSession)

		// GET /sessions/list - Lista todas as sessões ativas e registradas
		r.Get("/list", sessionHandler.ListSessions)

		// GET /sessions/list/{session} - Retorna informações detalhadas de uma sessão específica
		r.Get("/list/{session}", sessionHandler.GetSession)

		// DELETE /sessions/del/{session} - Remove permanentemente uma sessão existente
		r.Delete("/del/{session}", sessionHandler.DeleteSession)

		// POST /sessions/connect/{session} - Estabelece a conexão da sessão com o WhatsApp
		r.Post("/connect/{session}", sessionHandler.ConnectSession)

		// POST /sessions/disconnect/{session} - Desconecta a sessão do WhatsApp
		r.Post("/disconnect/{session}", sessionHandler.DisconnectSession)

		// GET /sessions/status/{session} - Consulta o status atual da sessão
		r.Get("/status/{session}", sessionHandler.GetSessionStatus)

		// GET /sessions/qr/{session} - Gera e retorna o QR Code para autenticação
		r.Get("/qr/{session}", sessionHandler.GetQRCode)

		// POST /sessions/pairphone/{session} - Emparelha um telefone com a sessão
		r.Post("/pairphone/{session}", sessionHandler.PairPhone)

		// POST /sessions/send/{session} - Envia mensagem de texto
		r.Post("/send/{session}", sessionHandler.SendTextMessage)
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
