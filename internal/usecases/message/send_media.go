package message

import (
	"context"
	"fmt"
	"io"

	"zapcore/internal/domain/message"
	"zapcore/internal/domain/session"
	"zapcore/internal/domain/whatsapp"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// SendMediaUseCase representa o caso de uso para enviar mídia
type SendMediaUseCase struct {
	messageRepo    message.Repository
	sessionRepo    session.Repository
	whatsappClient whatsapp.Client
	logger         zerolog.Logger
}

// NewSendMediaUseCase cria uma nova instância do caso de uso
func NewSendMediaUseCase(
	messageRepo message.Repository,
	sessionRepo session.Repository,
	whatsappClient whatsapp.Client,
	logger zerolog.Logger,
) *SendMediaUseCase {
	return &SendMediaUseCase{
		messageRepo:    messageRepo,
		sessionRepo:    sessionRepo,
		whatsappClient: whatsappClient,
		logger:         logger,
	}
}

// SendMediaRequest representa a requisição para enviar mídia
type SendMediaRequest struct {
	SessionID uuid.UUID           `json:"session_id" validate:"required"`
	ToJID     string              `json:"to_jid" validate:"required"`
	Type      message.MessageType `json:"type" validate:"required"`
	MediaData io.Reader           `json:"-"`
	MediaURL  string              `json:"media_url,omitempty"`
	Caption   string              `json:"caption,omitempty"`
	FileName  string              `json:"file_name,omitempty"`
	MimeType  string              `json:"mime_type,omitempty"`
	ReplyToID string              `json:"reply_to_id,omitempty"`
}

// SendMediaResponse representa a resposta do envio de mídia
type SendMediaResponse struct {
	MessageID  uuid.UUID             `json:"message_id"`
	WhatsAppID string                `json:"whatsapp_id"`
	Status     message.MessageStatus `json:"status"`
	Timestamp  string                `json:"timestamp"`
	Message    string                `json:"message"`
}

// Execute executa o caso de uso de envio de mídia
func (uc *SendMediaUseCase) Execute(ctx context.Context, req *SendMediaRequest) (*SendMediaResponse, error) {
	// Verificar se a sessão existe e está conectada
	sess, err := uc.sessionRepo.GetByID(ctx, req.SessionID)
	if err != nil {
		if err == session.ErrSessionNotFound {
			return nil, err
		}
		uc.logger.Error().Err(err).Msg("erro ao buscar sessão")
		return nil, fmt.Errorf("erro interno do servidor")
	}

	if !sess.IsActive {
		return nil, session.ErrSessionNotActive
	}

	if !sess.IsConnected() {
		return nil, session.ErrSessionNotConnected
	}

	// Validar tipo de mídia
	if !isValidMediaType(req.Type) {
		return nil, message.ErrInvalidMediaType
	}

	// Criar mensagem
	msg := message.NewMessage(req.SessionID, req.Type, message.MessageDirectionOutbound)
	msg.ToJID = req.ToJID
	msg.FromJID = sess.JID
	
	if req.Caption != "" {
		msg.SetCaption(req.Caption)
	}
	
	if req.ReplyToID != "" {
		msg.SetReplyTo(req.ReplyToID)
	}

	// Salvar mensagem como pendente
	if err := uc.messageRepo.Create(ctx, msg); err != nil {
		uc.logger.Error().Err(err).Msg("erro ao salvar mensagem")
		return nil, fmt.Errorf("erro ao salvar mensagem: %w", err)
	}

	// Enviar via WhatsApp baseado no tipo
	var whatsappResp *whatsapp.MessageResponse
	
	switch req.Type {
	case message.MessageTypeImage:
		whatsappReq := &whatsapp.SendImageRequest{
			SessionID: req.SessionID,
			ToJID:     req.ToJID,
			ImageData: req.MediaData,
			ImageURL:  req.MediaURL,
			Caption:   req.Caption,
			ReplyToID: req.ReplyToID,
			MimeType:  req.MimeType,
			FileName:  req.FileName,
		}
		whatsappResp, err = uc.whatsappClient.SendImageMessage(ctx, whatsappReq)
		
	case message.MessageTypeAudio:
		whatsappReq := &whatsapp.SendAudioRequest{
			SessionID: req.SessionID,
			ToJID:     req.ToJID,
			AudioData: req.MediaData,
			AudioURL:  req.MediaURL,
			ReplyToID: req.ReplyToID,
			MimeType:  req.MimeType,
			FileName:  req.FileName,
		}
		whatsappResp, err = uc.whatsappClient.SendAudioMessage(ctx, whatsappReq)
		
	case message.MessageTypeVideo:
		whatsappReq := &whatsapp.SendVideoRequest{
			SessionID: req.SessionID,
			ToJID:     req.ToJID,
			VideoData: req.MediaData,
			VideoURL:  req.MediaURL,
			Caption:   req.Caption,
			ReplyToID: req.ReplyToID,
			MimeType:  req.MimeType,
			FileName:  req.FileName,
		}
		whatsappResp, err = uc.whatsappClient.SendVideoMessage(ctx, whatsappReq)
		
	case message.MessageTypeDocument:
		whatsappReq := &whatsapp.SendDocumentRequest{
			SessionID:    req.SessionID,
			ToJID:        req.ToJID,
			DocumentData: req.MediaData,
			DocumentURL:  req.MediaURL,
			FileName:     req.FileName,
			ReplyToID:    req.ReplyToID,
			MimeType:     req.MimeType,
		}
		whatsappResp, err = uc.whatsappClient.SendDocumentMessage(ctx, whatsappReq)
		
	case message.MessageTypeSticker:
		whatsappReq := &whatsapp.SendStickerRequest{
			SessionID:   req.SessionID,
			ToJID:       req.ToJID,
			StickerData: req.MediaData,
			StickerURL:  req.MediaURL,
			ReplyToID:   req.ReplyToID,
			MimeType:    req.MimeType,
		}
		whatsappResp, err = uc.whatsappClient.SendStickerMessage(ctx, whatsappReq)
		
	default:
		return nil, message.ErrInvalidMediaType
	}

	if err != nil {
		// Marcar mensagem como falha
		msg.UpdateStatus(message.MessageStatusFailed)
		uc.messageRepo.Update(ctx, msg)
		
		uc.logger.Error().Err(err).Msg("erro ao enviar mídia")
		
		return nil, fmt.Errorf("erro ao enviar mídia: %w", err)
	}

	// Atualizar mensagem com ID do WhatsApp
	msg.MessageID = whatsappResp.MessageID
	msg.UpdateStatus(message.MessageStatusSent)
	
	if err := uc.messageRepo.Update(ctx, msg); err != nil {
		uc.logger.Error().Err(err).Msg("erro ao atualizar mensagem")
	}

	// Atualizar último acesso da sessão
	uc.sessionRepo.UpdateLastSeen(ctx, req.SessionID)

	uc.logger.Info().Msg("mídia enviada com sucesso")

	return &SendMediaResponse{
		MessageID:  msg.ID,
		WhatsAppID: whatsappResp.MessageID,
		Status:     msg.Status,
		Timestamp:  msg.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		Message:    "Mídia enviada com sucesso",
	}, nil
}

// isValidMediaType verifica se o tipo de mídia é válido
func isValidMediaType(msgType message.MessageType) bool {
	validTypes := []message.MessageType{
		message.MessageTypeImage,
		message.MessageTypeAudio,
		message.MessageTypeVideo,
		message.MessageTypeDocument,
		message.MessageTypeSticker,
	}
	
	for _, validType := range validTypes {
		if msgType == validType {
			return true
		}
	}
	
	return false
}

