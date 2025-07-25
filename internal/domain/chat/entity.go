package chat

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ChatType representa os tipos de chat
type ChatType string

const (
	ChatTypeIndividual ChatType = "individual"
	ChatTypeGroup      ChatType = "group"
)

// Chat representa um chat/conversa do WhatsApp
type Chat struct {
	bun.BaseModel `bun:"table:zapcore_chats,alias:c"`

	ID              uuid.UUID      `bun:"id,pk,type:uuid" json:"id"`
	SessionID       uuid.UUID      `bun:"sessionId,type:uuid,notnull" json:"sessionId"`
	JID             string         `bun:"jid,type:varchar(100),notnull" json:"jid"`
	Name            string         `bun:"name,type:varchar(255)" json:"name,omitempty"`
	Type            ChatType       `bun:"chatType,type:varchar(20),notnull" json:"type"`
	LastMessageTime *time.Time     `bun:"lastMessageTime,type:timestamptz" json:"lastMessageTime,omitempty"`
	MessageCount    int            `bun:"messageCount,type:integer" json:"messageCount"`
	UnreadCount     int            `bun:"unreadCount,type:integer" json:"unreadCount"`
	IsMuted         bool           `bun:"isMuted,type:boolean" json:"isMuted"`
	IsPinned        bool           `bun:"isPinned,type:boolean" json:"isPinned"`
	IsArchived      bool           `bun:"isArchived,type:boolean" json:"isArchived"`
	Metadata        map[string]any `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	CreatedAt       time.Time      `bun:"createdAt,type:timestamptz,notnull" json:"createdAt"`
	UpdatedAt       time.Time      `bun:"updatedAt,type:timestamptz,notnull" json:"updatedAt"`
}

// NewChat cria uma nova instância de Chat
func NewChat(sessionID uuid.UUID, jid string, chatType ChatType) *Chat {
	now := time.Now()
	return &Chat{
		ID:           uuid.New(),
		SessionID:    sessionID,
		JID:          jid,
		Type:         chatType,
		MessageCount: 0,
		UnreadCount:  0,
		IsMuted:      false,
		IsPinned:     false,
		IsArchived:   false,
		CreatedAt:    now,
		UpdatedAt:    now,
		Metadata:     make(map[string]any),
	}
}

// SetName define o nome do chat
func (c *Chat) SetName(name string) {
	c.Name = name
	c.UpdatedAt = time.Now()
}

// UpdateLastMessage atualiza o timestamp da última mensagem
func (c *Chat) UpdateLastMessage(timestamp time.Time) {
	c.LastMessageTime = &timestamp
	c.UpdatedAt = time.Now()
}

// IncrementMessageCount incrementa o contador de mensagens
func (c *Chat) IncrementMessageCount() {
	c.MessageCount++
	c.UpdatedAt = time.Now()
}

// IncrementUnreadCount incrementa o contador de mensagens não lidas
func (c *Chat) IncrementUnreadCount() {
	c.UnreadCount++
	c.UpdatedAt = time.Now()
}

// MarkAsRead marca todas as mensagens como lidas
func (c *Chat) MarkAsRead() {
	c.UnreadCount = 0
	c.UpdatedAt = time.Now()
}

// Mute silencia o chat
func (c *Chat) Mute() {
	c.IsMuted = true
	c.UpdatedAt = time.Now()
}

// Unmute remove o silenciamento do chat
func (c *Chat) Unmute() {
	c.IsMuted = false
	c.UpdatedAt = time.Now()
}

// Pin fixa o chat
func (c *Chat) Pin() {
	c.IsPinned = true
	c.UpdatedAt = time.Now()
}

// Unpin remove a fixação do chat
func (c *Chat) Unpin() {
	c.IsPinned = false
	c.UpdatedAt = time.Now()
}

// Archive arquiva o chat
func (c *Chat) Archive() {
	c.IsArchived = true
	c.UpdatedAt = time.Now()
}

// Unarchive desarquiva o chat
func (c *Chat) Unarchive() {
	c.IsArchived = false
	c.UpdatedAt = time.Now()
}

// SetMetadata define um valor nos metadados
func (c *Chat) SetMetadata(key string, value any) {
	if c.Metadata == nil {
		c.Metadata = make(map[string]any)
	}
	c.Metadata[key] = value
	c.UpdatedAt = time.Now()
}

// GetMetadata obtém um valor dos metadados
func (c *Chat) GetMetadata(key string) (any, bool) {
	if c.Metadata == nil {
		return nil, false
	}
	value, exists := c.Metadata[key]
	return value, exists
}

// IsGroup verifica se o chat é um grupo
func (c *Chat) IsGroup() bool {
	return c.Type == ChatTypeGroup
}

// IsIndividual verifica se o chat é individual
func (c *Chat) IsIndividual() bool {
	return c.Type == ChatTypeIndividual
}

// HasUnreadMessages verifica se há mensagens não lidas
func (c *Chat) HasUnreadMessages() bool {
	return c.UnreadCount > 0
}

// IsActive verifica se o chat está ativo (não arquivado)
func (c *Chat) IsActive() bool {
	return !c.IsArchived
}
