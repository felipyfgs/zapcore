package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

func main() {
	// Criar o roteador Chi
	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(render.SetContentType(render.ContentTypeJSON))

	// Inicializar o gerenciador de sessões
	sessionManager := NewSessionManager()

	// Configurar rotas
	r.Route("/sessions", func(r chi.Router) {
		r.Post("/", sessionManager.CreateSession) // POST /sessions
		r.Get("/", sessionManager.ListSessions)   // GET /sessions

		r.Route("/{sessionID}", func(r chi.Router) {
			r.Get("/", sessionManager.GetSession)                   // GET /sessions/{session}
			r.Delete("/", sessionManager.DeleteSession)             // DELETE /sessions/{session}
			r.Post("/connect", sessionManager.ConnectSession)       // POST /sessions/{session}/connect
			r.Post("/disconnect", sessionManager.DisconnectSession) // POST /sessions/{session}/disconnect
			r.Get("/status", sessionManager.GetSessionStatus)       // GET /sessions/{session}/status
			r.Get("/qr", sessionManager.GetQRCode)                  // GET /sessions/{session}/qr
			r.Get("/device", sessionManager.GetSessionWithDevice)   // GET /sessions/{session}/device
			r.Post("/send", sessionManager.SendMessage)             // POST /sessions/{session}/send
		})
	})

	// Configurar servidor
	server := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Canal para capturar sinais do sistema
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Iniciar servidor em goroutine
	go func() {
		fmt.Println("🚀 Servidor iniciado na porta 8080")
		fmt.Println("📱 API WhatsApp disponível em http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Erro ao iniciar servidor: %v", err)
		}
	}()

	// Aguardar sinal de encerramento
	<-quit
	fmt.Println("\n🛑 Encerrando servidor...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Erro ao encerrar servidor: %v", err)
	}

	fmt.Println("✅ Servidor encerrado com sucesso")
}
