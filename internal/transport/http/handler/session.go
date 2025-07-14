package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	entity "wamex/internal/domain/entity"
	domainService "wamex/internal/domain/service"
	"wamex/internal/infra/database"
	"wamex/internal/infra/storage"
	"wamex/internal/usecase/media"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// SessionHandler gerencia as requisições HTTP para sessões
type SessionHandler struct {
	service   domainService.SessionService
	mediaRepo *database.MediaRepository
}

// NewSessionHandler cria uma nova instância do handler de sessões
func NewSessionHandler(service domainService.SessionService, mediaRepo *database.MediaRepository) *SessionHandler {
	return &SessionHandler{
		service:   service,
		mediaRepo: mediaRepo,
	}
}

// CreateSession cria uma nova sessão WhatsApp
// POST /sessions/add
func (h *SessionHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req entity.CreateSessionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	session, err := h.service.CreateSession(&req)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create session", err)
		return
	}

	response := entity.SessionResponse{
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

	response := entity.SessionResponse{
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

	response := entity.SessionResponse{
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

	response := entity.SessionResponse{
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

	response := entity.SessionResponse{
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

	response := entity.SessionResponse{
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

	response := entity.QRCodeResponse{
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

	var req entity.PairPhoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	err := h.service.PairPhone(sessionID, req.Phone)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to pair phone", err)
		return
	}

	response := entity.SessionResponse{
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
	var req entity.SendTextMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload", err)
		return
	}

	// Validações robustas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	if len(req.Phone) < 10 || len(req.Phone) > 15 {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number must be between 10 and 15 digits", "phone")
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
		h.writeMessageError(w, http.StatusNotFound, entity.ErrorCodeSessionNotFound, "Session not found", "sessionID")
		return
	}
	if session == nil {
		h.writeMessageError(w, http.StatusNotFound, entity.ErrorCodeSessionNotFound, "Session is nil", "sessionID")
		return
	}

	// Envia a mensagem
	err = h.service.SendTextMessage(session.Session, req.Phone, req.Body)
	if err != nil {
		// Trata diferentes tipos de erro
		if strings.Contains(err.Error(), "not connected") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeSessionOffline, "Session is not connected", "sessionID")
			return
		}
		if strings.Contains(err.Error(), "invalid recipient") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Invalid phone number format", "phone")
			return
		}

		h.writeMessageError(w, http.StatusInternalServerError, entity.ErrorCodeSendFailed, "Failed to send message", "")
		return
	}

	// Resposta de sucesso padronizada
	response := entity.MessageResponse{
		Success:   true,
		Message:   "Text message sent successfully",
		Timestamp: time.Now(),
		ID:        req.ID,
		Details: &entity.Details{
			Phone:       req.Phone,
			Type:        entity.MessageTypeText,
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
	var req entity.SendDocumentMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload", "")
		return
	}

	// Validações robustas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	if len(req.Phone) < 10 || len(req.Phone) > 15 {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number must be between 10 and 15 digits", "phone")
		return
	}

	// Valida que pelo menos uma fonte de mídia foi fornecida
	sources := []string{}
	if req.Document != "" {
		sources = append(sources, "document")
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
		h.writeMessageError(w, http.StatusBadRequest, "MISSING_MEDIA_SOURCE", "At least one media source must be provided (document, filePath, url, or minioId)", "")
		return
	}
	if len(sources) > 1 {
		h.writeMessageError(w, http.StatusBadRequest, "MULTIPLE_MEDIA_SOURCES", fmt.Sprintf("Only one media source should be provided, received: %v", sources), "")
		return
	}

	if req.Filename == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_FILENAME", "Filename is required for documents", "filename")
		return
	}

	// Validação específica para base64 se fornecido
	if req.Document != "" && !strings.HasPrefix(req.Document, "data:") {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidBase64, "Document must be in base64 data URL format", "document")
		return
	}

	// Busca informações da sessão para resposta
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, entity.ErrorCodeSessionNotFound, "Session not found", "sessionID")
		return
	}

	// Envia a mensagem de documento usando o novo sistema multi-source
	err = h.service.SendDocumentMessageMultiSource(session.Session, req.Phone, req.Document, req.FilePath, req.URL, req.MinioID, req.Filename, req.MimeType)
	if err != nil {
		// Trata diferentes tipos de erro
		if strings.Contains(err.Error(), "not connected") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeSessionOffline, "Session is not connected", "sessionID")
			return
		}
		if strings.Contains(err.Error(), "invalid recipient") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Invalid phone number format", "phone")
			return
		}
		if strings.Contains(err.Error(), "failed to process") || strings.Contains(err.Error(), "arquivo não encontrado") || strings.Contains(err.Error(), "erro ao fazer download") {
			// Log do erro real para debug
			log.Error().
				Err(err).
				Str("session", session.Session).
				Str("phone", req.Phone).
				Str("url", req.URL).
				Str("file_path", req.FilePath).
				Msg("Erro detalhado no envio de documento")
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidBase64, "Invalid document format or data", "document")
			return
		}
		if strings.Contains(err.Error(), "failed to upload") {
			h.writeMessageError(w, http.StatusInternalServerError, entity.ErrorCodeUploadFailed, "Failed to upload document to WhatsApp", "")
			return
		}

		h.writeMessageError(w, http.StatusInternalServerError, entity.ErrorCodeSendFailed, "Failed to send document message", "")
		return
	}

	// Resposta de sucesso padronizada
	response := entity.MessageResponse{
		Success:   true,
		Message:   "Document message sent successfully",
		Timestamp: time.Now(),
		ID:        req.ID,
		Details: &entity.Details{
			Phone:       req.Phone,
			Type:        entity.MessageTypeDocument,
			Status:      "sent",
			SentAt:      time.Now(),
			SessionName: session.Session,
			MediaInfo: &entity.MediaInfo{
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
	var req entity.SendAudioMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload", "")
		return
	}

	// Validações robustas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	if len(req.Phone) < 10 || len(req.Phone) > 15 {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number must be between 10 and 15 digits", "phone")
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
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidBase64, "Audio must be in base64 data URL format (data:audio/...)", "audio")
		return
	}

	// Busca informações da sessão para resposta
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, entity.ErrorCodeSessionNotFound, "Session not found", "sessionID")
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
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeSessionOffline, "Session is not connected", "sessionID")
			return
		}
		if strings.Contains(err.Error(), "invalid recipient") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Invalid phone number format", "phone")
			return
		}
		if strings.Contains(err.Error(), "failed to process") || strings.Contains(err.Error(), "arquivo não encontrado") || strings.Contains(err.Error(), "erro ao fazer download") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidBase64, "Invalid audio format or data", "audio")
			return
		}
		if strings.Contains(err.Error(), "failed to upload") {
			h.writeMessageError(w, http.StatusInternalServerError, entity.ErrorCodeUploadFailed, "Failed to upload audio to WhatsApp", "")
			return
		}
		if strings.Contains(err.Error(), "apenas uma fonte") || strings.Contains(err.Error(), "nenhuma fonte") {
			h.writeMessageError(w, http.StatusBadRequest, "INVALID_MEDIA_SOURCE", err.Error(), "")
			return
		}

		h.writeMessageError(w, http.StatusInternalServerError, entity.ErrorCodeSendFailed, "Failed to send audio message", "")
		return
	}

	// Resposta de sucesso padronizada
	response := entity.MessageResponse{
		Success:   true,
		Message:   "Audio message sent successfully",
		Timestamp: time.Now(),
		ID:        req.ID,
		Details: &entity.Details{
			Phone:       req.Phone,
			Type:        entity.MessageTypeAudio,
			Status:      "sent",
			SentAt:      time.Now(),
			SessionName: session.Session,
			MediaInfo: &entity.MediaInfo{
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
	var req entity.SendImageMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload", "")
		return
	}

	// Validações robustas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	if len(req.Phone) < 10 || len(req.Phone) > 15 {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number must be between 10 and 15 digits", "phone")
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
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidBase64, "Image must be in base64 data URL format (data:image/...)", "image")
		return
	}

	// Busca informações da sessão para resposta
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, entity.ErrorCodeSessionNotFound, "Session not found", "sessionID")
		return
	}

	// Envia a mensagem de imagem usando o novo sistema multi-source
	err = h.service.SendImageMessageMultiSource(session.Session, req.Phone, req.Image, req.FilePath, req.URL, req.MinioID, req.Caption, req.MimeType)
	if err != nil {
		// Trata diferentes tipos de erro
		if strings.Contains(err.Error(), "not connected") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeSessionOffline, "Session is not connected", "sessionID")
			return
		}
		if strings.Contains(err.Error(), "invalid recipient") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Invalid phone number format", "phone")
			return
		}
		if strings.Contains(err.Error(), "failed to process") || strings.Contains(err.Error(), "arquivo não encontrado") || strings.Contains(err.Error(), "erro ao fazer download") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidBase64, "Invalid image source or data", "image")
			return
		}
		if strings.Contains(err.Error(), "failed to upload") {
			h.writeMessageError(w, http.StatusInternalServerError, entity.ErrorCodeUploadFailed, "Failed to upload image to WhatsApp", "")
			return
		}
		if strings.Contains(err.Error(), "apenas uma fonte") || strings.Contains(err.Error(), "nenhuma fonte") {
			h.writeMessageError(w, http.StatusBadRequest, "INVALID_MEDIA_SOURCE", err.Error(), "")
			return
		}

		h.writeMessageError(w, http.StatusInternalServerError, entity.ErrorCodeSendFailed, "Failed to send image message", "")
		return
	}

	// Resposta de sucesso padronizada
	response := entity.MessageResponse{
		Success:   true,
		Message:   "Image message sent successfully",
		Timestamp: time.Now(),
		ID:        req.ID,
		Details: &entity.Details{
			Phone:       req.Phone,
			Type:        entity.MessageTypeImage,
			Status:      "sent",
			SentAt:      time.Now(),
			SessionName: session.Session,
			MediaInfo: &entity.MediaInfo{
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
	var req entity.SendStickerMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload", "")
		return
	}

	// Validações robustas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	if len(req.Phone) < 10 || len(req.Phone) > 15 {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number must be between 10 and 15 digits", "phone")
		return
	}

	if req.Sticker == "" {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidBase64, "Sticker data is required", "sticker")
		return
	}

	// Valida formato base64 para sticker (deve ser WebP)
	if !strings.HasPrefix(req.Sticker, "data:image/webp") {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeUnsupportedMime, "Sticker must be in WebP format (data:image/webp;base64,...)", "sticker")
		return
	}

	// Busca informações da sessão para resposta
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, entity.ErrorCodeSessionNotFound, "Session not found", "sessionID")
		return
	}

	// Envia a mensagem de sticker
	err = h.service.SendStickerMessage(session.Session, req.Phone, req.Sticker)
	if err != nil {
		// Trata diferentes tipos de erro
		if strings.Contains(err.Error(), "not connected") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeSessionOffline, "Session is not connected", "sessionID")
			return
		}
		if strings.Contains(err.Error(), "invalid recipient") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Invalid phone number format", "phone")
			return
		}
		if strings.Contains(err.Error(), "failed to process") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidBase64, "Invalid sticker format or data", "sticker")
			return
		}
		if strings.Contains(err.Error(), "failed to upload") {
			h.writeMessageError(w, http.StatusInternalServerError, entity.ErrorCodeUploadFailed, "Failed to upload sticker to WhatsApp", "")
			return
		}

		h.writeMessageError(w, http.StatusInternalServerError, entity.ErrorCodeSendFailed, "Failed to send sticker message", "")
		return
	}

	// Resposta de sucesso padronizada
	response := entity.MessageResponse{
		Success:   true,
		Message:   "Sticker message sent successfully",
		Timestamp: time.Now(),
		ID:        req.ID,
		Details: &entity.Details{
			Phone:       req.Phone,
			Type:        entity.MessageTypeSticker,
			Status:      "sent",
			SentAt:      time.Now(),
			SessionName: session.Session,
			MediaInfo: &entity.MediaInfo{
				MimeType: entity.MimeTypeImageWebP,
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
	response := entity.SessionResponse{
		Success: false,
		Message: message,
	}

	if err != nil {
		response.Error = err.Error()
		log.Error().Err(err).Str("message", message).Msg("Handler error")
	}

	h.writeJSONResponse(w, statusCode, response)
}

// SendLocationMessage envia uma mensagem de localização
// POST /message/{sessionID}/send/location - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) SendLocationMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_SESSION", "Session ID is required", "sessionID")
		return
	}

	// Decodifica a requisição
	var req entity.SendLocationMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload", "")
		return
	}

	// Validações básicas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	// Valida formato do telefone
	if len(req.Phone) < 10 || len(req.Phone) > 15 {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number must be between 10 and 15 digits", "phone")
		return
	}

	// Validações de coordenadas
	if req.Latitude == 0 {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_LATITUDE", "Latitude is required", "latitude")
		return
	}

	if req.Longitude == 0 {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_LONGITUDE", "Longitude is required", "longitude")
		return
	}

	// Valida faixa de coordenadas
	if req.Latitude < -90 || req.Latitude > 90 {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_LATITUDE", "Latitude must be between -90 and 90", "latitude")
		return
	}

	if req.Longitude < -180 || req.Longitude > 180 {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_LONGITUDE", "Longitude must be between -180 and 180", "longitude")
		return
	}

	// Busca informações da sessão para resposta
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, entity.ErrorCodeSessionNotFound, "Session not found", "sessionID")
		return
	}

	// Envia a mensagem de localização
	err = h.service.SendLocationMessage(session.Session, req.Phone, req.Latitude, req.Longitude, req.Name)
	if err != nil {
		// Trata diferentes tipos de erro
		if strings.Contains(err.Error(), "not connected") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeSessionOffline, "Session is not connected", "sessionID")
			return
		}
		if strings.Contains(err.Error(), "invalid recipient") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Invalid phone number format", "phone")
			return
		}

		h.writeMessageError(w, http.StatusInternalServerError, entity.ErrorCodeSendFailed, "Failed to send location message", "")
		return
	}

	// Resposta de sucesso padronizada
	response := entity.MessageResponse{
		Success:   true,
		Message:   "Location message sent successfully",
		Timestamp: time.Now(),
		ID:        req.ID,
		Details: &entity.Details{
			Phone:       req.Phone,
			Type:        entity.MessageTypeLocation,
			Status:      "sent",
			SentAt:      time.Now(),
			SessionName: session.Session,
			MediaInfo: &entity.MediaInfo{
				Dimensions: fmt.Sprintf("%.6f,%.6f", req.Latitude, req.Longitude),
			},
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// SendContactMessage envia uma mensagem de contato
// POST /message/{sessionID}/send/contact - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) SendContactMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_SESSION", "Session ID is required", "sessionID")
		return
	}

	// Decodifica a requisição
	var req entity.SendContactMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload", "")
		return
	}

	// Validações básicas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	// Valida formato do telefone
	if len(req.Phone) < 10 || len(req.Phone) > 15 {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number must be between 10 and 15 digits", "phone")
		return
	}

	// Validações específicas do contato
	if req.Name == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_NAME", "Contact name is required", "name")
		return
	}

	if req.Vcard == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_VCARD", "vCard data is required", "vcard")
		return
	}

	// Validação básica do formato vCard
	if !strings.Contains(req.Vcard, "BEGIN:VCARD") || !strings.Contains(req.Vcard, "END:VCARD") {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_VCARD", "Invalid vCard format", "vcard")
		return
	}

	// Busca informações da sessão para resposta
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, entity.ErrorCodeSessionNotFound, "Session not found", "sessionID")
		return
	}

	// Envia a mensagem de contato
	err = h.service.SendContactMessage(session.Session, req.Phone, req.Name, req.Vcard)
	if err != nil {
		// Trata diferentes tipos de erro
		if strings.Contains(err.Error(), "not connected") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeSessionOffline, "Session is not connected", "sessionID")
			return
		}
		if strings.Contains(err.Error(), "invalid recipient") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Invalid phone number format", "phone")
			return
		}

		h.writeMessageError(w, http.StatusInternalServerError, entity.ErrorCodeSendFailed, "Failed to send contact message", "")
		return
	}

	// Resposta de sucesso padronizada
	response := entity.MessageResponse{
		Success:   true,
		Message:   "Contact message sent successfully",
		Timestamp: time.Now(),
		ID:        req.ID,
		Details: &entity.Details{
			Phone:       req.Phone,
			Type:        entity.MessageTypeContact,
			Status:      "sent",
			SentAt:      time.Now(),
			SessionName: session.Session,
			MediaInfo: &entity.MediaInfo{
				Filename: req.Name, // Nome do contato
			},
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// ReactToMessage reage a uma mensagem existente
// POST /message/{sessionID}/react - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) ReactToMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_SESSION", "Session ID is required", "sessionID")
		return
	}

	// Decodifica a requisição
	var req entity.ReactToMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload", "")
		return
	}

	// Validações básicas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	// Valida formato do telefone
	if len(req.Phone) < 10 || len(req.Phone) > 15 {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number must be between 10 and 15 digits", "phone")
		return
	}

	// Validações específicas da reação
	if req.MessageID == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_MESSAGE_ID", "Message ID is required", "messageId")
		return
	}

	if req.Reaction == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_REACTION", "Reaction is required", "reaction")
		return
	}

	// Busca informações da sessão para resposta
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, entity.ErrorCodeSessionNotFound, "Session not found", "sessionID")
		return
	}

	// Envia a reação
	err = h.service.ReactToMessage(session.Session, req.Phone, req.MessageID, req.Reaction)
	if err != nil {
		// Trata diferentes tipos de erro
		if strings.Contains(err.Error(), "not connected") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeSessionOffline, "Session is not connected", "sessionID")
			return
		}
		if strings.Contains(err.Error(), "invalid recipient") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Invalid phone number format", "phone")
			return
		}

		h.writeMessageError(w, http.StatusInternalServerError, entity.ErrorCodeSendFailed, "Failed to send reaction", "")
		return
	}

	// Resposta de sucesso padronizada
	response := entity.MessageResponse{
		Success:   true,
		Message:   "Reaction sent successfully",
		Timestamp: time.Now(),
		ID:        req.ID,
		Details: &entity.Details{
			Phone:       req.Phone,
			Type:        entity.MessageTypeReaction,
			Status:      "sent",
			SentAt:      time.Now(),
			SessionName: session.Session,
			MediaInfo: &entity.MediaInfo{
				Filename: req.Reaction, // Emoji da reação
			},
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// SendVideoMessage envia uma mensagem de vídeo
// POST /message/{sessionID}/send/video - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) SendVideoMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_SESSION", "Session ID is required", "sessionID")
		return
	}

	// Decodifica a requisição
	var req entity.SendVideoMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload", "")
		return
	}

	// Validações básicas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	// Valida formato do telefone
	if len(req.Phone) < 10 || len(req.Phone) > 15 {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number must be between 10 and 15 digits", "phone")
		return
	}

	// Valida que pelo menos uma fonte de mídia foi fornecida
	sources := []string{}
	if req.Video != "" {
		sources = append(sources, "video")
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
		h.writeMessageError(w, http.StatusBadRequest, "MISSING_MEDIA_SOURCE", "At least one media source must be provided (video, filePath, url, or minioId)", "")
		return
	}
	if len(sources) > 1 {
		h.writeMessageError(w, http.StatusBadRequest, "MULTIPLE_MEDIA_SOURCES", fmt.Sprintf("Only one media source should be provided, received: %v", sources), "")
		return
	}

	// Validação específica para base64 se fornecido
	if req.Video != "" && !strings.HasPrefix(req.Video, "data:") {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidBase64, "Video must be in base64 data URL format", "video")
		return
	}

	// Busca informações da sessão para resposta
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, entity.ErrorCodeSessionNotFound, "Session not found", "sessionID")
		return
	}

	// Envia a mensagem de vídeo usando o novo sistema multi-source
	err = h.service.SendVideoMessageMultiSource(session.Session, req.Phone, req.Video, req.FilePath, req.URL, req.MinioID, req.Caption, req.MimeType, req.JPEGThumbnail)
	if err != nil {
		// Trata diferentes tipos de erro
		if strings.Contains(err.Error(), "not connected") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeSessionOffline, "Session is not connected", "sessionID")
			return
		}
		if strings.Contains(err.Error(), "invalid recipient") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Invalid phone number format", "phone")
			return
		}
		if strings.Contains(err.Error(), "failed to process") || strings.Contains(err.Error(), "arquivo não encontrado") || strings.Contains(err.Error(), "erro ao fazer download") {
			// Log do erro real para debug
			log.Error().
				Err(err).
				Str("session", session.Session).
				Str("phone", req.Phone).
				Str("url", req.URL).
				Str("file_path", req.FilePath).
				Msg("Erro detalhado no envio de vídeo")
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidBase64, "Invalid video format or data", "video")
			return
		}

		h.writeMessageError(w, http.StatusInternalServerError, entity.ErrorCodeSendFailed, "Failed to send video message", "")
		return
	}

	// Resposta de sucesso padronizada
	response := entity.MessageResponse{
		Success:   true,
		Message:   "Video message sent successfully",
		Timestamp: time.Now(),
		ID:        req.ID,
		Details: &entity.Details{
			Phone:       req.Phone,
			Type:        entity.MessageTypeVideo,
			Status:      "sent",
			SentAt:      time.Now(),
			SessionName: session.Session,
			MediaInfo: &entity.MediaInfo{
				Filename: req.Caption, // Usando filename para armazenar a caption
			},
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// EditMessage edita uma mensagem de texto existente
// POST /message/{sessionID}/edit - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) EditMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_SESSION", "Session ID is required", "sessionID")
		return
	}

	// Decodifica a requisição
	var req entity.EditMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload", "")
		return
	}

	// Validações básicas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	// Valida formato do telefone
	if len(req.Phone) < 10 || len(req.Phone) > 15 {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number must be between 10 and 15 digits", "phone")
		return
	}

	// Validações específicas da edição
	if req.MessageID == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_MESSAGE_ID", "Message ID is required", "messageId")
		return
	}

	if req.NewText == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_TEXT", "New text is required", "newText")
		return
	}

	// Busca informações da sessão para resposta
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, entity.ErrorCodeSessionNotFound, "Session not found", "sessionID")
		return
	}

	// Edita a mensagem
	err = h.service.EditMessage(session.Session, req.Phone, req.MessageID, req.NewText)
	if err != nil {
		// Trata diferentes tipos de erro
		if strings.Contains(err.Error(), "not connected") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeSessionOffline, "Session is not connected", "sessionID")
			return
		}
		if strings.Contains(err.Error(), "invalid recipient") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Invalid phone number format", "phone")
			return
		}

		h.writeMessageError(w, http.StatusInternalServerError, entity.ErrorCodeSendFailed, "Failed to edit message", "")
		return
	}

	// Resposta de sucesso padronizada
	response := entity.MessageResponse{
		Success:   true,
		Message:   "Message edited successfully",
		Timestamp: time.Now(),
		ID:        req.ID,
		Details: &entity.Details{
			Phone:       req.Phone,
			Type:        entity.MessageTypeEdit,
			Status:      "edited",
			SentAt:      time.Now(),
			SessionName: session.Session,
			MediaInfo: &entity.MediaInfo{
				Filename: req.NewText, // Novo texto
			},
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// SendPollMessage envia uma mensagem de enquete (apenas para grupos)
// POST /message/{sessionID}/send/poll - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) SendPollMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_SESSION", "Session ID is required", "sessionID")
		return
	}

	// Decodifica a requisição
	var req entity.SendPollMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload", "")
		return
	}

	// Validações básicas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	// Valida formato do telefone (deve ser um grupo)
	if !strings.Contains(req.Phone, "@g.us") && !strings.Contains(req.Phone, "-") {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_GROUP", "Polls can only be sent to groups", "phone")
		return
	}

	// Validações específicas da enquete
	if req.Header == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_HEADER", "Poll header is required", "header")
		return
	}

	if len(req.Options) < 2 {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_OPTIONS", "Poll must have at least 2 options", "options")
		return
	}

	if len(req.Options) > 12 {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_OPTIONS", "Poll cannot have more than 12 options", "options")
		return
	}

	// Valida se as opções não estão vazias
	for i, option := range req.Options {
		if strings.TrimSpace(option) == "" {
			h.writeMessageError(w, http.StatusBadRequest, "INVALID_OPTIONS", fmt.Sprintf("Option %d cannot be empty", i+1), "options")
			return
		}
	}

	// Busca informações da sessão para resposta
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, entity.ErrorCodeSessionNotFound, "Session not found", "sessionID")
		return
	}

	// Envia a enquete
	err = h.service.SendPollMessage(session.Session, req.Phone, req.Header, req.Options, req.MaxSelections)
	if err != nil {
		// Trata diferentes tipos de erro
		if strings.Contains(err.Error(), "not connected") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeSessionOffline, "Session is not connected", "sessionID")
			return
		}
		if strings.Contains(err.Error(), "invalid recipient") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Invalid phone number format", "phone")
			return
		}
		if strings.Contains(err.Error(), "only be sent to groups") {
			h.writeMessageError(w, http.StatusBadRequest, "INVALID_GROUP", "Polls can only be sent to groups", "phone")
			return
		}

		h.writeMessageError(w, http.StatusInternalServerError, entity.ErrorCodeSendFailed, "Failed to send poll message", "")
		return
	}

	// Resposta de sucesso padronizada
	response := entity.MessageResponse{
		Success:   true,
		Message:   "Poll message sent successfully",
		Timestamp: time.Now(),
		ID:        req.ID,
		Details: &entity.Details{
			Phone:       req.Phone,
			Type:        entity.MessageTypePoll,
			Status:      "sent",
			SentAt:      time.Now(),
			SessionName: session.Session,
			MediaInfo: &entity.MediaInfo{
				Filename: fmt.Sprintf("%s (%d options)", req.Header, len(req.Options)),
			},
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// SendListMessage envia uma mensagem de lista interativa
// POST /message/{sessionID}/send/list - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) SendListMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_SESSION", "Session ID is required", "sessionID")
		return
	}

	// Decodifica a requisição
	var req entity.SendListMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload", "")
		return
	}

	// Validações básicas
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number is required", "phone")
		return
	}

	// Valida formato do telefone
	if len(req.Phone) < 10 || len(req.Phone) > 15 {
		h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Phone number must be between 10 and 15 digits", "phone")
		return
	}

	// Validações específicas da lista
	if req.Header == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_HEADER", "List header is required", "header")
		return
	}

	if req.ButtonText == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_BUTTON_TEXT", "Button text is required", "buttonText")
		return
	}

	if len(req.Sections) == 0 {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_SECTIONS", "List must have at least one section", "sections")
		return
	}

	// Valida seções e itens
	totalItems := 0
	for i, section := range req.Sections {
		if len(section.Rows) == 0 {
			h.writeMessageError(w, http.StatusBadRequest, "INVALID_SECTIONS", fmt.Sprintf("Section %d must have at least one item", i+1), "sections")
			return
		}

		for j, item := range section.Rows {
			if strings.TrimSpace(item.Title) == "" {
				h.writeMessageError(w, http.StatusBadRequest, "INVALID_SECTIONS", fmt.Sprintf("Item %d in section %d must have a title", j+1, i+1), "sections")
				return
			}
			if strings.TrimSpace(item.RowID) == "" {
				h.writeMessageError(w, http.StatusBadRequest, "INVALID_SECTIONS", fmt.Sprintf("Item %d in section %d must have a rowId", j+1, i+1), "sections")
				return
			}
			totalItems++
		}
	}

	// Limite máximo de itens (WhatsApp permite até 10 seções com até 10 itens cada)
	if totalItems > 100 {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_SECTIONS", "List cannot have more than 100 total items", "sections")
		return
	}

	// Busca informações da sessão para resposta
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, entity.ErrorCodeSessionNotFound, "Session not found", "sessionID")
		return
	}

	// Envia a lista
	err = h.service.SendListMessage(session.Session, req.Phone, req.Header, req.Body, req.Footer, req.ButtonText, req.Sections)
	if err != nil {
		// Trata diferentes tipos de erro
		if strings.Contains(err.Error(), "not connected") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeSessionOffline, "Session is not connected", "sessionID")
			return
		}
		if strings.Contains(err.Error(), "invalid recipient") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Invalid phone number format", "phone")
			return
		}

		h.writeMessageError(w, http.StatusInternalServerError, entity.ErrorCodeSendFailed, "Failed to send list message", "")
		return
	}

	// Resposta de sucesso padronizada
	response := entity.MessageResponse{
		Success:   true,
		Message:   "List message sent successfully",
		Timestamp: time.Now(),
		ID:        req.ID,
		Details: &entity.Details{
			Phone:       req.Phone,
			Type:        entity.MessageTypeList,
			Status:      "sent",
			SentAt:      time.Now(),
			SessionName: session.Session,
			MediaInfo: &entity.MediaInfo{
				Filename: fmt.Sprintf("%s (%d sections, %d items)", req.Header, len(req.Sections), totalItems),
			},
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// writeMessageError escreve uma resposta de erro específica para mensagens
func (h *SessionHandler) writeMessageError(w http.ResponseWriter, statusCode int, code, message, field string) {
	response := entity.MessageResponse{
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

// ============================================================================
// UPLOAD MULTIPART HANDLERS
// ============================================================================

// processMultipartFile processa um arquivo de upload multipart
func (h *SessionHandler) processMultipartFile(file multipart.File, header *multipart.FileHeader, messageType entity.MessageType) ([]byte, string, error) {
	// Limita o tamanho do arquivo baseado no tipo
	var maxSize int64
	switch messageType {
	case entity.MessageTypeImage:
		maxSize = entity.MaxImageSize
	case entity.MessageTypeAudio:
		maxSize = entity.MaxAudioSize
	case entity.MessageTypeDocument:
		maxSize = entity.MaxDocumentSize
	case entity.MessageTypeVideo:
		maxSize = entity.MaxDocumentSize // Vídeos usam o mesmo limite de documentos
	default:
		return nil, "", fmt.Errorf("tipo de mensagem não suportado: %s", messageType)
	}

	// Verifica o tamanho do arquivo
	if header.Size > maxSize {
		return nil, "", fmt.Errorf("arquivo muito grande: %d bytes (máximo: %d bytes)", header.Size, maxSize)
	}

	// Lê o arquivo completo
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, "", fmt.Errorf("erro ao ler arquivo: %w", err)
	}

	// Detecta o tipo MIME
	mimeType := http.DetectContentType(data)

	// Valida o tipo MIME baseado no tipo de mensagem
	if err := h.validateMimeType(mimeType, messageType); err != nil {
		return nil, "", err
	}

	return data, mimeType, nil
}

// validateMimeType valida se o tipo MIME é permitido para o tipo de mensagem
func (h *SessionHandler) validateMimeType(mimeType string, messageType entity.MessageType) error {
	switch messageType {
	case entity.MessageTypeImage:
		if !strings.HasPrefix(mimeType, "image/") {
			return fmt.Errorf("tipo de arquivo inválido para imagem: %s", mimeType)
		}
	case entity.MessageTypeAudio:
		if !strings.HasPrefix(mimeType, "audio/") {
			return fmt.Errorf("tipo de arquivo inválido para áudio: %s", mimeType)
		}
	case entity.MessageTypeVideo:
		if !strings.HasPrefix(mimeType, "video/") {
			return fmt.Errorf("tipo de arquivo inválido para vídeo: %s", mimeType)
		}
	case entity.MessageTypeDocument:
		// Documentos podem ter vários tipos MIME
		allowedTypes := []string{
			"application/pdf",
			"application/msword",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/vnd.ms-excel",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			"text/plain",
			"application/zip",
			"application/x-rar-compressed",
		}

		for _, allowedType := range allowedTypes {
			if mimeType == allowedType {
				return nil
			}
		}
		return fmt.Errorf("tipo de documento não suportado: %s", mimeType)
	}
	return nil
}

// getFileExtension retorna a extensão de arquivo baseada no tipo MIME
func getFileExtension(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "audio/mpeg":
		return ".mp3"
	case "audio/ogg":
		return ".ogg"
	case "audio/wav":
		return ".wav"
	case "video/mp4":
		return ".mp4"
	case "video/avi":
		return ".avi"
	case "video/quicktime":
		return ".mov"
	case "application/pdf":
		return ".pdf"
	case "application/msword":
		return ".doc"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ".docx"
	case "text/plain":
		return ".txt"
	default:
		return ".bin"
	}
}

// convertToBase64DataURL converte dados binários para data URL base64
func convertToBase64DataURL(data []byte, mimeType string) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
}

// SendImageMessageMultipart envia uma mensagem de imagem via upload multipart
// POST /message/{sessionID}/send/image/upload - Aceita tanto sessionID quanto sessionName
func (h *SessionHandler) SendImageMessageMultipart(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_SESSION", "Session ID is required", "sessionID")
		return
	}

	// Limita o tamanho da requisição
	r.Body = http.MaxBytesReader(w, r.Body, entity.MaxImageSize+1024*1024) // +1MB para outros campos do form

	// Parse do multipart form
	if err := r.ParseMultipartForm(entity.MaxImageSize); err != nil {
		h.writeMessageError(w, http.StatusBadRequest, "FORM_TOO_LARGE", "Uploaded file is too large", "file")
		return
	}

	// Obtém o arquivo do form
	file, header, err := r.FormFile("file")
	if err != nil {
		h.writeMessageError(w, http.StatusBadRequest, "MISSING_FILE", "File is required", "file")
		return
	}
	defer file.Close()

	// Processa o arquivo
	data, mimeType, err := h.processMultipartFile(file, header, entity.MessageTypeImage)
	if err != nil {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_FILE", err.Error(), "file")
		return
	}

	// Obtém outros campos do form
	phone := r.FormValue("phone")
	if phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, "MISSING_PHONE", "Phone number is required", "phone")
		return
	}

	caption := r.FormValue("caption")
	filename := header.Filename
	if filename == "" {
		filename = "image" + getFileExtension(mimeType)
	}

	// Busca a sessão
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found", "sessionID")
		return
	}

	// Converte dados para base64 para usar o sistema existente
	base64Data := convertToBase64DataURL(data, mimeType)

	// Envia a mensagem usando o sistema existente
	err = h.service.SendImageMessageMultiSource(session.Session, phone, base64Data, "", "", "", caption, mimeType)
	if err != nil {
		if strings.Contains(err.Error(), "not connected") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeSessionOffline, "Session is not connected", "sessionID")
			return
		}
		if strings.Contains(err.Error(), "invalid recipient") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Invalid phone number format", "phone")
			return
		}
		h.writeMessageError(w, http.StatusInternalServerError, "SEND_FAILED", "Failed to send image message", "")
		return
	}

	response := entity.MessageResponse{
		Success:   true,
		Message:   "Image message sent successfully via upload",
		Timestamp: time.Now(),
		Details: &entity.Details{
			Phone:       phone,
			Type:        entity.MessageTypeImage,
			Status:      "sent",
			SentAt:      time.Now(),
			SessionName: session.Session,
			MediaInfo: &entity.MediaInfo{
				Filename:     filename,
				MimeType:     mimeType,
				OriginalSize: int64(len(data)),
			},
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// SendMediaMessage envia uma mensagem de mídia com suporte a múltiplas fontes
// POST /message/{sessionID}/send/media
// Suporta: MediaID (MinIO), Base64, URL, Upload direto (multipart)
func (h *SessionHandler) SendMediaMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if sessionID == "" {
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_SESSION", "Session ID is required", "sessionID")
		return
	}

	// Detectar tipo de content (JSON ou multipart)
	contentType := r.Header.Get("Content-Type")
	isMultipart := strings.HasPrefix(contentType, "multipart/form-data")

	var req entity.SendMediaMessageRequest
	var file multipart.File
	var header *multipart.FileHeader

	if isMultipart {
		// Processar multipart form
		if err := r.ParseMultipartForm(100 << 20); err != nil { // 100MB max
			h.writeMessageError(w, http.StatusBadRequest, "INVALID_MULTIPART", "Invalid multipart form", "body")
			return
		}

		// Extrair campos do form
		req.Phone = r.FormValue("phone")
		req.Caption = r.FormValue("caption")
		req.MessageType = r.FormValue("messageType")
		req.Filename = r.FormValue("filename")

		// Extrair arquivo
		var err error
		file, header, err = r.FormFile("file")
		if err != nil {
			h.writeMessageError(w, http.StatusBadRequest, "MISSING_FILE", "File is required for multipart upload", "file")
			return
		}
		defer file.Close()
	} else {
		// Parse do JSON request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.writeMessageError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON format", "body")
			return
		}
	}

	// Validar campos obrigatórios
	if req.Phone == "" {
		h.writeMessageError(w, http.StatusBadRequest, "MISSING_PHONE", "Phone number is required", "phone")
		return
	}

	// Buscar a sessão
	session, err := h.service.GetSessionByID(sessionID)
	if err != nil {
		h.writeMessageError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found", "sessionID")
		return
	}

	log.Info().
		Str("session_id", sessionID).
		Str("phone", req.Phone).
		Bool("has_media_id", req.MediaID != "").
		Bool("has_base64", req.Base64 != "").
		Bool("has_url", req.URL != "").
		Bool("has_file", file != nil).
		Msg("Processando envio de mídia multi-source")

	// Criar MediaSourceProcessor
	mediaService := media.NewMediaService()

	// Inicializar MinIO client
	minioClient, err := storage.InitializeMinIO()
	if err != nil {
		h.writeMessageError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Storage service unavailable", "")
		return
	}

	processor := media.NewMediaSourceProcessor(h.mediaRepo, minioClient, mediaService)

	// Processar mídia usando o processador unificado
	processed, err := processor.ProcessRequest(&req, file, header)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao processar mídia")
		h.writeMessageError(w, http.StatusBadRequest, "MEDIA_PROCESSING_ERROR", err.Error(), "")
		return
	}

	// Converter dados para base64 para usar sistema existente
	base64Data := convertToBase64DataURL(processed.Data, processed.MimeType)

	// Enviar mensagem usando o sistema existente baseado no tipo detectado
	var sendErr error
	switch processed.MessageType {
	case entity.MessageTypeImage:
		sendErr = h.service.SendImageMessageMultiSource(session.Session, req.Phone, base64Data, "", "", "", req.Caption, processed.MimeType)
	case entity.MessageTypeAudio:
		sendErr = h.service.SendAudioMessageMultiSource(session.Session, req.Phone, base64Data, "", "", "")
	case entity.MessageTypeVideo:
		// Para vídeo, precisamos de um thumbnail JPEG (vazio por enquanto)
		var emptyThumbnail []byte
		sendErr = h.service.SendVideoMessageMultiSource(session.Session, req.Phone, base64Data, "", "", "", req.Caption, processed.MimeType, emptyThumbnail)
	case entity.MessageTypeDocument:
		sendErr = h.service.SendDocumentMessageMultiSource(session.Session, req.Phone, base64Data, "", "", "", processed.Filename, processed.MimeType)
	case entity.MessageTypeSticker:
		// Sticker usa o método básico, não tem MultiSource
		sendErr = h.service.SendStickerMessage(session.Session, req.Phone, base64Data)
	default:
		h.writeMessageError(w, http.StatusBadRequest, "INVALID_MESSAGE_TYPE", "Unsupported message type", "messageType")
		return
	}

	if sendErr != nil {
		if strings.Contains(sendErr.Error(), "not connected") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeSessionOffline, "Session is not connected", "sessionID")
			return
		}
		if strings.Contains(sendErr.Error(), "invalid recipient") {
			h.writeMessageError(w, http.StatusBadRequest, entity.ErrorCodeInvalidPhone, "Invalid phone number format", "phone")
			return
		}
		h.writeMessageError(w, http.StatusInternalServerError, "SEND_FAILED", "Failed to send media message", "")
		return
	}

	// Criar resposta unificada usando a nova estrutura
	response := entity.NewSendMediaMessageResponse(
		req.Phone,
		session.Session,
		processed,
		"", // WhatsApp message ID (não disponível no sistema atual)
		"", // WhatsApp direct path (não disponível no sistema atual)
		"", // WhatsApp URL (não disponível no sistema atual)
	)

	log.Info().
		Str("session_name", session.Session).
		Str("phone", req.Phone).
		Str("message_type", string(processed.MessageType)).
		Str("source", processed.Source).
		Str("filename", processed.Filename).
		Int64("size", processed.Size).
		Dur("processing_time", processed.ProcessingTime).
		Msg("Mensagem de mídia enviada com sucesso")

	h.writeJSONResponse(w, http.StatusOK, response)
}
