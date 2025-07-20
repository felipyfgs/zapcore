package message

import (
	"time"

	"github.com/google/uuid"
)

// MessageType representa os tipos de mensagem suportados
type MessageType string

const (
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
	MessageTypePoll         MessageType = "pollMessage"
	MessageTypeReaction     MessageType = "reactionMessage"
	MessageTypeButtons      MessageType = "buttonsMessage"
	MessageTypeList         MessageType = "listMessage"
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
	ID          uuid.UUID        `json:"id"`
	SessionID   uuid.UUID        `json:"session_id"`
	MessageID   string           `json:"message_id"`   // ID único do WhatsApp
	Type        MessageType      `json:"type"`
	Direction   MessageDirection `json:"direction"`
	Status      MessageStatus    `json:"status"`
	FromJID     string           `json:"from_jid"`
	ToJID       string           `json:"to_jid"`
	Content     string           `json:"content,omitempty"`
	MediaID     *uuid.UUID       `json:"media_id,omitempty"`
	Caption     string           `json:"caption,omitempty"`
	Timestamp   time.Time        `json:"timestamp"`
	ReplyToID   string           `json:"reply_to_id,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// NewMessage cria uma nova instância de Message
func NewMessage(sessionID uuid.UUID, messageType MessageType, direction MessageDirection) *Message {
	now := time.Now()
	return &Message{
		ID:        uuid.New(),
		SessionID: sessionID,
		Type:      messageType,
		Direction: direction,
		Status:    MessageStatusPending,
		Timestamp: now,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]any),
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
	m.ReplyToID = replyToID
	m.UpdatedAt = time.Now()
}

// SetMetadata define um valor nos metadados
func (m *Message) SetMetadata(key string, value any) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]any)
	}
	m.Metadata[key] = value
	m.UpdatedAt = time.Now()
}

// GetMetadata obtém um valor dos metadados
func (m *Message) GetMetadata(key string) (any, bool) {
	if m.Metadata == nil {
		return nil, false
	}
	value, exists := m.Metadata[key]
	return value, exists
}

// IsMediaMessage verifica se a mensagem contém mídia
func (m *Message) IsMediaMessage() bool {
	return m.Type == MessageTypeImage ||
		m.Type == MessageTypeVideo ||
		m.Type == MessageTypeAudio ||
		m.Type == MessageTypeDocument ||
		m.Type == MessageTypeSticker ||
		m.Type == MessageTypeGif
}

// IsInteractiveMessage verifica se a mensagem é interativa
func (m *Message) IsInteractiveMessage() bool {
	return m.Type == MessageTypeButtons ||
		m.Type == MessageTypeList ||
		m.Type == MessageTypePoll
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
	return m.ReplyToID != ""
}

