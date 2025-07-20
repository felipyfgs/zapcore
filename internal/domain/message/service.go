package message

import (
	"context"
	"io"

	"github.com/google/uuid"
)

// Service define a interface para o serviço de mensagens
type Service interface {
	// SendText envia uma mensagem de texto
	SendText(ctx context.Context, sessionID uuid.UUID, toJID, content string) (*Message, error)

	// SendMedia envia uma mensagem de mídia
	SendMedia(ctx context.Context, req *SendMediaRequest) (*Message, error)

	// EditMessage edita uma mensagem existente
	EditMessage(ctx context.Context, messageID, newContent string) (*Message, error)

	// GetConversation obtém mensagens de uma conversa
	GetConversation(ctx context.Context, sessionID uuid.UUID, jid string, filters ListFilters) ([]*Message, error)

	// MarkAsRead marca mensagens como lidas
	MarkAsRead(ctx context.Context, sessionID uuid.UUID, messageIDs []string) error

	// GetPendingMessages obtém mensagens pendentes para reenvio
	GetPendingMessages(ctx context.Context, sessionID uuid.UUID) ([]*Message, error)

	// RetryFailedMessages reprocessa mensagens com falha
	RetryFailedMessages(ctx context.Context, sessionID uuid.UUID) error
}

// SendMediaRequest representa uma requisição de envio de mídia
type SendMediaRequest struct {
	SessionID uuid.UUID     `json:"session_id"`
	ToJID     string        `json:"to_jid"`
	Type      MessageType   `json:"type"`
	MediaData io.Reader     `json:"-"`
	MediaURL  string        `json:"media_url,omitempty"`
	Caption   string        `json:"caption,omitempty"`
	FileName  string        `json:"file_name,omitempty"`
	MimeType  string        `json:"mime_type,omitempty"`
	ReplyToID string        `json:"reply_to_id,omitempty"`
}

// MediaManager define a interface para gerenciamento de mídia
type MediaManager interface {
	// Upload faz upload de um arquivo de mídia
	Upload(ctx context.Context, data io.Reader, fileName, mimeType string) (*MediaInfo, error)

	// Download faz download de um arquivo de mídia
	Download(ctx context.Context, mediaID uuid.UUID) (io.ReadCloser, *MediaInfo, error)

	// Delete remove um arquivo de mídia
	Delete(ctx context.Context, mediaID uuid.UUID) error

	// GetInfo obtém informações de um arquivo de mídia
	GetInfo(ctx context.Context, mediaID uuid.UUID) (*MediaInfo, error)

	// GenerateThumbnail gera thumbnail para imagens/vídeos
	GenerateThumbnail(ctx context.Context, mediaID uuid.UUID) (*MediaInfo, error)
}

// MediaInfo representa informações de um arquivo de mídia
type MediaInfo struct {
	ID        uuid.UUID `json:"id"`
	FileName  string    `json:"file_name"`
	MimeType  string    `json:"mime_type"`
	Size      int64     `json:"size"`
	URL       string    `json:"url,omitempty"`
	Thumbnail string    `json:"thumbnail,omitempty"`
	CreatedAt string    `json:"created_at"`
}

