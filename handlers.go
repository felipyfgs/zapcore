package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// CreateSession - POST /sessions
func (sm *SessionManager) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest

	// Parse do JSON
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{
			Error:   "invalid_request",
			Message: "Erro ao decodificar JSON: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Sanitizar nome se fornecido
	originalName := req.Name
	if req.Name != "" {
		req.Name = SanitizeSessionName(req.Name)

		// Validar nome sanitizado
		if err := ValidateSessionName(req.Name); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, ErrorResponse{
				Error:   "invalid_session_name",
				Message: fmt.Sprintf("Nome inválido '%s': %s", originalName, err.Error()),
				Code:    http.StatusBadRequest,
			})
			return
		}

		// Verificar se já existe uma sessão com esse nome
		if _, exists := sm.GetSessionByName(req.Name); exists {
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, ErrorResponse{
				Error:   "session_name_exists",
				Message: fmt.Sprintf("Já existe uma sessão com o nome '%s'", req.Name),
				Code:    http.StatusConflict,
			})
			return
		}
	}

	// Criar sessão
	session, err := sm.CreateNewSession(req.Name)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse{
			Error:   "session_creation_failed",
			Message: "Erro ao criar sessão: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Resposta de sucesso com informação sobre sanitização se aplicável
	message := "Sessão criada com sucesso"
	if originalName != "" && originalName != req.Name {
		message = fmt.Sprintf("Sessão criada com sucesso. Nome sanitizado de '%s' para '%s'", originalName, req.Name)
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, CreateSessionResponse{
		Session: session.GetInfo(),
		Message: message,
	})
}

// ListSessions - GET /sessions
func (sm *SessionManager) ListSessions(w http.ResponseWriter, r *http.Request) {
	sessions := sm.GetAllSessions()

	render.JSON(w, r, SessionListResponse{
		Sessions: sessions,
		Count:    len(sessions),
	})
}

// GetSession - GET /sessions/{sessionID}
func (sm *SessionManager) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionIdentifier := chi.URLParam(r, "sessionID")

	session, exists := sm.GetSessionByNameOrID(sessionIdentifier)
	if !exists {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, ErrorResponse{
			Error:   "session_not_found",
			Message: "Sessão não encontrada: " + sessionIdentifier,
			Code:    http.StatusNotFound,
		})
		return
	}

	render.JSON(w, r, SessionResponse{
		Session: session.GetInfo(),
	})
}

// DeleteSession - DELETE /sessions/{sessionID}
func (sm *SessionManager) DeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionIdentifier := chi.URLParam(r, "sessionID")

	// Buscar sessão por nome ou ID para obter o ID real
	session, exists := sm.GetSessionByNameOrID(sessionIdentifier)
	if !exists {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, ErrorResponse{
			Error:   "session_not_found",
			Message: "Sessão não encontrada: " + sessionIdentifier,
			Code:    http.StatusNotFound,
		})
		return
	}

	err := sm.RemoveSession(session.ID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse{
			Error:   "session_deletion_failed",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	render.Status(r, http.StatusNoContent)
}

// ConnectSession - POST /sessions/{sessionID}/connect
func (sm *SessionManager) ConnectSession(w http.ResponseWriter, r *http.Request) {
	sessionIdentifier := chi.URLParam(r, "sessionID")

	// Buscar sessão por nome ou ID para obter o ID real
	session, exists := sm.GetSessionByNameOrID(sessionIdentifier)
	if !exists {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, ErrorResponse{
			Error:   "session_not_found",
			Message: "Sessão não encontrada: " + sessionIdentifier,
			Code:    http.StatusNotFound,
		})
		return
	}

	err := sm.ConnectSessionToWhatsApp(session.ID)
	if err != nil {
		status := http.StatusInternalServerError
		errorCode := "connection_failed"

		// Determinar tipo de erro
		if err.Error() == "sessão já está conectada" {
			status = http.StatusConflict
			errorCode = "already_connected"
		}

		render.Status(r, status)
		render.JSON(w, r, ErrorResponse{
			Error:   errorCode,
			Message: err.Error(),
			Code:    status,
		})
		return
	}

	render.JSON(w, r, ConnectResponse{
		SessionID: session.ID,
		Status:    StatusConnecting,
		Message:   "Conexão iniciada com sucesso",
	})
}

// DisconnectSession - POST /sessions/{sessionID}/disconnect
func (sm *SessionManager) DisconnectSession(w http.ResponseWriter, r *http.Request) {
	sessionIdentifier := chi.URLParam(r, "sessionID")

	// Buscar sessão por nome ou ID para obter o ID real
	session, exists := sm.GetSessionByNameOrID(sessionIdentifier)
	if !exists {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, ErrorResponse{
			Error:   "session_not_found",
			Message: "Sessão não encontrada: " + sessionIdentifier,
			Code:    http.StatusNotFound,
		})
		return
	}

	err := sm.DisconnectSessionFromWhatsApp(session.ID)
	if err != nil {
		status := http.StatusInternalServerError
		errorCode := "disconnection_failed"

		// Determinar tipo de erro
		if err.Error() == "sessão não está conectada" {
			status = http.StatusConflict
			errorCode = "not_connected"
		}

		render.Status(r, status)
		render.JSON(w, r, ErrorResponse{
			Error:   errorCode,
			Message: err.Error(),
			Code:    status,
		})
		return
	}

	render.JSON(w, r, DisconnectResponse{
		SessionID: session.ID,
		Status:    StatusDisconnected,
		Message:   "Sessão desconectada com sucesso",
	})
}

