package message

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// MessageType representa os tipos de mensagem suportados
type MessageType string

const (
	// Tipos básicos de mensagem
	MessageTypeText         MessageType = "textMessage"
	MessageTypeImage        MessageType = "imageMessage"
	MessageTypeVideo        MessageType = "videoMessage"
	MessageTypeAudio        MessageType = "audioMessage"
	MessageTypeDocument     MessageType = "documentMessage"
	MessageTypeSticker      MessageType = "stickerMessage"
	MessageTypeContact      MessageType = "contactMessage"
	MessageTypeLocation     MessageType = "locationMessage"
	MessageTypeLiveLocation MessageType = "liveLocationMessage"
	MessageTypeGif          MessageType = "gifMessage"

	// Tipos interativos
	MessageTypePoll        MessageType = "pollMessage"
	MessageTypePollUpdate  MessageType = "pollUpdateMessage"
	MessageTypeReaction    MessageType = "reactionMessage"
	MessageTypeButtons     MessageType = "buttonsMessage"
	MessageTypeList        MessageType = "listMessage"
	MessageTypeInteractive MessageType = "interactiveMessage"

	// Tipos especiais
	MessageTypeExtendedText MessageType = "extendedTextMessage"
	MessageTypeQuoted       MessageType = "quotedMessage"
	MessageTypeGroupInvite  MessageType = "groupInviteMessage"
	MessageTypeProtocol     MessageType = "protocolMessage"
	MessageTypeEphemeral    MessageType = "ephemeralMessage"
	MessageTypeViewOnce     MessageType = "viewOnceMessage"

	// Tipos de mídia especiais
	MessageTypePtt MessageType = "pttMessage"

	// Tipos mais recentes
	MessageTypeNewsletterAdmin MessageType = "newsletterAdminInviteMessage"
	MessageTypeCallLog         MessageType = "callLogMessage"
	MessageTypeScheduledCall   MessageType = "scheduledCallCreationMessage"
	MessageTypeEvent           MessageType = "eventMessage"
	MessageTypePaymentInvite   MessageType = "paymentInviteMessage"

	// Tipo desconhecido
	MessageTypeUnknown MessageType = "unknownMessage"
)

// MessageDirection representa a direção da mensagem
type MessageDirection string

const (
	MessageDirectionInbound  MessageDirection = "inbound"
	MessageDirectionOutbound MessageDirection = "outbound"
)

// MessageStatus representa o status da mensagem
type MessageStatus string

const (
	MessageStatusPending   MessageStatus = "pending"
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
)

// Message representa uma mensagem do WhatsApp
type Message struct {
	bun.BaseModel `bun:"table:zapcore_messages,alias:m"`

	ID              uuid.UUID        `bun:"id,pk,type:uuid" json:"id"`
	SessionID       uuid.UUID        `bun:"sessionId,type:uuid,notnull" json:"sessionId"`
	MsgID           string           `bun:"msgId,type:varchar(255),notnull" json:"msgId"` // ID único do WhatsApp
	MessageType     MessageType      `bun:"messageType,type:varchar(50),notnull" json:"messageType"`
	Direction       MessageDirection `bun:"direction,type:varchar(20),notnull" json:"direction"`
	Status          MessageStatus    `bun:"status,type:varchar(20),notnull" json:"status"`
	SenderJID       string           `bun:"senderJid,type:varchar(100),notnull" json:"senderJid"`
	ChatJID         string           `bun:"chatJid,type:varchar(100),notnull" json:"chatJid"`
	Content         string           `bun:"content,type:text" json:"content,omitempty"`
	MediaID         *uuid.UUID       `bun:"mediaId,type:uuid" json:"mediaId,omitempty"`
	MediaPath       string           `bun:"mediaPath,type:varchar(500)" json:"mediaPath,omitempty"`
	MediaSize       int64            `bun:"mediaSize,type:bigint" json:"mediaSize,omitempty"`
	MediaMimeType   string           `bun:"mediaMimeType,type:varchar(100)" json:"mediaMimeType,omitempty"`
	MediaFileName   string           `bun:"mediaFileName,type:varchar(255)" json:"mediaFileName,omitempty"`
	Caption         string           `bun:"caption,type:text" json:"caption,omitempty"`
	Timestamp       time.Time        `bun:"timestamp,type:timestamptz,notnull" json:"timestamp"`
	QuotedMessageID string           `bun:"quotedMessageId,type:varchar(255)" json:"quotedMessageId,omitempty"`
	PushName        string           `bun:"pushName,type:varchar(255)" json:"pushName,omitempty"`
	IsFromMe        bool             `bun:"isFromMe,type:boolean" json:"isFromMe"`
	IsGroup         bool             `bun:"isGroup,type:boolean" json:"isGroup"`
	MediaType       string           `bun:"mediaType,type:varchar(50)" json:"mediaType,omitempty"`
	RawPayload      map[string]any   `bun:"rawPayload,type:jsonb" json:"rawPayload,omitempty"`
	CreatedAt       time.Time        `bun:"createdAt,type:timestamptz,notnull" json:"createdAt"`
	UpdatedAt       time.Time        `bun:"updatedAt,type:timestamptz,notnull" json:"updatedAt"`
}

