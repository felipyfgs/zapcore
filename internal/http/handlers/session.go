package handlers

import (
	"fmt"
	"net/http"
	"regexp"

	"zapcore/internal/usecases/session"
	"zapcore/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Regex para validação de nomes de sessão
var sessionNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// SessionHandler gerencia as requisições HTTP para sessões
type SessionHandler struct {
	createUseCase     *session.CreateUseCase
	connectUseCase    *session.ConnectUseCase
	disconnectUseCase *session.DisconnectUseCase
	listUseCase       *session.ListUseCase
	getStatusUseCase  *session.GetStatusUseCase
	logger            *logger.Logger
}

// NewSessionHandler cria uma nova instância do handler
func NewSessionHandler(
	createUseCase *session.CreateUseCase,
	connectUseCase *session.ConnectUseCase,
	disconnectUseCase *session.DisconnectUseCase,
	listUseCase *session.ListUseCase,
	getStatusUseCase *session.GetStatusUseCase,
	zeroLogger zerolog.Logger,
) *SessionHandler {
	return &SessionHandler{
		createUseCase:     createUseCase,
		connectUseCase:    connectUseCase,
		disconnectUseCase: disconnectUseCase,
		listUseCase:       listUseCase,
		getStatusUseCase:  getStatusUseCase,
		logger:            logger.NewFromZerolog(zeroLogger),
	}
}

// validateSessionName valida se o nome da sessão atende aos critérios
func (h *SessionHandler) validateSessionName(name string) error {
	if name == "" {
		return fmt.Errorf("nome da sessão é obrigatório")
	}

	if len(name) < 3 {
		return fmt.Errorf("nome da sessão deve ter pelo menos 3 caracteres")
	}

	if len(name) > 50 {
		return fmt.Errorf("nome da sessão deve ter no máximo 50 caracteres")
	}

	if !sessionNameRegex.MatchString(name) {
		return fmt.Errorf("nome da sessão deve conter apenas letras (a-z, A-Z), números (0-9), hífens (-) e underscores (_). Não são permitidos espaços, acentos ou caracteres especiais")
	}

	return nil
}

// resolveSessionIdentifier resolve um identificador que pode ser UUID ou nome da sessão
func (h *SessionHandler) resolveSessionIdentifier(c *gin.Context, identifier string) (uuid.UUID, error) {
	// Primeiro, tenta interpretar como UUID
	if sessionID, err := uuid.Parse(identifier); err == nil {
		return sessionID, nil
	}

	// Se não for UUID, busca por nome da sessão
	sess, err := h.getStatusUseCase.GetByName(c.Request.Context(), identifier)
	if err != nil {
		return uuid.Nil, fmt.Errorf("sessão não encontrada com identificador '%s'", identifier)
	}

	return sess.ID, nil
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

	// Validar nome da sessão
	if err := h.validateSessionName(req.Name); err != nil {
		h.logger.Error().Err(err).Str("session_name", req.Name).Msg("Nome da sessão inválido")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Nome da sessão inválido",
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
// @Param sessionID path string true "ID ou nome da sessão"
// @Success 200 {object} session.GetStatusResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sessions/{sessionID} [get]
func (h *SessionHandler) GetStatus(c *gin.Context) {
	identifier := c.Param("sessionID")
	sessionID, err := h.resolveSessionIdentifier(c, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Sessão não encontrada",
			Message: err.Error(),
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
// @Param sessionID path string true "ID ou nome da sessão"
// @Success 200 {object} session.ConnectResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sessions/{sessionID}/connect [post]
func (h *SessionHandler) Connect(c *gin.Context) {
	identifier := c.Param("sessionID")
	sessionID, err := h.resolveSessionIdentifier(c, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Sessão não encontrada",
			Message: err.Error(),
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
// @Param sessionID path string true "ID ou nome da sessão"
// @Success 200 {object} session.DisconnectResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sessions/{sessionID}/logout [post]
func (h *SessionHandler) Disconnect(c *gin.Context) {
	identifier := c.Param("sessionID")
	sessionID, err := h.resolveSessionIdentifier(c, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Sessão não encontrada",
			Message: err.Error(),
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
