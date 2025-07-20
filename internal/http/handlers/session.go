package handlers

import (
	"net/http"

	"zapcore/internal/usecases/session"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// SessionHandler gerencia as requisições HTTP para sessões
type SessionHandler struct {
	createUseCase     *session.CreateUseCase
	connectUseCase    *session.ConnectUseCase
	disconnectUseCase *session.DisconnectUseCase
	listUseCase       *session.ListUseCase
	getStatusUseCase  *session.GetStatusUseCase
	logger            zerolog.Logger
}

// NewSessionHandler cria uma nova instância do handler
func NewSessionHandler(
	createUseCase *session.CreateUseCase,
	connectUseCase *session.ConnectUseCase,
	disconnectUseCase *session.DisconnectUseCase,
	listUseCase *session.ListUseCase,
	getStatusUseCase *session.GetStatusUseCase,
	logger zerolog.Logger,
) *SessionHandler {
	return &SessionHandler{
		createUseCase:     createUseCase,
		connectUseCase:    connectUseCase,
		disconnectUseCase: disconnectUseCase,
		listUseCase:       listUseCase,
		getStatusUseCase:  getStatusUseCase,
		logger:            logger,
	}
}

// Create cria uma nova sessão
// @Summary Criar nova sessão
// @Description Cria uma nova sessão do WhatsApp
// @Tags sessions
// @Accept json
// @Produce json
// @Param request body session.CreateRequest true "Dados da sessão"
// @Success 201 {object} session.CreateResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sessions/add [post]
func (h *SessionHandler) Create(c *gin.Context) {
	var req session.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Erro ao fazer bind da requisição")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Dados inválidos",
			Message: err.Error(),
		})
		return
	}

	response, err := h.createUseCase.Execute(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, response)
}

// List lista todas as sessões
// @Summary Listar sessões
// @Description Lista todas as sessões ativas e registradas no sistema
// @Tags sessions
// @Accept json
// @Produce json
// @Param status query string false "Filtrar por status"
// @Param is_active query bool false "Filtrar por ativo"
// @Param limit query int false "Limite de resultados"
// @Param offset query int false "Offset para paginação"
// @Success 200 {object} session.ListResponse
// @Failure 500 {object} ErrorResponse
// @Router /sessions/list [get]
func (h *SessionHandler) List(c *gin.Context) {
	var req session.ListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error().Err(err).Msg("Erro ao fazer bind da query")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Parâmetros inválidos",
			Message: err.Error(),
		})
		return
	}

	response, err := h.listUseCase.Execute(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetStatus obtém o status de uma sessão específica
// @Summary Obter status da sessão
// @Description Retorna as informações detalhadas de uma sessão específica
// @Tags sessions
// @Accept json
// @Produce json
// @Param sessionID path string true "ID da sessão"
// @Success 200 {object} session.GetStatusResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sessions/{sessionID} [get]
func (h *SessionHandler) GetStatus(c *gin.Context) {
	sessionIDStr := c.Param("sessionID")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "ID da sessão inválido",
			Message: "O ID da sessão deve ser um UUID válido",
		})
		return
	}

	req := &session.GetStatusRequest{
		SessionID: sessionID,
	}

	response, err := h.getStatusUseCase.Execute(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// Connect estabelece conexão da sessão com o WhatsApp
// @Summary Conectar sessão
// @Description Estabelece a conexão da sessão com o WhatsApp
// @Tags sessions
// @Accept json
// @Produce json
// @Param sessionID path string true "ID da sessão"
// @Success 200 {object} session.ConnectResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sessions/{sessionID}/connect [post]
func (h *SessionHandler) Connect(c *gin.Context) {
	sessionIDStr := c.Param("sessionID")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "ID da sessão inválido",
			Message: "O ID da sessão deve ser um UUID válido",
		})
		return
	}

	req := &session.ConnectRequest{
		SessionID: sessionID,
	}

	response, err := h.connectUseCase.Execute(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// Disconnect desconecta a sessão do WhatsApp
// @Summary Desconectar sessão
// @Description Desconecta a sessão do WhatsApp, encerrando a comunicação
// @Tags sessions
// @Accept json
// @Produce json
// @Param sessionID path string true "ID da sessão"
// @Success 200 {object} session.DisconnectResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sessions/{sessionID}/logout [post]
func (h *SessionHandler) Disconnect(c *gin.Context) {
	sessionIDStr := c.Param("sessionID")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "ID da sessão inválido",
			Message: "O ID da sessão deve ser um UUID válido",
		})
		return
	}

	req := &session.DisconnectRequest{
		SessionID: sessionID,
	}

	response, err := h.disconnectUseCase.Execute(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// handleError trata erros de forma centralizada
func (h *SessionHandler) handleError(c *gin.Context, err error) {
	// Aqui você pode adicionar a lógica de tratamento de erros específicos
	// baseado nos erros do domain
	h.logger.Error().Err(err).Msg("Erro interno do servidor")
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error:   "Erro interno do servidor",
		Message: "Ocorreu um erro inesperado",
	})
}
