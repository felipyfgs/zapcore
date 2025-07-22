package whatsapp

import (
	"io"
	"time"

	"github.com/google/uuid"
)

// SendTextRequest representa uma requisição de envio de texto
type SendTextRequest struct {
	SessionID uuid.UUID `json:"sessionId" validate:"required"`
	ToJID     string    `json:"to_jid" validate:"required"`
	Content   string    `json:"content" validate:"required"`
	ReplyToID string    `json:"reply_to_id,omitempty"`
}

// SendImageRequest representa uma requisição de envio de imagem
type SendImageRequest struct {
	SessionID  uuid.UUID `json:"sessionId" validate:"required"`
	ToJID      string    `json:"to_jid" validate:"required"`
	ImageData  io.Reader `json:"-"`
	ImageURL   string    `json:"image_url,omitempty"`
	Base64Data string    `json:"base64_data,omitempty"` // Dados em base64
	Caption    string    `json:"caption,omitempty"`
	ReplyToID  string    `json:"reply_to_id,omitempty"`
	MimeType   string    `json:"mime_type,omitempty"`
	FileName   string    `json:"file_name,omitempty"`
}

// SendAudioRequest representa uma requisição de envio de áudio
type SendAudioRequest struct {
	SessionID  uuid.UUID `json:"sessionId" validate:"required"`
	ToJID      string    `json:"to_jid" validate:"required"`
	AudioData  io.Reader `json:"-"`
	AudioURL   string    `json:"audio_url,omitempty"`
	Base64Data string    `json:"base64_data,omitempty"` // Dados em base64
	ReplyToID  string    `json:"reply_to_id,omitempty"`
	MimeType   string    `json:"mime_type,omitempty"`
	FileName   string    `json:"file_name,omitempty"`
	IsVoice    bool      `json:"is_voice,omitempty"`
}

// SendVideoRequest representa uma requisição de envio de vídeo
type SendVideoRequest struct {
	SessionID  uuid.UUID `json:"sessionId" validate:"required"`
	ToJID      string    `json:"to_jid" validate:"required"`
	VideoData  io.Reader `json:"-"`
	VideoURL   string    `json:"video_url,omitempty"`
	Base64Data string    `json:"base64_data,omitempty"` // Dados em base64
	Caption    string    `json:"caption,omitempty"`
	ReplyToID  string    `json:"reply_to_id,omitempty"`
	MimeType   string    `json:"mime_type,omitempty"`
	FileName   string    `json:"file_name,omitempty"`
}

// SendDocumentRequest representa uma requisição de envio de documento
type SendDocumentRequest struct {
	SessionID    uuid.UUID `json:"sessionId" validate:"required"`
	ToJID        string    `json:"to_jid" validate:"required"`
	DocumentData io.Reader `json:"-"`
	DocumentURL  string    `json:"document_url,omitempty"`
	Base64Data   string    `json:"base64_data,omitempty"` // Dados em base64
	FileName     string    `json:"file_name" validate:"required"`
	Caption      string    `json:"caption,omitempty"`
	ReplyToID    string    `json:"reply_to_id,omitempty"`
	MimeType     string    `json:"mime_type,omitempty"`
}

// SendStickerRequest representa uma requisição de envio de sticker
type SendStickerRequest struct {
	SessionID   uuid.UUID `json:"sessionId" validate:"required"`
	ToJID       string    `json:"to_jid" validate:"required"`
	StickerData io.Reader `json:"-"`
	StickerURL  string    `json:"sticker_url,omitempty"`
	Base64Data  string    `json:"base64_data,omitempty"` // Dados em base64
	ReplyToID   string    `json:"reply_to_id,omitempty"`
	MimeType    string    `json:"mime_type,omitempty"`
}

// SendLocationRequest representa uma requisição de envio de localização
type SendLocationRequest struct {
	SessionID uuid.UUID `json:"sessionId" validate:"required"`
	ToJID     string    `json:"to_jid" validate:"required"`
	Latitude  float64   `json:"latitude" validate:"required"`
	Longitude float64   `json:"longitude" validate:"required"`
	Name      string    `json:"name,omitempty"`
	Address   string    `json:"address,omitempty"`
	ReplyToID string    `json:"reply_to_id,omitempty"`
}

