package handler

import (
	"encoding/json"
	"net/http"

	"wamex/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// SessionHandler gerencia as requisições HTTP para sessões
type SessionHandler struct {
	service domain.SessionService
}

// NewSessionHandler cria uma nova instância do handler de sessões
func NewSessionHandler(service domain.SessionService) *SessionHandler {
	return &SessionHandler{
		service: service,
	}
}

// CreateSession cria uma nova sessão WhatsApp
// POST /sessions/add
func (h *SessionHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateSessionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	session, err := h.service.CreateSession(&req)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create session", err)
		return
	}

	response := domain.SessionResponse{
		Success: true,
		Message: "Session created successfully",
		Data:    session,
	}

	h.writeJSONResponse(w, http.StatusCreated, response)
}

// ListSessions lista todas as sessões
// GET /sessions/list
func (h *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.service.ListSessions()
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to list sessions", err)
		return
	}

	response := domain.SessionResponse{
		Success: true,
		Message: "Sessions retrieved successfully",
		Data:    sessions,
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetSession obtém uma sessão específica
// GET /sessions/list/{session}
func (h *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session")

	if sessionID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Session ID is required", nil)
		return
	}

	session, err := h.service.GetSession(sessionID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get session", err)
		return
	}

	if session == nil {
		h.writeErrorResponse(w, http.StatusNotFound, "Session not found", nil)
		return
	}

	response := domain.SessionResponse{
		Success: true,
		Message: "Session retrieved successfully",
		Data:    session,
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// DeleteSession remove uma sessão
// DELETE /sessions/del/{session}
func (h *SessionHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session")

	if sessionID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Session ID is required", nil)
		return
	}

	err := h.service.DeleteSession(sessionID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete session", err)
		return
	}

	response := domain.SessionResponse{
		Success: true,
		Message: "Session deleted successfully",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// ConnectSession estabelece conexão com WhatsApp
// POST /sessions/connect/{session}
func (h *SessionHandler) ConnectSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session")

	if sessionID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Session ID is required", nil)
		return
	}

	err := h.service.ConnectSession(sessionID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to connect session", err)
		return
	}

	response := domain.SessionResponse{
		Success: true,
		Message: "Session connection initiated",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// DisconnectSession desconecta uma sessão
// POST /sessions/disconnect/{session}
func (h *SessionHandler) DisconnectSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session")

	if sessionID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Session ID is required", nil)
		return
	}

	err := h.service.DisconnectSession(sessionID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to disconnect session", err)
		return
	}

	response := domain.SessionResponse{
		Success: true,
		Message: "Session disconnected successfully",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetSessionStatus obtém o status de uma sessão
// GET /sessions/status/{session}
func (h *SessionHandler) GetSessionStatus(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session")

	if sessionID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Session ID is required", nil)
		return
	}

	statusResponse, err := h.service.GetSessionStatus(sessionID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get session status", err)
		return
	}

	h.writeJSONResponse(w, http.StatusOK, statusResponse)
}

// GetQRCode obtém o QR code de uma sessão
// GET /sessions/qr/{session}
func (h *SessionHandler) GetQRCode(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session")

	if sessionID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Session ID is required", nil)
		return
	}

	qrCode, err := h.service.GenerateQRCode(sessionID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get QR code", err)
		return
	}

	if qrCode == "" {
		h.writeErrorResponse(w, http.StatusNotFound, "QR code not available", nil)
		return
	}

	response := domain.QRCodeResponse{
		Success:   true,
		SessionID: sessionID,
		QRCode:    qrCode,
		Message:   "QR code retrieved successfully",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// PairPhone emparelha um telefone com a sessão
// POST /sessions/pairphone/{session}
func (h *SessionHandler) PairPhone(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["session"]

	if sessionID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Session ID is required", nil)
		return
	}

	var req domain.PairPhoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	err := h.service.PairPhone(sessionID, req.Phone)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to pair phone", err)
		return
	}

	response := domain.SessionResponse{
		Success: true,
		Message: "Phone pairing initiated",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// SendTextMessage envia uma mensagem de texto
// POST /sessions/send/{session}
func (h *SessionHandler) SendTextMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session")

	if sessionID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Session name is required", nil)
		return
	}

	// Estrutura para receber os dados da mensagem
	type SendMessageRequest struct {
		To      string `json:"to"`
		Message string `json:"message"`
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload", err)
		return
	}

	// Validações
	if req.To == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Recipient phone number is required", nil)
		return
	}

	if req.Message == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Message text is required", nil)
		return
	}

	// Envia a mensagem
	err := h.service.SendTextMessage(sessionID, req.To, req.Message)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to send message", err)
		return
	}

	// Resposta de sucesso
	response := domain.SessionResponse{
		Success: true,
		Message: "Message sent successfully",
		Data: map[string]interface{}{
			"to":      req.To,
			"message": req.Message,
			"session": sessionID,
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// writeJSONResponse escreve uma resposta JSON
func (h *SessionHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

// writeErrorResponse escreve uma resposta de erro
func (h *SessionHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	response := domain.SessionResponse{
		Success: false,
		Message: message,
	}

	if err != nil {
		response.Error = err.Error()
		log.Error().Err(err).Str("message", message).Msg("Handler error")
	}

	h.writeJSONResponse(w, statusCode, response)
}
