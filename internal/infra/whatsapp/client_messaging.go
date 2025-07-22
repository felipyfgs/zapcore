package whatsapp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"zapcore/internal/domain/whatsapp"
	"zapcore/internal/shared/media"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// MessageSender gerencia envio de mensagens WhatsApp
type MessageSender struct {
	client *WhatsAppClient
}

// NewMessageSender cria novo gerenciador de mensagens
func NewMessageSender(client *WhatsAppClient) *MessageSender {
	return &MessageSender{client: client}
}

// SendTextMessage envia mensagem de texto
func (ms *MessageSender) SendTextMessage(ctx context.Context, req *whatsapp.SendTextRequest) (*whatsapp.MessageResponse, error) {
	ms.client.logger.Debug().
		Str("session_id", req.SessionID.String()).
		Str("to_jid", req.ToJID).
		Int("content_length", len(req.Content)).
		Msg("SendTextMessage chamado")

	client, err := ms.getClient(req.SessionID)
	if err != nil {
		ms.client.logger.Error().Err(err).Str("session_id", req.SessionID.String()).Msg("Erro ao obter cliente")
		return nil, err
	}

	jid, err := ms.parseJID(req.ToJID)
	if err != nil {
		ms.client.logger.Error().Err(err).Str("to_jid", req.ToJID).Msg("Erro ao fazer parse do JID")
		return nil, fmt.Errorf("JID inválido: %w", err)
	}

	message := ms.buildTextMessage(req.Content, req.ReplyToID)

	resp, err := client.SendMessage(ctx, jid, message)
	if err != nil {
		return nil, fmt.Errorf("erro ao enviar mensagem: %w", err)
	}

	return &whatsapp.MessageResponse{
		MessageID: resp.ID,
		Status:    "sent",
		Timestamp: resp.Timestamp.Unix(),
	}, nil
}

// SendImageMessage envia imagem
func (ms *MessageSender) SendImageMessage(ctx context.Context, req *whatsapp.SendImageRequest) (*whatsapp.MessageResponse, error) {
	client, err := ms.getClient(req.SessionID)
	if err != nil {
		return nil, err
	}

	jid, err := ms.parseJID(req.ToJID)
	if err != nil {
		return nil, fmt.Errorf("JID inválido: %w", err)
	}

	// Obter e validar dados da imagem
	imageData, err := ms.getAndValidateMediaData(ctx, req.ImageData, req.ImageURL, req.Base64Data, req.MimeType, "image")
	if err != nil {
		return nil, err
	}

	// Fazer upload da imagem
	uploaded, err := client.Upload(ctx, imageData, whatsmeow.MediaImage)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer upload da imagem: %w", err)
	}

	// Criar mensagem de imagem
	message := ms.buildImageMessage(uploaded, req.MimeType, req.Caption, imageData, req.ReplyToID)

	resp, err := client.SendMessage(ctx, jid, message)
	if err != nil {
		return nil, fmt.Errorf("erro ao enviar imagem: %w", err)
	}

	return &whatsapp.MessageResponse{
		MessageID: resp.ID,
		Status:    "sent",
		Timestamp: resp.Timestamp.Unix(),
	}, nil
}

// SendAudioMessage envia áudio
func (ms *MessageSender) SendAudioMessage(ctx context.Context, req *whatsapp.SendAudioRequest) (*whatsapp.MessageResponse, error) {
	client, err := ms.getClient(req.SessionID)
	if err != nil {
		return nil, err
	}

	jid, err := ms.parseJID(req.ToJID)
	if err != nil {
		return nil, fmt.Errorf("JID inválido: %w", err)
	}

	// Obter e validar dados do áudio
	audioData, err := ms.getAndValidateMediaData(ctx, req.AudioData, req.AudioURL, req.Base64Data, req.MimeType, "audio")
	if err != nil {
		return nil, err
	}

	// Fazer upload do áudio
	uploaded, err := client.Upload(ctx, audioData, whatsmeow.MediaAudio)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer upload do áudio: %w", err)
	}

	// Criar mensagem de áudio
	message := ms.buildAudioMessage(uploaded, req.MimeType, audioData, req.ReplyToID)

	resp, err := client.SendMessage(ctx, jid, message)
	if err != nil {
		return nil, fmt.Errorf("erro ao enviar áudio: %w", err)
	}

	return &whatsapp.MessageResponse{
		MessageID: resp.ID,
		Status:    "sent",
		Timestamp: resp.Timestamp.Unix(),
	}, nil
}