// SendContactRequest representa uma requisição de envio de contato
type SendContactRequest struct {
	SessionID uuid.UUID      `json:"sessionId" validate:"required"`
	ToJID     string         `json:"to_jid" validate:"required"`
	Contacts  []ContactVCard `json:"contacts" validate:"required,min=1"`
	ReplyToID string         `json:"reply_to_id,omitempty"`
}

// ContactVCard representa um contato no formato vCard
type ContactVCard struct {
	Name         string `json:"name" validate:"required"`
	PhoneNumber  string `json:"phone_number" validate:"required"`
	Organization string `json:"organization,omitempty"`
	Email        string `json:"email,omitempty"`
}

// SendButtonsRequest representa uma requisição de envio de botões
type SendButtonsRequest struct {
	SessionID uuid.UUID `json:"sessionId" validate:"required"`
	ToJID     string    `json:"to_jid" validate:"required"`
	Text      string    `json:"text" validate:"required"`
	Footer    string    `json:"footer,omitempty"`
	Buttons   []Button  `json:"buttons" validate:"required,min=1,max=3"`
	ReplyToID string    `json:"reply_to_id,omitempty"`
}

// Button representa um botão interativo
type Button struct {
	ID    string `json:"id" validate:"required"`
	Title string `json:"title" validate:"required"`
}

// SendListRequest representa uma requisição de envio de lista
type SendListRequest struct {
	SessionID  uuid.UUID     `json:"sessionId" validate:"required"`
	ToJID      string        `json:"to_jid" validate:"required"`
	Text       string        `json:"text" validate:"required"`
	Footer     string        `json:"footer,omitempty"`
	ButtonText string        `json:"button_text" validate:"required"`
	Sections   []ListSection `json:"sections" validate:"required,min=1"`
	ReplyToID  string        `json:"reply_to_id,omitempty"`
}

// ListSection representa uma seção da lista
type ListSection struct {
	Title string    `json:"title" validate:"required"`
	Rows  []ListRow `json:"rows" validate:"required,min=1"`
}

// ListRow representa uma linha da lista
type ListRow struct {
	ID          string `json:"id" validate:"required"`
	Title       string `json:"title" validate:"required"`
	Description string `json:"description,omitempty"`
}

// SendPollRequest representa uma requisição de envio de enquete
type SendPollRequest struct {
	SessionID   uuid.UUID    `json:"sessionId" validate:"required"`
	ToJID       string       `json:"to_jid" validate:"required"`
	Question    string       `json:"question" validate:"required"`
	Options     []PollOption `json:"options" validate:"required,min=2,max=12"`
	SelectCount int          `json:"select_count,omitempty"` // 0 = single choice, >1 = multiple choice
	ReplyToID   string       `json:"reply_to_id,omitempty"`
}

// PollOption representa uma opção da enquete
type PollOption struct {
	Name string `json:"name" validate:"required"`
}

// EditMessageRequest representa uma requisição de edição de mensagem
type EditMessageRequest struct {
	SessionID uuid.UUID `json:"sessionId" validate:"required"`
	MessageID string    `json:"messageId" validate:"required"`
	NewText   string    `json:"new_text" validate:"required"`
}

// SendReactionRequest representa uma requisição para enviar reação
type SendReactionRequest struct {
	SessionID uuid.UUID `json:"sessionId" validate:"required"`
	ChatJID   string    `json:"chat_jid" validate:"required"`
	MessageID string    `json:"messageId" validate:"required"`
	SenderJID string    `json:"sender_jid,omitempty"`
	Reaction  string    `json:"reaction"` // emoji ou string vazia para remover
}

// RevokeMessageRequest representa uma requisição para revogar mensagem
type RevokeMessageRequest struct {
	SessionID uuid.UUID `json:"sessionId" validate:"required"`
	ChatJID   string    `json:"chat_jid" validate:"required"`
	MessageID string    `json:"messageId" validate:"required"`
}

