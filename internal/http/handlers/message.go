package handlers

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	messageEntity "zapcore/internal/domain/message"
	"zapcore/internal/shared/media"
	"zapcore/internal/usecases/message"
	"zapcore/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
) *MessageHandler {
	return &MessageHandler{
		sendTextUseCase:  sendTextUseCase,
		sendMediaUseCase: sendMediaUseCase,
		logger:           logger.Get(),
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
		Str("to", req.To).
		Int("text_length", len(req.Text)).
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
	To       string `json:"to" binding:"required"`
	Base64   string `json:"base64,omitempty"`   // Base64 com MIME type
	URL      string `json:"url,omitempty"`      // URL pública HTTP/HTTPS
	File     string `json:"file,omitempty"`     // Caminho local do arquivo
	FileName string `json:"fileName,omitempty"` // Nome do arquivo
	Caption  string `json:"caption,omitempty"`
	ReplyID  string `json:"replyId,omitempty"`
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

	// Parse do body (JSON ou form-data)
	var req MediaRequest

	// Verificar se é form-data ou JSON
	contentType := c.GetHeader("Content-Type")
	h.logger.Debug().
		Str("content_type", contentType).
		Msg("Processando requisição de mídia")

	// Variáveis para armazenar dados do form-data
	var formFile multipart.File
	var formHeader *multipart.FileHeader

	if strings.Contains(contentType, "multipart/form-data") {
		h.logger.Debug().Msg("Processando form-data")
		// Parse form-data
		req.To = c.PostForm("to")
		req.Caption = c.PostForm("caption")
		req.ReplyID = c.PostForm("replyId")

		h.logger.Debug().
			Str("to", req.To).
			Str("caption", req.Caption).
			Msg("Dados do form-data extraídos")

		// Validar campo obrigatório "to"
		if req.To == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Campo 'to' obrigatório",
				Message: "O campo 'to' deve ser fornecido",
			})
			return
		}

		// Processar arquivo enviado
		file, header, err := c.Request.FormFile("media")
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Arquivo de mídia obrigatório",
				Message: "Erro ao processar arquivo: " + err.Error(),
			})
			return
		}

		// Armazenar para uso posterior
		formFile = file
		formHeader = header
		req.FileName = header.Filename

		// Marcar que temos dados de arquivo (não base64)
		req.File = "form-data-file" // Marcador especial para indicar que temos arquivo

	} else {
		// Parse JSON
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Dados inválidos",
				Message: "Erro ao processar JSON: " + err.Error(),
			})
			return
		}
	}

	// Contar quantos campos de mídia estão preenchidos
	mediaFieldsCount := 0
	var mediaSource string

	// Verificar form-data
	if req.File == "form-data-file" {
		mediaFieldsCount++
		mediaSource = "form-data"
	}
	if req.Base64 != "" {
		mediaFieldsCount++
		if mediaSource == "" {
			mediaSource = "base64"
		}
	}
	if req.URL != "" {
		mediaFieldsCount++
		if mediaSource == "" {
			mediaSource = "url"
		}
	}

	// Validar que apenas um campo de mídia está presente
	if mediaFieldsCount == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "MEDIA_REQUIRED",
			Message: "É obrigatório fornecer um dos campos: 'base64', 'url' ou usar form-data",
		})
		return
	}

	if mediaFieldsCount > 1 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "MEDIA_CONFLICT",
			Message: "Apenas um dos campos 'base64', 'url' ou form-data deve ser enviado por vez",
		})
		return
	}

	h.logger.Debug().
		Str("media_source", mediaSource).
		Str("media_type", mediaType).
		Str("base64", req.Base64).
		Str("url", req.URL).
		Str("file", req.File).
		Msg("Validação de mídia aprovada")

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
	var mediaURL string

	h.logger.Debug().
		Str("req_file", req.File).
		Str("req_base64", req.Base64).
		Str("req_url", req.URL).
		Msg("Iniciando processamento de mídia")

	if req.File == "form-data-file" {
		// Usar arquivo já processado do form-data
		if formFile == nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Erro interno",
				Message: "Arquivo form-data não encontrado",
			})
			return
		}
		defer formFile.Close()

		mediaData = formFile
		fileName = formHeader.Filename
		mimeType = h.detectMimeTypeFromFilename(fileName)

	} else if req.Base64 != "" {
		// Validar e processar base64
		if err := h.validateBase64Data(req.Base64); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "INVALID_BASE64",
				Message: err.Error(),
			})
			return
		}

		mediaData, fileName, mimeType, err = h.processBase64Media(req.Base64)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "BASE64_PROCESSING_ERROR",
				Message: err.Error(),
			})
			return
		}
	} else if req.URL != "" {
		// Validar e processar URL pública
		if err := h.validateURL(req.URL); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "INVALID_URL",
				Message: err.Error(),
			})
			return
		}
		mediaURL = req.URL
		mediaData = nil

		// Detectar MIME type baseado na extensão da URL
		mimeType = h.detectMimeTypeFromURL(req.URL)
		h.logger.Debug().
			Str("url", req.URL).
			Str("detected_mime_type", mimeType).
			Msg("MIME type detectado da URL")

		// Se não conseguiu detectar, usar um padrão baseado no tipo de mídia
		if mimeType == "" {
			switch mediaType {
			case "image":
				mimeType = "image/jpeg" // Padrão para imagens
			case "video":
				mimeType = "video/mp4" // Padrão para vídeos
			case "audio":
				mimeType = "audio/mpeg" // Padrão para áudios
			case "document":
				mimeType = "application/pdf" // Padrão para documentos
			}
			h.logger.Debug().
				Str("fallback_mime_type", mimeType).
				Str("media_type", mediaType).
				Msg("Usando MIME type padrão")
		}
	}

	// Criar requisição para o use case
	useCaseReq := &message.SendMediaRequest{
		SessionID:  sessionID,
		ToJID:      req.To,
		Type:       messageEntity.MessageType(messageType),
		MediaData:  mediaData,
		MediaURL:   mediaURL,
		Base64Data: req.Base64, // Passar dados base64 se disponível
		Caption:    req.Caption,
		FileName:   fileName,
		MimeType:   mimeType,
		ReplyToID:  req.ReplyID,
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
	h.logger.Debug().
		Str("base64_prefix", base64Data[:min(50, len(base64Data))]).
		Msg("🔄 Iniciando processamento de base64")

	// Usar o processador de mídia para decodificar base64
	processor := media.NewMediaProcessor()
	processedMedia, err := processor.ProcessBase64Media(base64Data)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("base64_prefix", base64Data[:min(50, len(base64Data))]).
			Msg("❌ Erro ao processar base64")
		return nil, "", "", fmt.Errorf("erro ao processar base64: %w", err)
	}

	h.logger.Debug().
		Str("mime_type", processedMedia.MimeType).
		Str("file_name", processedMedia.FileName).
		Int64("size", processedMedia.Size).
		Msg("✅ Base64 processado com sucesso")

	return processedMedia.GetReader(), processedMedia.FileName, processedMedia.MimeType, nil
}

