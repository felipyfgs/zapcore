package handlers

import (
	"net/http"

	"zapcore/internal/usecases/message"
	"zapcore/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// MessageHandler gerencia as requisições HTTP para mensagens
type MessageHandler struct {
	sendTextUseCase  *message.SendTextUseCase
	sendMediaUseCase *message.SendMediaUseCase
	logger           *logger.Logger
}

// NewMessageHandler cria uma nova instância do handler
func NewMessageHandler(
	sendTextUseCase *message.SendTextUseCase,
	sendMediaUseCase *message.SendMediaUseCase,
	zeroLogger zerolog.Logger,
) *MessageHandler {
	return &MessageHandler{
		sendTextUseCase:  sendTextUseCase,
		sendMediaUseCase: sendMediaUseCase,
		logger:           logger.NewFromZerolog(zeroLogger),
	}
}

// SendText envia uma mensagem de texto
// @Summary Enviar mensagem de texto
// @Description Envia uma mensagem de texto para o destinatário na sessão especificada
// @Tags messages
// @Accept json
// @Produce json
// @Param sessionID path string true "ID da sessão"
// @Param request body message.SendTextRequest true "Dados da mensagem"
// @Success 200 {object} message.SendTextResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /messages/{sessionID}/send/text [post]
func (h *MessageHandler) SendText(c *gin.Context) {
	sessionIDStr := c.Param("sessionID")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "ID da sessão inválido",
			Message: "O ID da sessão deve ser um UUID válido",
		})
		return
	}

	var req message.SendTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("erro ao fazer bind do JSON")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Dados inválidos",
			Message: err.Error(),
		})
		return
	}

	req.SessionID = sessionID

	response, err := h.sendTextUseCase.Execute(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// SendImage envia uma imagem
// @Summary Enviar imagem
// @Description Envia uma imagem para o destinatário na sessão especificada
// @Tags messages
// @Accept multipart/form-data
// @Produce json
// @Param sessionID path string true "ID da sessão"
// @Param to_jid formData string true "JID do destinatário"
// @Param image formData file true "Arquivo de imagem"
// @Param caption formData string false "Legenda da imagem"
// @Param reply_to_id formData string false "ID da mensagem sendo respondida"
// @Success 200 {object} message.SendMediaResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /messages/{sessionID}/send/image [post]
func (h *MessageHandler) SendImage(c *gin.Context) {
	// h.sendMedia(c, "image") // TODO: Implementar quando MessageType estiver definido
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not implemented",
		"message": "Funcionalidade ainda não implementada",
	})
}

// SendAudio envia um áudio
// @Summary Enviar áudio
// @Description Envia um arquivo de áudio para o destinatário na sessão especificada
// @Tags messages
// @Accept multipart/form-data
// @Produce json
// @Param sessionID path string true "ID da sessão"
// @Param to_jid formData string true "JID do destinatário"
// @Param audio formData file true "Arquivo de áudio"
// @Param reply_to_id formData string false "ID da mensagem sendo respondida"
// @Success 200 {object} message.SendMediaResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /messages/{sessionID}/send/audio [post]
func (h *MessageHandler) SendAudio(c *gin.Context) {
	// h.sendMedia(c, message.MessageTypeAudio) // TODO: Implementar
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not implemented",
		"message": "Funcionalidade ainda não implementada",
	})
}

// SendVideo envia um vídeo
// @Summary Enviar vídeo
// @Description Envia um vídeo para o destinatário na sessão especificada
// @Tags messages
// @Accept multipart/form-data
// @Produce json
// @Param sessionID path string true "ID da sessão"
// @Param to_jid formData string true "JID do destinatário"
// @Param video formData file true "Arquivo de vídeo"
// @Param caption formData string false "Legenda do vídeo"
// @Param reply_to_id formData string false "ID da mensagem sendo respondida"
// @Success 200 {object} message.SendMediaResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /messages/{sessionID}/send/video [post]
func (h *MessageHandler) SendVideo(c *gin.Context) {
	// h.sendMedia(c, message.MessageTypeVideo) // TODO: Implementar
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not implemented",
		"message": "Funcionalidade ainda não implementada",
	})
}

// SendDocument envia um documento
// @Summary Enviar documento
// @Description Envia um documento para o destinatário na sessão especificada
// @Tags messages
// @Accept multipart/form-data
// @Produce json
// @Param sessionID path string true "ID da sessão"
// @Param to_jid formData string true "JID do destinatário"
// @Param document formData file true "Arquivo do documento"
// @Param reply_to_id formData string false "ID da mensagem sendo respondida"
// @Success 200 {object} message.SendMediaResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /messages/{sessionID}/send/document [post]
func (h *MessageHandler) SendDocument(c *gin.Context) {
	// h.sendMedia(c, message.MessageTypeDocument) // TODO: Implementar
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not implemented",
		"message": "Funcionalidade ainda não implementada",
	})
}

// sendMedia é um método auxiliar para envio de mídia - TODO: Implementar quando MessageType estiver definido
func (h *MessageHandler) sendMedia(c *gin.Context, messageType string) { // messageType message.MessageType) {
	sessionIDStr := c.Param("sessionID")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "ID da sessão inválido",
			Message: "O ID da sessão deve ser um UUID válido",
		})
		return
	}

	// Obter arquivo do form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Arquivo não encontrado",
			Message: "É necessário enviar um arquivo",
		})
		return
	}
	defer file.Close()

	// Obter outros campos do form
	toJID := c.PostForm("to_jid")
	if toJID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "JID do destinatário obrigatório",
			Message: "O campo to_jid é obrigatório",
		})
		return
	}

	caption := c.PostForm("caption")
	replyToID := c.PostForm("reply_to_id")

	req := &message.SendMediaRequest{
		SessionID: sessionID,
		ToJID:     toJID,
		// Type:      messageType, // TODO: Corrigir quando MessageType estiver definido
		MediaData: file,
		Caption:   caption,
		FileName:  header.Filename,
		MimeType:  header.Header.Get("Content-Type"),
		ReplyToID: replyToID,
	}

	response, err := h.sendMediaUseCase.Execute(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// handleError trata erros de forma centralizada
func (h *MessageHandler) handleError(c *gin.Context, err error) {
	// Aqui você pode adicionar a lógica de tratamento de erros específicos
	h.logger.Error().Err(err).Msg("Erro interno do servidor")
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error:   "Erro interno do servidor",
		Message: "Ocorreu um erro inesperado",
	})
}
