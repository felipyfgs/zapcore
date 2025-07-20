package message

import (
	"context"
	"fmt"

	"zapcore/internal/domain/message"
	"zapcore/internal/domain/session"
	"zapcore/internal/domain/whatsapp"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// SendTextUseCase representa o caso de uso para enviar mensagem de texto
type SendTextUseCase struct {
	messageRepo    message.Repository
	sessionRepo    session.Repository
	whatsappClient whatsapp.Client
	logger         zerolog.Logger
}

// NewSendTextUseCase cria uma nova instância do caso de uso
func NewSendTextUseCase(
	messageRepo message.Repository,
	sessionRepo session.Repository,
	whatsappClient whatsapp.Client,
	logger zerolog.Logger,
) *SendTextUseCase {
	return &SendTextUseCase{
		messageRepo:    messageRepo,
		sessionRepo:    sessionRepo,
		whatsappClient: whatsappClient,
		logger:         logger,
	}
}

// SendTextRequest representa a requisição para enviar texto
type SendTextRequest struct {
	SessionID uuid.UUID `json:"session_id" validate:"required"`
	ToJID     string    `json:"to_jid" validate:"required"`
	Content   string    `json:"content" validate:"required,min=1,max=4096"`
	ReplyToID string    `json:"reply_to_id,omitempty"`
}

// SendTextResponse representa a resposta do envio de texto
type SendTextResponse struct {
	MessageID  uuid.UUID             `json:"message_id"`
	WhatsAppID string                `json:"whatsapp_id"`
	Status     message.MessageStatus `json:"status"`
	Timestamp  string                `json:"timestamp"`
	Message    string                `json:"message"`
}

// Execute executa o caso de uso de envio de texto
func (uc *SendTextUseCase) Execute(ctx context.Context, req *SendTextRequest) (*SendTextResponse, error) {
	// Verificar se a sessão existe e está conectada
	sess, err := uc.sessionRepo.GetByID(ctx, req.SessionID)
	if err != nil {
		if err == session.ErrSessionNotFound {
			return nil, err
		}
		uc.logger.Error().Err(err).Msg("Erro ao validar sessão")
		return nil, fmt.Errorf("erro interno do servidor")
	}

	if !sess.IsActive {
		return nil, session.ErrSessionNotActive
	}

	if !sess.IsConnected() {
		return nil, session.ErrSessionNotConnected
	}

	// Criar mensagem
	msg := message.NewMessage(req.SessionID, message.MessageTypeText, message.MessageDirectionOutbound)
	msg.ToJID = req.ToJID
	msg.FromJID = sess.JID
	msg.SetContent(req.Content)

	if req.ReplyToID != "" {
		msg.SetReplyTo(req.ReplyToID)
	}

	// Salvar mensagem como pendente
	if err := uc.messageRepo.Create(ctx, msg); err != nil {
		uc.logger.Error().Err(err).Msg("Erro ao salvar mensagem")
		return nil, fmt.Errorf("erro ao salvar mensagem: %w", err)
	}

	// Preparar requisição para WhatsApp
	whatsappReq := &whatsapp.SendTextRequest{
		SessionID: req.SessionID,
		ToJID:     req.ToJID,
		Content:   req.Content,
		ReplyToID: req.ReplyToID,
	}

	// Enviar via WhatsApp
	whatsappResp, err := uc.whatsappClient.SendTextMessage(ctx, whatsappReq)
	if err != nil {
		// Marcar mensagem como falha
		msg.UpdateStatus(message.MessageStatusFailed)
		uc.messageRepo.Update(ctx, msg)

		uc.logger.Error().Err(err).Msg("Erro ao enviar mensagem via WhatsApp")

		return nil, fmt.Errorf("erro ao enviar mensagem: %w", err)
	}

	// Atualizar mensagem com ID do WhatsApp
	msg.MessageID = whatsappResp.MessageID
	msg.UpdateStatus(message.MessageStatusSent)

	if err := uc.messageRepo.Update(ctx, msg); err != nil {
		uc.logger.Error().Err(err).Msg("Erro ao atualizar status da mensagem")
		// Não retorna erro pois a mensagem foi enviada
	}

	// Atualizar último acesso da sessão
	uc.sessionRepo.UpdateLastSeen(ctx, req.SessionID)

	uc.logger.Info().
		Str("session_id", req.SessionID.String()).
		Str("message_id", msg.ID.String()).
		Str("whatsapp_id", whatsappResp.MessageID).
		Str("to_jid", req.ToJID).
		Int("content_length", len(req.Content)).
		Msg("Mensagem de texto enviada com sucesso")

	return &SendTextResponse{
		MessageID:  msg.ID,
		WhatsAppID: whatsappResp.MessageID,
		Status:     msg.Status,
		Timestamp:  msg.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		Message:    "Mensagem enviada com sucesso",
	}, nil
}
