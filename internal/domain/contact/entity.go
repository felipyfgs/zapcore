package contact

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Contact representa um contato do WhatsApp
type Contact struct {
	bun.BaseModel `bun:"table:zapcore_contacts,alias:ct"`

	ID           uuid.UUID      `bun:"id,pk,type:uuid" json:"id"`
	SessionID    uuid.UUID      `bun:"sessionId,type:uuid,notnull" json:"sessionId"`
	JID          string         `bun:"jid,type:varchar(100),notnull" json:"jid"`
	Name         string         `bun:"-" json:"name,omitempty"` // Não existe na migration atual
	PushName     string         `bun:"pushName,type:varchar(255)" json:"pushName,omitempty"`
	BusinessName string         `bun:"businessName,type:varchar(255)" json:"businessName,omitempty"`
	AvatarURL    string         `bun:"avatarUrl,type:varchar(500)" json:"avatarUrl,omitempty"`
	IsBusiness   bool           `bun:"-" json:"isBusiness"` // Não existe na migration atual
	IsGroup      bool           `bun:"isGroup,type:boolean" json:"isGroup"`
	LastSeen     *time.Time     `bun:"lastSeen,type:timestamptz" json:"lastSeen,omitempty"`
	Metadata     map[string]any `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	CreatedAt    time.Time      `bun:"createdAt,type:timestamptz,notnull" json:"createdAt"`
	UpdatedAt    time.Time      `bun:"updatedAt,type:timestamptz,notnull" json:"updatedAt"`
}

// NewContact cria uma nova instância de Contact
func NewContact(sessionID uuid.UUID, jid string) *Contact {
	now := time.Now()
	return &Contact{
		ID:         uuid.New(),
		SessionID:  sessionID,
		JID:        jid,
		IsBusiness: false,
		IsGroup:    false,
		CreatedAt:  now,
		UpdatedAt:  now,
		Metadata:   make(map[string]any),
	}
}

// SetName define o nome do contato
func (c *Contact) SetName(name string) {
	c.Name = name
	c.UpdatedAt = time.Now()
}

// SetPushName define o nome de exibição do WhatsApp
func (c *Contact) SetPushName(pushName string) {
	c.PushName = pushName
	c.UpdatedAt = time.Now()
}

// SetBusinessName define o nome do negócio
func (c *Contact) SetBusinessName(businessName string) {
	c.BusinessName = businessName
	c.UpdatedAt = time.Now()
}

// SetAvatarURL define a URL do avatar
func (c *Contact) SetAvatarURL(avatarURL string) {
	c.AvatarURL = avatarURL
	c.UpdatedAt = time.Now()
}

// MarkAsBusiness marca o contato como conta business
func (c *Contact) MarkAsBusiness() {
	c.IsBusiness = true
	c.UpdatedAt = time.Now()
}

// MarkAsGroup marca o contato como grupo
func (c *Contact) MarkAsGroup() {
	c.IsGroup = true
	c.UpdatedAt = time.Now()
}

// UpdateLastSeen atualiza o último acesso do contato
func (c *Contact) UpdateLastSeen(lastSeen time.Time) {
	c.LastSeen = &lastSeen
	c.UpdatedAt = time.Now()
}

// SetMetadata define um valor nos metadados
func (c *Contact) SetMetadata(key string, value any) {
	if c.Metadata == nil {
		c.Metadata = make(map[string]any)
	}
	c.Metadata[key] = value
	c.UpdatedAt = time.Now()
}

// GetMetadata obtém um valor dos metadados
func (c *Contact) GetMetadata(key string) (any, bool) {
	if c.Metadata == nil {
		return nil, false
	}
	value, exists := c.Metadata[key]
	return value, exists
}

// GetDisplayName retorna o nome de exibição preferencial
func (c *Contact) GetDisplayName() string {
	if c.Name != "" {
		return c.Name
	}
	if c.PushName != "" {
		return c.PushName
	}
	if c.BusinessName != "" {
		return c.BusinessName
	}
	return c.JID
}

// IsOnline verifica se o contato está online (baseado no LastSeen)
func (c *Contact) IsOnline() bool {
	if c.LastSeen == nil {
		return false
	}
	// Considera online se visto nos últimos 5 minutos
	return time.Since(*c.LastSeen) < 5*time.Minute
}