// SendVideoMessage envia vídeo
func (ms *MessageSender) SendVideoMessage(ctx context.Context, req *whatsapp.SendVideoRequest) (*whatsapp.MessageResponse, error) {
	client, err := ms.getClient(req.SessionID)
	if err != nil {
		return nil, err
	}

	jid, err := ms.parseJID(req.ToJID)
	if err != nil {
		return nil, fmt.Errorf("JID inválido: %w", err)
	}

	// Obter e validar dados do vídeo
	videoData, err := ms.getAndValidateMediaData(ctx, req.VideoData, req.VideoURL, req.Base64Data, req.MimeType, "video")
	if err != nil {
		return nil, err
	}

	// Fazer upload do vídeo
	uploaded, err := client.Upload(ctx, videoData, whatsmeow.MediaVideo)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer upload do vídeo: %w", err)
	}

	// Criar mensagem de vídeo
	message := ms.buildVideoMessage(uploaded, req.MimeType, req.Caption, videoData, req.ReplyToID)

	resp, err := client.SendMessage(ctx, jid, message)
	if err != nil {
		return nil, fmt.Errorf("erro ao enviar vídeo: %w", err)
	}

	return &whatsapp.MessageResponse{
		MessageID: resp.ID,
		Status:    "sent",
		Timestamp: resp.Timestamp.Unix(),
	}, nil
}

// SendDocumentMessage envia documento
func (ms *MessageSender) SendDocumentMessage(ctx context.Context, req *whatsapp.SendDocumentRequest) (*whatsapp.MessageResponse, error) {
	client, err := ms.getClient(req.SessionID)
	if err != nil {
		return nil, err
	}

	jid, err := ms.parseJID(req.ToJID)
	if err != nil {
		return nil, fmt.Errorf("JID inválido: %w", err)
	}

	// Obter e validar dados do documento
	documentData, err := ms.getAndValidateMediaData(ctx, req.DocumentData, req.DocumentURL, req.Base64Data, req.MimeType, "document")
	if err != nil {
		return nil, err
	}

	// Fazer upload do documento
	uploaded, err := client.Upload(ctx, documentData, whatsmeow.MediaDocument)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer upload do documento: %w", err)
	}

	// Criar mensagem de documento
	message := ms.buildDocumentMessage(uploaded, req.MimeType, req.FileName, documentData, req.ReplyToID)

	resp, err := client.SendMessage(ctx, jid, message)
	if err != nil {
		return nil, fmt.Errorf("erro ao enviar documento: %w", err)
	}

	return &whatsapp.MessageResponse{
		MessageID: resp.ID,
		Status:    "sent",
		Timestamp: resp.Timestamp.Unix(),
	}, nil
}

// SendStickerMessage envia sticker
func (ms *MessageSender) SendStickerMessage(ctx context.Context, req *whatsapp.SendStickerRequest) (*whatsapp.MessageResponse, error) {
	client, err := ms.getClient(req.SessionID)
	if err != nil {
		return nil, err
	}

	jid, err := ms.parseJID(req.ToJID)
	if err != nil {
		return nil, fmt.Errorf("JID inválido: %w", err)
	}

	// Obter e validar dados do sticker
	stickerData, err := ms.getAndValidateMediaData(ctx, req.StickerData, req.StickerURL, req.Base64Data, req.MimeType, "sticker")
	if err != nil {
		return nil, err
	}

	// Fazer upload do sticker
	uploaded, err := client.Upload(ctx, stickerData, whatsmeow.MediaImage)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer upload do sticker: %w", err)
	}

	// Criar mensagem de sticker
	message := ms.buildStickerMessage(uploaded, req.MimeType, stickerData, req.ReplyToID)

	resp, err := client.SendMessage(ctx, jid, message)
	if err != nil {
		return nil, fmt.Errorf("erro ao enviar sticker: %w", err)
	}

	return &whatsapp.MessageResponse{
		MessageID: resp.ID,
		Status:    "sent",
		Timestamp: resp.Timestamp.Unix(),
	}, nil
}

// getClient obtém cliente whatsmeow para sessão
func (ms *MessageSender) getClient(sessionID uuid.UUID) (*whatsmeow.Client, error) {
	ms.client.clientsMutex.RLock()
	client, exists := ms.client.clients[sessionID]
	ms.client.clientsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("cliente não encontrado para sessão %s", sessionID.String())
	}

	if !client.IsConnected() {
		return nil, fmt.Errorf("cliente não está conectado para sessão %s", sessionID.String())
	}

	return client, nil
}