// detectMimeTypeFromFilename detecta o tipo MIME baseado no nome do arquivo
func (h *MessageHandler) detectMimeTypeFromFilename(filename string) string {
	ext := strings.ToLower(filename)

	// Imagens
	if strings.HasSuffix(ext, ".jpg") || strings.HasSuffix(ext, ".jpeg") {
		return "image/jpeg"
	}
	if strings.HasSuffix(ext, ".png") {
		return "image/png"
	}
	if strings.HasSuffix(ext, ".gif") {
		return "image/gif"
	}
	if strings.HasSuffix(ext, ".webp") {
		return "image/webp"
	}

	// Vídeos
	if strings.HasSuffix(ext, ".mp4") {
		return "video/mp4"
	}
	if strings.HasSuffix(ext, ".avi") {
		return "video/avi"
	}
	if strings.HasSuffix(ext, ".mov") {
		return "video/mov"
	}

	// Áudios
	if strings.HasSuffix(ext, ".mp3") {
		return "audio/mpeg"
	}
	if strings.HasSuffix(ext, ".wav") {
		return "audio/wav"
	}
	if strings.HasSuffix(ext, ".ogg") {
		return "audio/ogg"
	}

	// Documentos
	if strings.HasSuffix(ext, ".pdf") {
		return "application/pdf"
	}
	if strings.HasSuffix(ext, ".doc") {
		return "application/msword"
	}
	if strings.HasSuffix(ext, ".docx") {
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}

	// Padrão
	return "application/octet-stream"
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

// validateBase64Data valida dados em base64
func (h *MessageHandler) validateBase64Data(base64Data string) error {
	if len(base64Data) == 0 {
		return fmt.Errorf("dados base64 não podem estar vazios")
	}

	// Verificar se tem o prefixo data: (data URI scheme)
	if !strings.HasPrefix(base64Data, "data:") {
		return fmt.Errorf("base64 deve começar com 'data:'")
	}

	// Verificar formato: data:mime/type;base64,dados
	parts := strings.Split(base64Data, ",")
	if len(parts) != 2 {
		return fmt.Errorf("formato de data URI inválido")
	}

	// Verificar se é base64 válido (apenas a parte dos dados)
	base64DataOnly := parts[1]
	if len(base64DataOnly)%4 != 0 {
		return fmt.Errorf("dados base64 com tamanho inválido")
	}

	// Tentar decodificar para verificar se é válido (usar o formato completo)
	_, _, err := media.DecodeBase64Media(base64Data)
	if err != nil {
		return fmt.Errorf("dados base64 inválidos: %w", err)
	}

	return nil
}

// validateURL valida uma URL
func (h *MessageHandler) validateURL(url string) error {
	if len(url) == 0 {
		return fmt.Errorf("URL não pode estar vazia")
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("URL deve começar com http:// ou https://")
	}

	if len(url) > 2048 {
		return fmt.Errorf("URL muito longa (máximo 2048 caracteres)")
	}

	return nil
}

// detectMimeTypeFromURL detecta MIME type baseado na extensão da URL
func (h *MessageHandler) detectMimeTypeFromURL(url string) string {
	// Extrair extensão da URL (remover query parameters)
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}

	// Encontrar a última ocorrência de '.' para pegar a extensão
	if idx := strings.LastIndex(url, "."); idx != -1 {
		ext := strings.ToLower(url[idx+1:])
		return h.getMimeTypeFromExtension(ext)
	}

	return ""
}

// getMimeTypeFromExtension retorna MIME type baseado na extensão
func (h *MessageHandler) getMimeTypeFromExtension(ext string) string {
	mimeTypes := map[string]string{
		// Imagens
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"gif":  "image/gif",
		"webp": "image/webp",

		// Áudios
		"mp3": "audio/mpeg",
		"wav": "audio/wav",
		"ogg": "audio/ogg",
		"aac": "audio/aac",
		"m4a": "audio/mp4",

		// Vídeos
		"mp4":  "video/mp4",
		"avi":  "video/x-msvideo",
		"mov":  "video/quicktime",
		"mkv":  "video/x-matroska",
		"webm": "video/webm",

		// Documentos
		"pdf":  "application/pdf",
		"doc":  "application/msword",
		"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"xls":  "application/vnd.ms-excel",
		"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"txt":  "text/plain",
	}

	if mimeType, exists := mimeTypes[ext]; exists {
		return mimeType
	}

	return ""
}