// DownloadMediaRequest representa uma requisição para download de mídia
type DownloadMediaRequest struct {
	SessionID     uuid.UUID `json:"sessionId" validate:"required"`
	DirectPath    string    `json:"direct_path" validate:"required"`
	MediaKey      []byte    `json:"media_key" validate:"required"`
	FileEncSHA256 []byte    `json:"file_enc_sha256" validate:"required"`
	FileSHA256    []byte    `json:"file_sha256" validate:"required"`
	FileLength    uint64    `json:"file_length" validate:"required"`
	MediaType     string    `json:"media_type" validate:"required"`
}

// UploadMediaRequest representa uma requisição para upload de mídia
type UploadMediaRequest struct {
	SessionID uuid.UUID `json:"sessionId" validate:"required"`
	MediaData io.Reader `json:"-"`
	MediaType string    `json:"media_type" validate:"required"` // image, video, audio, document
}

// UploadResponse representa a resposta do upload de mídia
type UploadResponse struct {
	URL           string `json:"url"`
	DirectPath    string `json:"direct_path"`
	Handle        string `json:"handle"`
	ObjectID      string `json:"object_id"`
	MediaKey      []byte `json:"media_key"`
	FileEncSHA256 []byte `json:"file_enc_sha256"`
	FileSHA256    []byte `json:"file_sha256"`
	FileLength    uint64 `json:"file_length"`
}

// MarkAsReadRequest representa uma requisição para marcar como lida
type MarkAsReadRequest struct {
	SessionID  uuid.UUID `json:"sessionId" validate:"required"`
	ChatJID    string    `json:"chat_jid" validate:"required"`
	MessageIDs []string  `json:"messageIds" validate:"required"`
	Timestamp  time.Time `json:"timestamp"`
	SenderJID  string    `json:"sender_jid,omitempty"`
}

// SendPresenceRequest representa uma requisição para enviar presença
type SendPresenceRequest struct {
	SessionID uuid.UUID    `json:"sessionId" validate:"required"`
	State     PresenceType `json:"state" validate:"required"`
}

// SendChatPresenceRequest representa uma requisição para enviar presença no chat
type SendChatPresenceRequest struct {
	SessionID uuid.UUID         `json:"sessionId"`
	ChatJID   string            `json:"chat_jid"`
	State     ChatPresenceType  `json:"state"`
	Media     ChatPresenceMedia `json:"media,omitempty"`
}

// ProfilePictureInfo representa informações da foto de perfil
type ProfilePictureInfo struct {
	URL        string    `json:"url"`
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	DirectPath string    `json:"direct_path"`
	Timestamp  time.Time `json:"timestamp"`
}

// UserInfo representa informações do usuário
type UserInfo struct {
	JID          string `json:"jid"`
	BusinessName string `json:"business_name,omitempty"`
	PushName     string `json:"push_name,omitempty"`
	VerifiedName string `json:"verified_name,omitempty"`
	Status       string `json:"status,omitempty"`
	PictureID    string `json:"picture_id,omitempty"`
	Devices      []int  `json:"devices,omitempty"`
}

// IsOnWhatsAppResponse representa a resposta de verificação se está no WhatsApp
type IsOnWhatsAppResponse struct {
	Query      string `json:"query"`
	JID        string `json:"jid"`
	IsIn       bool   `json:"is_in"`
	VerifyName string `json:"verify_name,omitempty"`
}

// GroupInfo representa informações do grupo
type GroupInfo struct {
	JID                           string             `json:"jid"`
	Name                          string             `json:"name"`
	Topic                         string             `json:"topic,omitempty"`
	TopicID                       string             `json:"topic_id,omitempty"`
	TopicSetAt                    time.Time          `json:"topic_set_at,omitempty"`
	TopicSetBy                    string             `json:"topic_set_by,omitempty"`
	GroupCreated                  time.Time          `json:"group_created"`
	ParticipantVersionID          string             `json:"participant_version_id"`
	Participants                  []GroupParticipant `json:"participants"`
	IsAnnounce                    bool               `json:"is_announce"`
	IsLocked                      bool               `json:"is_locked"`
	IsIncognito                   bool               `json:"is_incognito"`
	IsParent                      bool               `json:"is_parent"`
	LinkedParentJID               string             `json:"linked_parent_jid,omitempty"`
	DefaultMembershipApprovalMode string             `json:"default_membership_approval_mode"`
}