// parseJID faz parse do JID
func (ms *MessageSender) parseJID(jidStr string) (types.JID, error) {
	if jidStr == "" {
		return types.JID{}, fmt.Errorf("JID não pode estar vazio")
	}

	// Se não contém @, assumir que é um número de telefone
	if !strings.Contains(jidStr, "@") {
		// Remover caracteres não numéricos
		phoneNumber := strings.ReplaceAll(jidStr, "+", "")
		phoneNumber = strings.ReplaceAll(phoneNumber, "-", "")
		phoneNumber = strings.ReplaceAll(phoneNumber, " ", "")
		phoneNumber = strings.ReplaceAll(phoneNumber, "(", "")
		phoneNumber = strings.ReplaceAll(phoneNumber, ")", "")

		if len(phoneNumber) < 10 {
			return types.JID{}, fmt.Errorf("número de telefone muito curto: %s", phoneNumber)
		}

		jidStr = phoneNumber + "@s.whatsapp.net"
	}

	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return types.JID{}, fmt.Errorf("erro ao fazer parse do JID %s: %w", jidStr, err)
	}

	return jid, nil
}

// buildTextMessage constrói mensagem de texto
func (ms *MessageSender) buildTextMessage(content, replyToID string) *waProto.Message {
	message := &waProto.Message{
		Conversation: proto.String(content),
	}

	// Adicionar contexto de resposta se especificado
	if replyToID != "" {
		message.ExtendedTextMessage = &waProto.ExtendedTextMessage{
			Text: proto.String(content),
			ContextInfo: &waProto.ContextInfo{
				StanzaID: proto.String(replyToID),
			},
		}
		message.Conversation = nil
	}

	return message
}

// buildImageMessage constrói mensagem de imagem
func (ms *MessageSender) buildImageMessage(uploaded whatsmeow.UploadResponse, mimeType, caption string, data []byte, replyToID string) *waProto.Message {
	imageMsg := &waProto.ImageMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(mimeType),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(data))),
		Caption:       proto.String(caption),
	}

	message := &waProto.Message{
		ImageMessage: imageMsg,
	}

	if replyToID != "" {
		imageMsg.ContextInfo = &waProto.ContextInfo{
			StanzaID: proto.String(replyToID),
		}
	}

	return message
}

// buildAudioMessage constrói mensagem de áudio
func (ms *MessageSender) buildAudioMessage(uploaded whatsmeow.UploadResponse, mimeType string, data []byte, replyToID string) *waProto.Message {
	audioMsg := &waProto.AudioMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(mimeType),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(data))),
	}

	message := &waProto.Message{
		AudioMessage: audioMsg,
	}

	if replyToID != "" {
		audioMsg.ContextInfo = &waProto.ContextInfo{
			StanzaID: proto.String(replyToID),
		}
	}

	return message
}

// buildVideoMessage constrói mensagem de vídeo
func (ms *MessageSender) buildVideoMessage(uploaded whatsmeow.UploadResponse, mimeType, caption string, data []byte, replyToID string) *waProto.Message {
	videoMsg := &waProto.VideoMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(mimeType),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(data))),
		Caption:       proto.String(caption),
	}

	message := &waProto.Message{
		VideoMessage: videoMsg,
	}

	if replyToID != "" {
		videoMsg.ContextInfo = &waProto.ContextInfo{
			StanzaID: proto.String(replyToID),
		}
	}

	return message
}

// buildDocumentMessage constrói mensagem de documento
func (ms *MessageSender) buildDocumentMessage(uploaded whatsmeow.UploadResponse, mimeType, fileName string, data []byte, replyToID string) *waProto.Message {
	docMsg := &waProto.DocumentMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(mimeType),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(data))),
		FileName:      proto.String(fileName),
	}

	message := &waProto.Message{
		DocumentMessage: docMsg,
	}

	if replyToID != "" {
		docMsg.ContextInfo = &waProto.ContextInfo{
			StanzaID: proto.String(replyToID),
		}
	}

	return message
}

// buildStickerMessage constrói mensagem de sticker
func (ms *MessageSender) buildStickerMessage(uploaded whatsmeow.UploadResponse, mimeType string, data []byte, replyToID string) *waProto.Message {
	stickerMsg := &waProto.StickerMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(mimeType),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(data))),
	}

	message := &waProto.Message{
		StickerMessage: stickerMsg,
	}

	if replyToID != "" {
		stickerMsg.ContextInfo = &waProto.ContextInfo{
			StanzaID: proto.String(replyToID),
		}
	}

	return message
}

