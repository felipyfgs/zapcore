package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// SetupRoutes configura todas as rotas da API
func SetupRoutes(sessionManager *SessionManager) *chi.Mux {
	r := chi.NewRouter()

	// Middlewares globais
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	// Configurar CORS básico
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, apiKey")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Middleware de autenticação por API Key
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verificar API Key
			apiKey := r.Header.Get("apiKey")
			if apiKey == "" {
				apiKey = r.URL.Query().Get("apiKey")
			}

			expectedAPIKey := "njhfyikg" // Em produção, isso deveria vir de variável de ambiente
			if apiKey != expectedAPIKey {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "unauthorized", "message": "API Key inválida ou ausente", "code": 401}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Rota de health check (sem autenticação)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "message": "WAMEX API está funcionando"}`))
	})

	// Rotas da API
	setupSessionRoutes(r, sessionManager)

	return r
}

// setupSessionRoutes configura as rotas relacionadas a sessões
func setupSessionRoutes(r *chi.Mux, sessionManager *SessionManager) {
	r.Route("/sessions", func(r chi.Router) {
		// Rotas de gerenciamento de sessões
		r.Post("/", sessionManager.CreateSession) // POST /sessions
		r.Get("/", sessionManager.ListSessions)   // GET /sessions

		// Rotas específicas de uma sessão
		r.Route("/{sessionID}", func(r chi.Router) {
			// Informações da sessão
			r.Get("/", sessionManager.GetSession)       // GET /sessions/{session}
			r.Delete("/", sessionManager.DeleteSession) // DELETE /sessions/{session}

			// Controle de conexão
			r.Post("/connect", sessionManager.ConnectSession)       // POST /sessions/{session}/connect
			r.Post("/disconnect", sessionManager.DisconnectSession) // POST /sessions/{session}/disconnect
			r.Get("/status", sessionManager.GetSessionStatus)       // GET /sessions/{session}/status

			// QR Code e device info
			r.Get("/qr", sessionManager.GetQRCode)                // GET /sessions/{session}/qr
			r.Get("/device", sessionManager.GetSessionWithDevice) // GET /sessions/{session}/device

			// Envio de mensagens
			r.Post("/send", sessionManager.SendMessage) // POST /sessions/{session}/send
		})
	})
}