// GroupParticipant representa um participante do grupo
type GroupParticipant struct {
	JID          string                      `json:"jid"`
	IsAdmin      bool                        `json:"is_admin"`
	IsSuperAdmin bool                        `json:"is_super_admin"`
	DisplayName  string                      `json:"display_name,omitempty"`
	AddRequest   *GroupParticipantAddRequest `json:"add_request,omitempty"`
}

// GroupParticipantAddRequest representa uma solicitação para adicionar participante
type GroupParticipantAddRequest struct {
	Code       string    `json:"code"`
	Expiration time.Time `json:"expiration"`
}

// CreateGroupRequest representa uma requisição para criar grupo
type CreateGroupRequest struct {
	SessionID       uuid.UUID `json:"sessionId"`
	Name            string    `json:"name"`
	Participants    []string  `json:"participants"`
	CreateKey       string    `json:"create_key,omitempty"`
	IsParent        bool      `json:"is_parent,omitempty"`
	LinkedParentJID string    `json:"linked_parent_jid,omitempty"`
}

// UpdateGroupParticipantsRequest representa uma requisição para atualizar participantes
type UpdateGroupParticipantsRequest struct {
	SessionID    uuid.UUID                    `json:"sessionId"`
	GroupJID     string                       `json:"group_jid"`
	Participants []string                     `json:"participants"`
	Action       GroupParticipantChangeAction `json:"action"`
}

// MessageResponse representa a resposta de envio de mensagem
type MessageResponse struct {
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
	Error     string `json:"error,omitempty"`
}

// Contact representa um contato
type Contact struct {
	JID          string `json:"jid"`
	Name         string `json:"name"`
	NotifyName   string `json:"notify_name"`
	PhoneNumber  string `json:"phone_number"`
	IsBusiness   bool   `json:"isBusiness"`
	IsMyContact  bool   `json:"is_my_contact"`
	IsWAContact  bool   `json:"is_wa_contact"`
	ProfilePicID string `json:"profile_pic_id"`
}

// Chat representa um chat
type Chat struct {
	JID             string    `json:"jid"`
	Name            string    `json:"name"`
	IsGroup         bool      `json:"isGroup"`
	LastMessageTime time.Time `json:"last_message_time"`
	UnreadCount     int       `json:"unread_count"`
	IsMuted         bool      `json:"isMuted"`
	IsPinned        bool      `json:"is_pinned"`
	IsArchived      bool      `json:"is_archived"`
}

// ConnectionStatus representa o status da conexão
type ConnectionStatus string

const (
	StatusDisconnected ConnectionStatus = "disconnected"
	StatusConnecting   ConnectionStatus = "connecting"
	StatusConnected    ConnectionStatus = "connected"
	StatusLoggedOut    ConnectionStatus = "logged_out"
)

// PresenceType representa o tipo de presença
type PresenceType string

const (
	PresenceAvailable   PresenceType = "available"
	PresenceUnavailable PresenceType = "unavailable"
)

// ChatPresenceType representa o tipo de presença no chat
type ChatPresenceType string

const (
	ChatPresenceComposing ChatPresenceType = "composing"
	ChatPresencePaused    ChatPresenceType = "paused"
	ChatPresenceRecording ChatPresenceType = "recording"
)

// ChatPresenceMedia representa o tipo de mídia na presença do chat
type ChatPresenceMedia string

const (
	ChatPresenceMediaAudio ChatPresenceMedia = "audio"
)

// GroupParticipantChangeAction representa a ação de mudança de participante
type GroupParticipantChangeAction string

const (
	GroupParticipantChangeAdd     GroupParticipantChangeAction = "add"
	GroupParticipantChangeRemove  GroupParticipantChangeAction = "remove"
	GroupParticipantChangePromote GroupParticipantChangeAction = "promote"
	GroupParticipantChangeDemote  GroupParticipantChangeAction = "demote"
)