// getAndValidateMediaData obtém e valida dados de mídia
func (ms *MessageSender) getAndValidateMediaData(ctx context.Context, data io.Reader, url, base64Data, mimeType, mediaType string) ([]byte, error) {
	// Obter dados da mídia
	mediaReader, err := ms.getMediaData(ctx, data, url, base64Data)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter dados da mídia: %w", err)
	}

	// Ler dados
	mediaData, err := io.ReadAll(mediaReader)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler dados da mídia: %w", err)
	}

	// Validar mídia
	if err := ms.validateMedia(mediaData, mimeType, mediaType); err != nil {
		return nil, fmt.Errorf("validação de mídia falhou: %w", err)
	}

	return mediaData, nil
}

// getMediaData obtém dados de mídia de Reader, URL ou base64
func (ms *MessageSender) getMediaData(ctx context.Context, data io.Reader, mediaURL, base64Data string) (io.Reader, error) {
	// Prioridade: Reader > Base64 > URL
	if data != nil {
		return data, nil
	}

	if base64Data != "" {
		return ms.processBase64Data(base64Data)
	}

	if mediaURL != "" {
		return ms.downloadFromURL(ctx, mediaURL)
	}

	return nil, fmt.Errorf("dados de mídia, URL ou base64 devem ser fornecidos")
}

// processBase64Data processa dados em base64
func (ms *MessageSender) processBase64Data(base64Data string) (io.Reader, error) {
	processor := media.NewMediaProcessor()
	processedMedia, err := processor.ProcessBase64Media(base64Data)
	if err != nil {
		return nil, fmt.Errorf("erro ao processar base64: %w", err)
	}
	return processedMedia.GetReader(), nil
}

// downloadFromURL faz download de mídia de uma URL
func (ms *MessageSender) downloadFromURL(ctx context.Context, mediaURL string) (io.Reader, error) {
	// Validar URL
	parsedURL, err := url.Parse(mediaURL)
	if err != nil {
		return nil, fmt.Errorf("URL inválida: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("apenas URLs HTTP/HTTPS são suportadas")
	}

	// Fazer download da mídia
	req, err := http.NewRequestWithContext(ctx, "GET", mediaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer download da mídia: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("erro HTTP %d ao fazer download da mídia", resp.StatusCode)
	}

	return resp.Body, nil
}

// validateMedia valida dados de mídia
func (ms *MessageSender) validateMedia(data []byte, mimeType, mediaType string) error {
	if len(data) == 0 {
		return fmt.Errorf("dados de mídia estão vazios")
	}

	// Validar tamanho máximo (64MB)
	maxSize := 64 * 1024 * 1024
	if len(data) > maxSize {
		return fmt.Errorf("arquivo muito grande: %d bytes (máximo: %d bytes)", len(data), maxSize)
	}

	// Validar MIME type baseado no tipo de mídia
	switch mediaType {
	case "image":
		return ms.validateImageMime(mimeType)
	case "audio":
		return ms.validateAudioMime(mimeType)
	case "video":
		return ms.validateVideoMime(mimeType)
	case "document":
		return ms.validateDocumentMime(mimeType)
	case "sticker":
		return ms.validateStickerMime(mimeType)
	default:
		return fmt.Errorf("tipo de mídia não suportado: %s", mediaType)
	}
}

// validateImageMime valida MIME type de imagem
func (ms *MessageSender) validateImageMime(mimeType string) error {
	validMimes := []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
	if !slices.Contains(validMimes, mimeType) {
		return fmt.Errorf("MIME type de imagem não suportado: %s", mimeType)
	}
	return nil
}

// validateAudioMime valida MIME type de áudio
func (ms *MessageSender) validateAudioMime(mimeType string) error {
	validMimes := []string{"audio/mpeg", "audio/mp4", "audio/amr", "audio/ogg", "audio/wav"}
	if !slices.Contains(validMimes, mimeType) {
		return fmt.Errorf("MIME type de áudio não suportado: %s", mimeType)
	}
	return nil
}

// validateVideoMime valida MIME type de vídeo
func (ms *MessageSender) validateVideoMime(mimeType string) error {
	validMimes := []string{"video/mp4", "video/3gpp", "video/quicktime", "video/avi", "video/mkv"}
	if !slices.Contains(validMimes, mimeType) {
		return fmt.Errorf("MIME type de vídeo não suportado: %s", mimeType)
	}
	return nil
}

// validateDocumentMime valida MIME type de documento
func (ms *MessageSender) validateDocumentMime(mimeType string) error {
	// Documentos podem ter qualquer MIME type
	if mimeType == "" {
		return fmt.Errorf("MIME type de documento não pode estar vazio")
	}
	return nil
}

// validateStickerMime valida MIME type de sticker
func (ms *MessageSender) validateStickerMime(mimeType string) error {
	if mimeType != "image/webp" {
		return fmt.Errorf("stickers devem ser do tipo image/webp, recebido: %s", mimeType)
	}
	return nil
}