// NewMessage cria uma nova instância de Message
func NewMessage(sessionID uuid.UUID, messageType MessageType, direction MessageDirection) *Message {
	now := time.Now()
	return &Message{
		ID:          uuid.New(),
		SessionID:   sessionID,
		MessageType: messageType,
		Direction:   direction,
		Status:      MessageStatusPending,
		Timestamp:   now,
		CreatedAt:   now,
		UpdatedAt:   now,
		RawPayload:  make(map[string]any),
	}
}

// UpdateStatus atualiza o status da mensagem
func (m *Message) UpdateStatus(status MessageStatus) {
	m.Status = status
	m.UpdatedAt = time.Now()
}

// SetContent define o conteúdo da mensagem
func (m *Message) SetContent(content string) {
	m.Content = content
	m.UpdatedAt = time.Now()
}

// SetCaption define a legenda da mensagem
func (m *Message) SetCaption(caption string) {
	m.Caption = caption
	m.UpdatedAt = time.Now()
}

// SetMediaID define o ID da mídia
func (m *Message) SetMediaID(mediaID uuid.UUID) {
	m.MediaID = &mediaID
	m.UpdatedAt = time.Now()
}

// SetReplyTo define a mensagem que está sendo respondida
func (m *Message) SetReplyTo(replyToID string) {
	m.QuotedMessageID = replyToID
	m.UpdatedAt = time.Now()
}

// SetRawPayloadField define um valor no payload bruto
func (m *Message) SetRawPayloadField(key string, value any) {
	if m.RawPayload == nil {
		m.RawPayload = make(map[string]any)
	}
	m.RawPayload[key] = value
	m.UpdatedAt = time.Now()
}

// GetRawPayloadField obtém um valor do payload bruto
func (m *Message) GetRawPayloadField(key string) (any, bool) {
	if m.RawPayload == nil {
		return nil, false
	}
	value, exists := m.RawPayload[key]
	return value, exists
}

// IsMediaMessage verifica se a mensagem contém mídia
func (m *Message) IsMediaMessage() bool {
	return m.MessageType == MessageTypeImage ||
		m.MessageType == MessageTypeVideo ||
		m.MessageType == MessageTypeAudio ||
		m.MessageType == MessageTypeDocument ||
		m.MessageType == MessageTypeSticker ||
		m.MessageType == MessageTypeGif
}

// SetMediaInfo define as informações de mídia da mensagem
func (m *Message) SetMediaInfo(path string, size int64, mimeType, fileName string) {
	m.MediaPath = path
	m.MediaSize = size
	m.MediaMimeType = mimeType
	m.MediaFileName = fileName
}

// HasMediaStored verifica se a mensagem tem mídia armazenada
func (m *Message) HasMediaStored() bool {
	return m.MediaPath != "" && m.MediaSize > 0
}

// GetMediaExtension retorna a extensão do arquivo de mídia
func (m *Message) GetMediaExtension() string {
	if m.MediaFileName == "" {
		return ""
	}

	// Extrair extensão do nome do arquivo
	parts := strings.Split(m.MediaFileName, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}

	return ""
}

// IsInteractiveMessage verifica se a mensagem é interativa
func (m *Message) IsInteractiveMessage() bool {
	return m.MessageType == MessageTypeButtons ||
		m.MessageType == MessageTypeList ||
		m.MessageType == MessageTypePoll ||
		m.MessageType == MessageTypePollUpdate ||
		m.MessageType == MessageTypeInteractive ||
		m.MessageType == MessageTypeReaction
}

// IsInbound verifica se a mensagem é de entrada
func (m *Message) IsInbound() bool {
	return m.Direction == MessageDirectionInbound
}

// IsOutbound verifica se a mensagem é de saída
func (m *Message) IsOutbound() bool {
	return m.Direction == MessageDirectionOutbound
}

// IsPending verifica se a mensagem está pendente
func (m *Message) IsPending() bool {
	return m.Status == MessageStatusPending
}

// IsDelivered verifica se a mensagem foi entregue
func (m *Message) IsDelivered() bool {
	return m.Status == MessageStatusDelivered || m.Status == MessageStatusRead
}

// IsRead verifica se a mensagem foi lida
func (m *Message) IsRead() bool {
	return m.Status == MessageStatusRead
}

// HasReply verifica se a mensagem é uma resposta
func (m *Message) HasReply() bool {
	return m.QuotedMessageID != ""
}
