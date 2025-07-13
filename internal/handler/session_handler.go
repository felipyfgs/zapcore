package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"wamex/internal/domain"

	"github.com/go-chi/chi/v5"
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
// GET /sessions/{sessionID}/info - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	// Obtém o sessionID resolvido pelo middleware (pode ter sido convertido de name para ID)
	sessionID := chi.URLParam(r, "sessionID")

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
// DELETE /sessions/{sessionID}/delete - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

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
// POST /sessions/connect/{sessionID} - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) ConnectSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

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
// POST /sessions/disconnect/{sessionID} - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) DisconnectSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

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
// GET /sessions/status/{sessionID} - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) GetSessionStatus(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

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
// GET /sessions/qr/{sessionID} - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) GetQRCode(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

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
// POST /sessions/pairphone/{sessionID} - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) PairPhone(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

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
// POST /message/{sessionID}/send/text - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) SendTextMessage(w http.ResponseWriter, r *http.Request) {
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

	// Validações robustas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	if len(req.Phone) < 10 || len(req.Phone) > 15 {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidPhone, "Phone number must be between 10 and 15 digits", "phone")
		return
	}

	if req.Body == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_MESSAGE", "Message text is required", "body")
		return
	}

	if len(req.Body) > 4096 {
		h.writeMessageError(w, http.StatusBadRequest, "MESSAGE_TOO_LONG", "Message text cannot exceed 4096 characters", "body")
		return
	}

	// Busca informações da sessão para resposta
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, domain.ErrorCodeSessionNotFound, "Session not found", "sessionID")
		return
	}
	if session == nil {
		h.writeMessageError(w, http.StatusNotFound, domain.ErrorCodeSessionNotFound, "Session is nil", "sessionID")
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

// SendDocumentMessage envia uma mensagem de documento
// POST /message/{sessionID}/send/document - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) SendDocumentMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_SESSION", "Session ID is required", "sessionID")
		return
	}

	// Decodifica a requisição usando estrutura padronizada
	var req domain.SendDocumentMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload", "")
		return
	}

	// Validações robustas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	if len(req.Phone) < 10 || len(req.Phone) > 15 {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidPhone, "Phone number must be between 10 and 15 digits", "phone")
		return
	}

	if req.Document == "" {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidBase64, "Document data is required", "document")
		return
	}

	if req.Filename == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_FILENAME", "Filename is required for documents", "filename")
		return
	}

	// Valida formato base64 para documento
	if !strings.HasPrefix(req.Document, "data:") {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidBase64, "Document must be in base64 data URL format", "document")
		return
	}

	// Busca informações da sessão para resposta
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, domain.ErrorCodeSessionNotFound, "Session not found", "sessionID")
		return
	}

	// Envia a mensagem de documento
	err = h.service.SendDocumentMessage(session.Session, req.Phone, req.Document, req.Filename, req.MimeType)
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
		if strings.Contains(err.Error(), "failed to process") {
			h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidBase64, "Invalid document format or data", "document")
			return
		}
		if strings.Contains(err.Error(), "failed to upload") {
			h.writeMessageError(w, http.StatusInternalServerError, domain.ErrorCodeUploadFailed, "Failed to upload document to WhatsApp", "")
			return
		}

		h.writeMessageError(w, http.StatusInternalServerError, domain.ErrorCodeSendFailed, "Failed to send document message", "")
		return
	}

	// Resposta de sucesso padronizada
	response := domain.MessageResponse{
		Success:   true,
		Message:   "Document message sent successfully",
		Timestamp: time.Now(),
		ID:        req.ID,
		Details: &domain.Details{
			Phone:       req.Phone,
			Type:        domain.MessageTypeDocument,
			Status:      "sent",
			SentAt:      time.Now(),
			SessionName: session.Session,
			MediaInfo: &domain.MediaInfo{
				MimeType: req.MimeType,
				Filename: req.Filename,
			},
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// SendAudioMessage envia uma mensagem de áudio
// POST /message/{sessionID}/send/audio - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) SendAudioMessage(w http.ResponseWriter, r *http.Request) {
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

	// Validações robustas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	if len(req.Phone) < 10 || len(req.Phone) > 15 {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidPhone, "Phone number must be between 10 and 15 digits", "phone")
		return
	}

	// Valida que pelo menos uma fonte de mídia foi fornecida
	sources := []string{}
	if req.Audio != "" {
		sources = append(sources, "audio")
	}
	if req.FilePath != "" {
		sources = append(sources, "filePath")
	}
	if req.URL != "" {
		sources = append(sources, "url")
	}
	if req.MinioID != "" {
		sources = append(sources, "minioId")
	}

	if len(sources) == 0 {
		h.writeMessageError(w, http.StatusBadRequest, "MISSING_MEDIA_SOURCE", "At least one media source must be provided (audio, filePath, url, or minioId)", "")
		return
	}
	if len(sources) > 1 {
		h.writeMessageError(w, http.StatusBadRequest, "MULTIPLE_MEDIA_SOURCES", fmt.Sprintf("Only one media source should be provided, received: %v", sources), "")
		return
	}

	// Validação específica para base64 se fornecido
	if req.Audio != "" && !strings.HasPrefix(req.Audio, "data:audio/") {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidBase64, "Audio must be in base64 data URL format (data:audio/...)", "audio")
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
		// Log do erro real para debug
		log.Error().
			Err(err).
			Str("session", session.Session).
			Str("phone", req.Phone).
			Str("file_path", req.FilePath).
			Msg("Erro detalhado no envio de áudio")

		// Trata diferentes tipos de erro
		if strings.Contains(err.Error(), "not connected") {
			h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeSessionOffline, "Session is not connected", "sessionID")
			return
		}
		if strings.Contains(err.Error(), "invalid recipient") {
			h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidPhone, "Invalid phone number format", "phone")
			return
		}
		if strings.Contains(err.Error(), "failed to process") || strings.Contains(err.Error(), "arquivo não encontrado") || strings.Contains(err.Error(), "erro ao fazer download") {
			h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidBase64, "Invalid audio format or data", "audio")
			return
		}
		if strings.Contains(err.Error(), "failed to upload") {
			h.writeMessageError(w, http.StatusInternalServerError, domain.ErrorCodeUploadFailed, "Failed to upload audio to WhatsApp", "")
			return
		}
		if strings.Contains(err.Error(), "apenas uma fonte") || strings.Contains(err.Error(), "nenhuma fonte") {
			h.writeMessageError(w, http.StatusBadRequest, "INVALID_MEDIA_SOURCE", err.Error(), "")
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
			MediaInfo: &domain.MediaInfo{
				MimeType: req.MimeType,
				Duration: req.Duration,
			},
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// SendImageMessage envia uma mensagem de imagem
// POST /message/{sessionID}/send/image - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) SendImageMessage(w http.ResponseWriter, r *http.Request) {
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

	// Validações robustas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	if len(req.Phone) < 10 || len(req.Phone) > 15 {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidPhone, "Phone number must be between 10 and 15 digits", "phone")
		return
	}

	// Valida que pelo menos uma fonte de mídia foi fornecida
	sources := []string{}
	if req.Image != "" {
		sources = append(sources, "image")
	}
	if req.FilePath != "" {
		sources = append(sources, "filePath")
	}
	if req.URL != "" {
		sources = append(sources, "url")
	}
	if req.MinioID != "" {
		sources = append(sources, "minioId")
	}

	if len(sources) == 0 {
		h.writeMessageError(w, http.StatusBadRequest, "MISSING_MEDIA_SOURCE", "At least one media source must be provided (image, filePath, url, or minioId)", "")
		return
	}
	if len(sources) > 1 {
		h.writeMessageError(w, http.StatusBadRequest, "MULTIPLE_MEDIA_SOURCES", fmt.Sprintf("Only one media source should be provided, received: %v", sources), "")
		return
	}

	// Validação específica para base64 se fornecido
	if req.Image != "" && !strings.HasPrefix(req.Image, "data:image/") {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidBase64, "Image must be in base64 data URL format (data:image/...)", "image")
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
		if strings.Contains(err.Error(), "failed to process") || strings.Contains(err.Error(), "arquivo não encontrado") || strings.Contains(err.Error(), "erro ao fazer download") {
			h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidBase64, "Invalid image source or data", "image")
			return
		}
		if strings.Contains(err.Error(), "failed to upload") {
			h.writeMessageError(w, http.StatusInternalServerError, domain.ErrorCodeUploadFailed, "Failed to upload image to WhatsApp", "")
			return
		}
		if strings.Contains(err.Error(), "apenas uma fonte") || strings.Contains(err.Error(), "nenhuma fonte") {
			h.writeMessageError(w, http.StatusBadRequest, "INVALID_MEDIA_SOURCE", err.Error(), "")
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
				Filename: req.Filename,
			},
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// SendStickerMessage envia uma mensagem de sticker
// POST /message/{sessionID}/send/sticker - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) SendStickerMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_SESSION", "Session ID is required", "sessionID")
		return
	}

	// Decodifica a requisição usando estrutura padronizada
	var req domain.SendStickerMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload", "")
		return
	}

	// Validações robustas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	if len(req.Phone) < 10 || len(req.Phone) > 15 {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidPhone, "Phone number must be between 10 and 15 digits", "phone")
		return
	}

	if req.Sticker == "" {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidBase64, "Sticker data is required", "sticker")
		return
	}

	// Valida formato base64 para sticker (deve ser WebP)
	if !strings.HasPrefix(req.Sticker, "data:image/webp") {
		h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeUnsupportedMime, "Sticker must be in WebP format (data:image/webp;base64,...)", "sticker")
		return
	}

	// Busca informações da sessão para resposta
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, domain.ErrorCodeSessionNotFound, "Session not found", "sessionID")
		return
	}

	// Envia a mensagem de sticker
	err = h.service.SendStickerMessage(session.Session, req.Phone, req.Sticker)
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
		if strings.Contains(err.Error(), "failed to process") {
			h.writeMessageError(w, http.StatusBadRequest, domain.ErrorCodeInvalidBase64, "Invalid sticker format or data", "sticker")
			return
		}
		if strings.Contains(err.Error(), "failed to upload") {
			h.writeMessageError(w, http.StatusInternalServerError, domain.ErrorCodeUploadFailed, "Failed to upload sticker to WhatsApp", "")
			return
		}

		h.writeMessageError(w, http.StatusInternalServerError, domain.ErrorCodeSendFailed, "Failed to send sticker message", "")
		return
	}

	// Resposta de sucesso padronizada
	response := domain.MessageResponse{
		Success:   true,
		Message:   "Sticker message sent successfully",
		Timestamp: time.Now(),
		ID:        req.ID,
		Details: &domain.Details{
			Phone:       req.Phone,
			Type:        domain.MessageTypeSticker,
			Status:      "sent",
			SentAt:      time.Now(),
			SessionName: session.Session,
			MediaInfo: &domain.MediaInfo{
				MimeType: domain.MimeTypeImageWebP,
			},
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

// writeMessageError escreve uma resposta de erro específica para mensagens
func (h *SessionHandler) writeMessageError(w http.ResponseWriter, statusCode int, code, message, field string) {
	response := domain.MessageResponse{
		Success:   false,
		Message:   message,
		Timestamp: time.Now(),
	}

	// Log do erro para debugging (usando os parâmetros code e field)
	log.Debug().
		Str("error_code", code).
		Str("field", field).
		Str("message", message).
		Int("status_code", statusCode).
		Msg("Message error response")

	h.writeJSONResponse(w, statusCode, response)
}