// GetSessionStatus - GET /sessions/{sessionID}/status
func (sm *SessionManager) GetSessionStatus(w http.ResponseWriter, r *http.Request) {
	sessionIdentifier := chi.URLParam(r, "sessionID")

	// Buscar sessão por nome ou ID para obter o ID real
	session, exists := sm.GetSessionByNameOrID(sessionIdentifier)
	if !exists {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, ErrorResponse{
			Error:   "session_not_found",
			Message: "Sessão não encontrada: " + sessionIdentifier,
			Code:    http.StatusNotFound,
		})
		return
	}

	status, err := sm.GetSessionStatusByID(session.ID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse{
			Error:   "status_check_failed",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	render.JSON(w, r, StatusResponse{
		SessionID: session.ID,
		Status:    status,
		Message:   "Status obtido com sucesso",
	})
}

// GetQRCode - GET /sessions/{sessionID}/qr
func (sm *SessionManager) GetQRCode(w http.ResponseWriter, r *http.Request) {
	sessionIdentifier := chi.URLParam(r, "sessionID")

	// Buscar sessão por nome ou ID para obter o ID real
	session, exists := sm.GetSessionByNameOrID(sessionIdentifier)
	if !exists {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, ErrorResponse{
			Error:   "session_not_found",
			Message: "Sessão não encontrada: " + sessionIdentifier,
			Code:    http.StatusNotFound,
		})
		return
	}

	qrCode, err := sm.GetQRCodeBySessionID(session.ID)
	if err != nil {
		status := http.StatusInternalServerError
		errorCode := "qr_generation_failed"

		// Determinar tipo de erro
		if err.Error() == "sessão já está logada" {
			status = http.StatusConflict
			errorCode = "already_logged_in"
		}

		render.Status(r, status)
		render.JSON(w, r, ErrorResponse{
			Error:   errorCode,
			Message: err.Error(),
			Code:    status,
		})
		return
	}

	if qrCode == "" {
		render.JSON(w, r, QRResponse{
			SessionID: session.ID,
			QRCode:    "",
			Message:   "QR Code ainda não foi gerado. Tente conectar a sessão primeiro.",
		})
		return
	}

	render.JSON(w, r, QRResponse{
		SessionID: session.ID,
		QRCode:    qrCode,
		Message:   "QR Code obtido com sucesso",
	})
}

// GetSessionWithDevice - GET /sessions/{sessionID}/device
func (sm *SessionManager) GetSessionWithDevice(w http.ResponseWriter, r *http.Request) {
	sessionIdentifier := chi.URLParam(r, "sessionID")

	// Buscar sessão por nome ou ID para obter o ID real
	session, exists := sm.GetSessionByNameOrID(sessionIdentifier)
	if !exists {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, ErrorResponse{
			Error:   "session_not_found",
			Message: "Sessão não encontrada: " + sessionIdentifier,
			Code:    http.StatusNotFound,
		})
		return
	}

	sessionDB, err := sm.db.GetSessionWithDevice(r.Context(), session.ID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse{
			Error:   "database_error",
			Message: "Erro ao buscar dados da sessão: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	response := map[string]interface{}{
		"session_id": sessionDB.ID,
		"name":       sessionDB.Name,
		"status":     sessionDB.Status,
		"device_jid": sessionDB.DeviceJID,
		"device":     sessionDB.Device,
		"message":    "Dados da sessão e device obtidos com sucesso",
	}

	render.JSON(w, r, response)
}

// SendMessage - POST /sessions/{sessionID}/send
func (sm *SessionManager) SendMessage(w http.ResponseWriter, r *http.Request) {
	sessionIdentifier := chi.URLParam(r, "sessionID")

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{
			Error:   "invalid_request",
			Message: "Erro ao decodificar JSON: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Validar campos obrigatórios
	if req.To == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{
			Error:   "missing_to",
			Message: "Campo 'to' é obrigatório",
			Code:    http.StatusBadRequest,
		})
		return
	}

	if req.Message == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{
			Error:   "missing_message",
			Message: "Campo 'message' é obrigatório",
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Buscar sessão
	session, exists := sm.GetSessionByNameOrID(sessionIdentifier)
	if !exists {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, ErrorResponse{
			Error:   "session_not_found",
			Message: "Sessão não encontrada: " + sessionIdentifier,
			Code:    http.StatusNotFound,
		})
		return
	}

	// Verificar se a sessão está conectada
	if !session.Client.IsConnected() {
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, ErrorResponse{
			Error:   "session_not_connected",
			Message: "Sessão não está conectada ao WhatsApp",
			Code:    http.StatusConflict,
		})
		return
	}

	// Enviar mensagem
	err := sm.SendTextMessage(session.ID, req.To, req.Message)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse{
			Error:   "send_failed",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	render.JSON(w, r, SendMessageResponse{
		Success: true,
		Message: "Mensagem enviada com sucesso",
		To:      req.To,
		Content: req.Message,
	})
}
