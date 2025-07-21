package handlers

import (
	"fmt"
	"io"
	"net/http"

	messageEntity "zapcore/internal/domain/message"
	"zapcore/internal/shared/media"
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
	h.logger.Debug().Msg("SendText handler chamado")

	sessionIDStr := c.Param("sessionID")
	h.logger.Debug().Str("session_id_str", sessionIDStr).Msg("Session ID extraído do path")

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id_str", sessionIDStr).Msg("Erro ao fazer parse do session ID")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "ID da sessão inválido",
			Message: "O ID da sessão deve ser um UUID válido",
		})
		return
	}

	h.logger.Debug().Str("session_id", sessionID.String()).Msg("Session ID válido")

	var req message.SendTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("erro ao fazer bind do JSON")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Dados inválidos",
			Message: err.Error(),
		})
		return
	}

	h.logger.Debug().
		Str("to_jid", req.ToJID).
		Int("content_length", len(req.Content)).
		Msg("Dados da requisição processados")

	req.SessionID = sessionID

	h.logger.Debug().Msg("Chamando use case SendText")
	response, err := h.sendTextUseCase.Execute(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// SendImage envia uma imagem
// @Summary Enviar imagem
// @Description Envia uma imagem para o destinatário na sessão especificada via base64 ou URL
// @Tags messages
// @Accept json
// @Produce json
// @Param sessionID path string true "ID da sessão"
// @Param body body MediaRequest true "Dados da imagem (base64 ou URL)"
// @Success 200 {object} message.SendMediaResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /messages/{sessionID}/send/image [post]
func (h *MessageHandler) SendImage(c *gin.Context) {
	h.sendMediaHandler(c, "image")
}

// SendAudio envia um áudio
// @Summary Enviar áudio
// @Description Envia um arquivo de áudio para o destinatário na sessão especificada via base64 ou URL
// @Tags messages
// @Accept json
// @Produce json
// @Param sessionID path string true "ID da sessão"
// @Param body body MediaRequest true "Dados do áudio (base64 ou URL)"
// @Success 200 {object} message.SendMediaResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /messages/{sessionID}/send/audio [post]
func (h *MessageHandler) SendAudio(c *gin.Context) {
	h.sendMediaHandler(c, "audio")
}

// SendVideo envia um vídeo
// @Summary Enviar vídeo
// @Description Envia um vídeo para o destinatário na sessão especificada via base64 ou URL
// @Tags messages
// @Accept json
// @Produce json
// @Param sessionID path string true "ID da sessão"
// @Param body body MediaRequest true "Dados do vídeo (base64 ou URL)"
// @Success 200 {object} message.SendMediaResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /messages/{sessionID}/send/video [post]
func (h *MessageHandler) SendVideo(c *gin.Context) {
	h.sendMediaHandler(c, "video")
}

// SendDocument envia um documento
// @Summary Enviar documento
// @Description Envia um documento para o destinatário na sessão especificada via base64 ou URL
// @Tags messages
// @Accept json
// @Produce json
// @Param sessionID path string true "ID da sessão"
// @Param body body MediaRequest true "Dados do documento (base64 ou URL)"
// @Success 200 {object} message.SendMediaResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /messages/{sessionID}/send/document [post]
func (h *MessageHandler) SendDocument(c *gin.Context) {
	h.sendMediaHandler(c, "document")
}

// SendSticker envia um sticker
// @Summary Enviar sticker
// @Description Envia um sticker para o destinatário na sessão especificada via base64 ou URL
// @Tags messages
// @Accept json
// @Produce json
// @Param sessionID path string true "ID da sessão"
// @Param body body MediaRequest true "Dados do sticker (base64 ou URL)"
// @Success 200 {object} message.SendMediaResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /messages/{sessionID}/send/sticker [post]
func (h *MessageHandler) SendSticker(c *gin.Context) {
	h.sendMediaHandler(c, "sticker")
}

// MediaRequest representa a requisição padronizada para envio de mídia
type MediaRequest struct {
	To      string `json:"to" binding:"required"`
	Base64  string `json:"base64,omitempty"` // Base64 com MIME type
	URL     string `json:"url,omitempty"`    // URL pública HTTP/HTTPS
	File    string `json:"file,omitempty"`   // Caminho local do arquivo
	Caption string `json:"caption,omitempty"`
	ReplyID string `json:"replyId,omitempty"`
}

// sendMediaHandler é um método auxiliar para envio de mídia
func (h *MessageHandler) sendMediaHandler(c *gin.Context, mediaType string) {
	sessionIDStr := c.Param("sessionID")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "ID da sessão inválido",
			Message: "O ID da sessão deve ser um UUID válido",
		})
		return
	}

	// Parse do JSON body
	var req MediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Dados inválidos",
			Message: "Erro ao processar JSON: " + err.Error(),
		})
		return
	}

	// Contar quantos campos de mídia estão preenchidos
	mediaFieldsCount := 0
	if req.Base64 != "" {
		mediaFieldsCount++
	}
	if req.URL != "" {
		mediaFieldsCount++
	}
	if req.File != "" {
		mediaFieldsCount++
	}

	// Validar que apenas um campo de mídia está presente
	if mediaFieldsCount == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Parâmetros obrigatórios",
			Message: "É obrigatório fornecer um dos campos: 'base64', 'url' ou 'file'",
		})
		return
	}

	if mediaFieldsCount > 1 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Conflito de parâmetros",
			Message: "Apenas um dos campos 'base64', 'url' ou 'file' deve ser enviado",
		})
		return
	}

	// Determinar tipo de mensagem baseado no mediaType
	var messageType string
	switch mediaType {
	case "image":
		messageType = "imageMessage"
	case "video":
		messageType = "videoMessage"
	case "audio":
		messageType = "audioMessage"
	case "document":
		messageType = "documentMessage"
	case "sticker":
		messageType = "stickerMessage"
	default:
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Tipo de mídia inválido",
			Message: "Tipo de mídia não suportado",
		})
		return
	}

	// Processar mídia baseado no novo formato
	var mediaData io.Reader
	var fileName, mimeType string
	var mediaURL, filePath string

	if req.Base64 != "" {
		// Processar base64
		mediaData, fileName, mimeType, err = h.processBase64Media(req.Base64)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Erro ao processar base64",
				Message: err.Error(),
			})
			return
		}
	} else if req.URL != "" {
		// Processar URL pública - será tratado pelo use case
		mediaURL = req.URL
		mediaData = nil
	} else if req.File != "" {
		// Processar arquivo local - será tratado pelo use case
		filePath = req.File
		mediaData = nil
	}

	// Criar requisição para o use case
	useCaseReq := &message.SendMediaRequest{
		SessionID: sessionID,
		ToJID:     req.To,
		Type:      messageEntity.MessageType(messageType),
		MediaData: mediaData,
		MediaURL:  mediaURL,
		FilePath:  filePath, // Novo campo para caminho local
		Caption:   req.Caption,
		FileName:  fileName,
		MimeType:  mimeType,
		ReplyToID: req.ReplyID,
	}

	response, err := h.sendMediaUseCase.Execute(c.Request.Context(), useCaseReq)
	if err != nil {
		h.handleMediaError(c, err)
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

// processBase64Media processa dados de mídia em base64
func (h *MessageHandler) processBase64Media(base64Data string) (io.Reader, string, string, error) {
	// Usar o processador de mídia para decodificar base64
	processor := media.NewMediaProcessor()
	processedMedia, err := processor.ProcessBase64Media(base64Data)
	if err != nil {
		return nil, "", "", fmt.Errorf("erro ao processar base64: %w", err)
	}

	return processedMedia.GetReader(), processedMedia.FileName, processedMedia.MimeType, nil
}

// handleMediaError trata erros específicos de mídia
func (h *MessageHandler) handleMediaError(c *gin.Context, err error) {
	// Verificar se é um erro de sessão conhecido
	if err.Error() == "session not found" {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "SESSION_NOT_FOUND",
			Message: "Sessão não encontrada",
		})
		return
	}

	if err.Error() == "session not active" {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "SESSION_NOT_ACTIVE",
			Message: "Sessão não está ativa",
		})
		return
	}

	if err.Error() == "session not connected" {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "SESSION_NOT_CONNECTED",
			Message: "Sessão não está conectada ao WhatsApp",
		})
		return
	}

	// Verificar erros de validação de mídia
	errMsg := err.Error()
	switch {
	case contains(errMsg, "arquivo muito grande"):
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "FILE_TOO_LARGE",
			Message: "Arquivo muito grande para o tipo de mídia",
		})
		return
	case contains(errMsg, "tipo MIME"):
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_MIME_TYPE",
			Message: "Tipo de arquivo não suportado",
		})
		return
	case contains(errMsg, "URL inválida"):
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_URL",
			Message: "URL fornecida é inválida",
		})
		return
	case contains(errMsg, "base64"):
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_BASE64",
			Message: "Dados base64 inválidos",
		})
		return
	case contains(errMsg, "erro ao enviar"):
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   "SEND_FAILED",
			Message: "Falha ao enviar mídia pelo WhatsApp",
		})
		return
	default:
		// Erro genérico
		h.handleError(c, err)
	}
}

// contains verifica se uma string contém outra
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
