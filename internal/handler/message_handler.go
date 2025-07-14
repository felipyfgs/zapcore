package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"wamex/internal/domain"
	"wamex/internal/repository"

	"github.com/go-chi/chi/v5"
)

// MessageHandler gerencia as requisições HTTP para mensagens
type MessageHandler struct {
	service   domain.SessionService
	mediaRepo *repository.MediaRepository
}

// NewMessageHandler cria uma nova instância do handler de mensagens
func NewMessageHandler(service domain.SessionService, mediaRepo *repository.MediaRepository) *MessageHandler {
	return &MessageHandler{
		service:   service,
		mediaRepo: mediaRepo,
	}
}

// SendTextMessage envia uma mensagem de texto
// POST /message/{sessionID}/send/text - Aceita tanto sessionID quanto sessionName
func (h *MessageHandler) SendTextMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Session ID is required", nil)
		return
	}

	// Decodifica a requisição usando estrutura padronizada
	var req domain.SendTextMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload", err)
		return
	}

	// Validações básicas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	if req.Body == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_BODY", "Message body is required", "body")
		return
	}

	// Busca informações da sessão para resposta
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, domain.ErrorCodeSessionNotFound, "Session not found", "sessionID")
		return
	}

	// Envia a mensagem
	err = h.service.SendTextMessage(session.Session, req.Phone, req.Body)
	if err != nil {
		// Trata diferentes tipos de erro
		if strings.Contains(err.Error(), "not connected") {
			h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeSessionOffline, "Session is not connected", "sessionID")
			return
		}
		if strings.Contains(err.Error(), "invalid recipient") {
			h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidPhone, "Invalid phone number format", "phone")
			return
		}

		h.writeMessageError(w, http.StatusInternalServerError, domain.ErrorCodeSendFailed, "Failed to send message", "")
		return
	}

	// Resposta de sucesso padronizada
	response := domain.MessageResponse{
		Success:   true,
		Message:   "Text message sent successfully",
		Timestamp: time.Now(),
		ID:        req.ID,
		Details: &domain.Details{
			Phone:       req.Phone,
			Type:        domain.MessageTypeText,
			Status:      "sent",
			SentAt:      time.Now(),
			SessionName: session.Session,
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// SendImageMessage envia uma mensagem de imagem
// POST /message/{sessionID}/send/image - Aceita tanto sessionID quanto sessionName
func (h *MessageHandler) SendImageMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_SESSION", "Session ID is required", "sessionID")
		return
	}

	// Decodifica a requisição usando estrutura padronizada
	var req domain.SendImageMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload", "")
		return
	}

	// Validações básicas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	// Busca informações da sessão para resposta
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, domain.ErrorCodeSessionNotFound, "Session not found", "sessionID")
		return
	}

	// Envia a mensagem de imagem usando o novo sistema multi-source
	err = h.service.SendImageMessageMultiSource(session.Session, req.Phone, req.Image, req.FilePath, req.URL, req.MinioID, req.Caption, req.MimeType)
	if err != nil {
		// Trata diferentes tipos de erro
		if strings.Contains(err.Error(), "not connected") {
			h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeSessionOffline, "Session is not connected", "sessionID")
			return
		}
		if strings.Contains(err.Error(), "invalid recipient") {
			h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidPhone, "Invalid phone number format", "phone")
			return
		}

		h.writeMessageError(w, http.StatusInternalServerError, domain.ErrorCodeSendFailed, "Failed to send image message", "")
		return
	}

	// Resposta de sucesso padronizada
	response := domain.MessageResponse{
		Success:   true,
		Message:   "Image message sent successfully",
		Timestamp: time.Now(),
		ID:        req.ID,
		Details: &domain.Details{
			Phone:       req.Phone,
			Type:        domain.MessageTypeImage,
			Status:      "sent",
			SentAt:      time.Now(),
			SessionName: session.Session,
			MediaInfo: &domain.MediaInfo{
				MimeType: req.MimeType,
				Filename: req.Caption,
			},
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// SendAudioMessage envia uma mensagem de áudio
// POST /message/{sessionID}/send/audio - Aceita tanto sessionID quanto sessionName
func (h *MessageHandler) SendAudioMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_SESSION", "Session ID is required", "sessionID")
		return
	}

	// Decodifica a requisição usando estrutura padronizada
	var req domain.SendAudioMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload", "")
		return
	}

	// Validações básicas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	// Busca informações da sessão para resposta
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, domain.ErrorCodeSessionNotFound, "Session not found", "sessionID")
		return
	}

	// Envia a mensagem de áudio usando o novo sistema multi-source
	err = h.service.SendAudioMessageMultiSource(session.Session, req.Phone, req.Audio, req.FilePath, req.URL, req.MinioID)
	if err != nil {
		// Trata diferentes tipos de erro
		if strings.Contains(err.Error(), "not connected") {
			h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeSessionOffline, "Session is not connected", "sessionID")
			return
		}
		if strings.Contains(err.Error(), "invalid recipient") {
			h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidPhone, "Invalid phone number format", "phone")
			return
		}

		h.writeMessageError(w, http.StatusInternalServerError, domain.ErrorCodeSendFailed, "Failed to send audio message", "")
		return
	}

	// Resposta de sucesso padronizada
	response := domain.MessageResponse{
		Success:   true,
		Message:   "Audio message sent successfully",
		Timestamp: time.Now(),
		ID:        req.ID,
		Details: &domain.Details{
			Phone:       req.Phone,
			Type:        domain.MessageTypeAudio,
			Status:      "sent",
			SentAt:      time.Now(),
			SessionName: session.Session,
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// writeJSONResponse escreve uma resposta JSON
func (h *MessageHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// writeErrorResponse escreve uma resposta de erro
func (h *MessageHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	response := domain.MessageResponse{
		Success:   false,
		Message:   message,
		Timestamp: time.Now(),
	}

	h.writeJSONResponse(w, statusCode, response)
}

// writeMessageError escreve uma resposta de erro específica para mensagens
func (h *MessageHandler) writeMessageError(w http.ResponseWriter, statusCode int, errorCode, message, field string) {
	response := domain.MessageResponse{
		Success:   false,
		Message:   message,
		Timestamp: time.Now(),
	}

	h.writeJSONResponse(w, statusCode, response)
}
