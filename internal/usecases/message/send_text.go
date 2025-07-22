package message

import (
	"context"
	"fmt"
	"time"

	"zapcore/internal/domain/message"
	"zapcore/internal/domain/session"
	"zapcore/internal/domain/whatsapp"
	"zapcore/pkg/logger"

	"github.com/google/uuid"
)

// SendTextUseCase representa o caso de uso para enviar mensagem de texto
type SendTextUseCase struct {
	messageRepo    message.Repository
	sessionRepo    session.Repository
	whatsappClient whatsapp.Client
	logger         *logger.Logger
}

// NewSendTextUseCase cria uma nova instância do caso de uso
func NewSendTextUseCase(
	messageRepo message.Repository,
	sessionRepo session.Repository,
	whatsappClient whatsapp.Client,
) *SendTextUseCase {
	return &SendTextUseCase{
		messageRepo:    messageRepo,
		sessionRepo:    sessionRepo,
		whatsappClient: whatsappClient,
		logger:         logger.Get(),
	}
}

// SendTextRequest representa a requisição para enviar texto
type SendTextRequest struct {
	SessionID uuid.UUID `json:"sessionId" validate:"required"`
	To        string    `json:"to" validate:"required"`
	Text      string    `json:"text" validate:"required,min=1,max=4096"`
	ReplyID   string    `json:"replyId,omitempty"`
}

// SendTextResponse representa a resposta do envio de texto
type SendTextResponse struct {
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

	uc.logger.Debug().
		Str("session_id", req.SessionID.String()).
		Str("to", req.To).
		Int("text_length", len(req.Text)).
		Msg("Preparando envio de mensagem de texto")

	// Preparar requisição para WhatsApp
	whatsappReq := &whatsapp.SendTextRequest{
		SessionID: req.SessionID,
		ToJID:     req.To,
		Content:   req.Text,
		ReplyToID: req.ReplyID,
	}

	// Enviar via WhatsApp
	whatsappResp, err := uc.whatsappClient.SendTextMessage(ctx, whatsappReq)
	if err != nil {
		uc.logger.Error().Err(err).Msg("Erro ao enviar mensagem via WhatsApp")
		return nil, fmt.Errorf("erro ao enviar mensagem: %w", err)
	}

	// Atualizar último acesso da sessão
	uc.sessionRepo.UpdateLastSeen(ctx, req.SessionID)

	uc.logger.Info().
		Str("session_id", req.SessionID.String()).
		Str("whatsapp_id", whatsappResp.MessageID).
		Str("to", req.To).
		Int("text_length", len(req.Text)).
		Msg("Mensagem de texto enviada com sucesso via WhatsApp")

	return &SendTextResponse{
		WhatsAppID: whatsappResp.MessageID,
		Status:     message.MessageStatusSent,
		Timestamp:  time.Now().Format("2006-01-02T15:04:05Z07:00"),
		Message:    "Mensagem enviada com sucesso",
	}, nil
}
