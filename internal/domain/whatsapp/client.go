package whatsapp

import (
	"context"

	"github.com/google/uuid"
)

// Client define a interface para o cliente WhatsApp baseada na biblioteca whatsmeow
type Client interface {
	// Connect estabelece conexão com o WhatsApp
	Connect(ctx context.Context, sessionID uuid.UUID) error

	// Disconnect encerra a conexão
	Disconnect(ctx context.Context, sessionID uuid.UUID) error

	// GetQRCode gera QR Code para autenticação
	GetQRCode(ctx context.Context, sessionID uuid.UUID) (string, error)

	// PairPhone emparelha com um número de telefone usando código
	PairPhone(ctx context.Context, sessionID uuid.UUID, phoneNumber string, showPushNotification bool) error

	// GetStatus retorna o status da conexão
	GetStatus(ctx context.Context, sessionID uuid.UUID) (ConnectionStatus, error)

	// IsConnected verifica se está conectado
	IsConnected(ctx context.Context, sessionID uuid.UUID) bool

	// IsLoggedIn verifica se está logado
	IsLoggedIn(ctx context.Context, sessionID uuid.UUID) bool

	// SendTextMessage envia mensagem de texto
	SendTextMessage(ctx context.Context, req *SendTextRequest) (*MessageResponse, error)

	// SendImageMessage envia imagem
	SendImageMessage(ctx context.Context, req *SendImageRequest) (*MessageResponse, error)

	// SendAudioMessage envia áudio
	SendAudioMessage(ctx context.Context, req *SendAudioRequest) (*MessageResponse, error)

	// SendVideoMessage envia vídeo
	SendVideoMessage(ctx context.Context, req *SendVideoRequest) (*MessageResponse, error)

	// SendDocumentMessage envia documento
	SendDocumentMessage(ctx context.Context, req *SendDocumentRequest) (*MessageResponse, error)

	// SendStickerMessage envia sticker
	SendStickerMessage(ctx context.Context, req *SendStickerRequest) (*MessageResponse, error)

	// SendLocationMessage envia localização
	SendLocationMessage(ctx context.Context, req *SendLocationRequest) (*MessageResponse, error)

	// SendContactMessage envia contato
	SendContactMessage(ctx context.Context, req *SendContactRequest) (*MessageResponse, error)

	// SendReactionMessage envia reação
	SendReactionMessage(ctx context.Context, req *SendReactionRequest) (*MessageResponse, error)

	// SendPollMessage envia enquete
	SendPollMessage(ctx context.Context, req *SendPollRequest) (*MessageResponse, error)

	// EditMessage edita uma mensagem
	EditMessage(ctx context.Context, req *EditMessageRequest) (*MessageResponse, error)

	// RevokeMessage revoga uma mensagem
	RevokeMessage(ctx context.Context, req *RevokeMessageRequest) (*MessageResponse, error)

	// DownloadMedia faz download de mídia
	DownloadMedia(ctx context.Context, req *DownloadMediaRequest) ([]byte, error)

	// UploadMedia faz upload de mídia
	UploadMedia(ctx context.Context, req *UploadMediaRequest) (*UploadResponse, error)

	// SetProxy configura proxy
	SetProxy(ctx context.Context, sessionID uuid.UUID, proxyURL string) error

	// GetContacts obtém lista de contatos
	GetContacts(ctx context.Context, sessionID uuid.UUID) ([]*Contact, error)

	// GetChats obtém lista de chats
	GetChats(ctx context.Context, sessionID uuid.UUID) ([]*Chat, error)

	// GetGroupInfo obtém informações do grupo
	GetGroupInfo(ctx context.Context, sessionID uuid.UUID, groupJID string) (*GroupInfo, error)

	// MarkAsRead marca mensagem como lida
	MarkAsRead(ctx context.Context, req *MarkAsReadRequest) error

	// SendPresence define presença (online/offline/typing)
	SendPresence(ctx context.Context, req *SendPresenceRequest) error

	// SendChatPresence define presença no chat (typing/recording)
	SendChatPresence(ctx context.Context, req *SendChatPresenceRequest) error

	// GetProfilePicture obtém foto de perfil
	GetProfilePicture(ctx context.Context, sessionID uuid.UUID, jid string) (*ProfilePictureInfo, error)

	// SubscribePresence se inscreve para receber atualizações de presença
	SubscribePresence(ctx context.Context, sessionID uuid.UUID, jid string) error

	// GetUserInfo obtém informações do usuário
	GetUserInfo(ctx context.Context, sessionID uuid.UUID, jids []string) ([]*UserInfo, error)

	// IsOnWhatsApp verifica se números estão no WhatsApp
	IsOnWhatsApp(ctx context.Context, sessionID uuid.UUID, phones []string) ([]*IsOnWhatsAppResponse, error)

	// CreateGroup cria um grupo
	CreateGroup(ctx context.Context, req *CreateGroupRequest) (*GroupInfo, error)

	// LeaveGroup sai do grupo
	LeaveGroup(ctx context.Context, sessionID uuid.UUID, groupJID string) error

	// UpdateGroupParticipants atualiza participantes do grupo
	UpdateGroupParticipants(ctx context.Context, req *UpdateGroupParticipantsRequest) error

	// SetGroupName define nome do grupo
	SetGroupName(ctx context.Context, sessionID uuid.UUID, groupJID, name string) error

	// SetGroupDescription define descrição do grupo
	SetGroupDescription(ctx context.Context, sessionID uuid.UUID, groupJID, description string) error

	// GetGroupInviteLink obtém link de convite do grupo
	GetGroupInviteLink(ctx context.Context, sessionID uuid.UUID, groupJID string, reset bool) (string, error)

	// JoinGroupWithLink entra no grupo via link
	JoinGroupWithLink(ctx context.Context, sessionID uuid.UUID, inviteCode string) (*GroupInfo, error)
}
