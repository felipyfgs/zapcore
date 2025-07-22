package whatsapp

import (
	"context"
	"fmt"
	"sync"

	"zapcore/internal/domain/session"
	"zapcore/internal/domain/whatsapp"
	"zapcore/internal/infra/storage"
	"zapcore/pkg/logger"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
)

// Constantes para validação de mídia (baseado na documentação oficial do WhatsApp Business API 2024)
const (
	// Limites de tamanho em bytes - atualizados conforme documentação oficial
	MaxImageSize    = 5 * 1024 * 1024   // 5MB (limite oficial do WhatsApp)
	MaxVideoSize    = 16 * 1024 * 1024  // 16MB (limite oficial do WhatsApp)
	MaxAudioSize    = 16 * 1024 * 1024  // 16MB (limite oficial do WhatsApp)
	MaxDocumentSize = 100 * 1024 * 1024 // 100MB (limite oficial do WhatsApp Cloud API)
	MaxStickerSize  = 500 * 1024        // 500KB (limite oficial do WhatsApp)
)

// Tipos MIME suportados
var (
	SupportedImageMimes = []string{
		"image/jpeg",
		"image/png",
		"image/webp",
	}

	SupportedVideoMimes = []string{
		"video/mp4",
		"video/3gpp",
		"video/quicktime",
		"video/avi",
		"video/mkv",
	}

	SupportedAudioMimes = []string{
		"audio/mpeg",
		"audio/mp3",
		"audio/aac",
		"audio/ogg",
		"audio/wav",
		"audio/m4a",
	}

	SupportedDocumentMimes = []string{
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"text/plain",
		"application/zip",
		"application/x-rar-compressed",
	}

	SupportedStickerMimes = []string{
		"image/webp",
	}
)

// WhatsAppClient implementa a interface whatsapp.Client
type WhatsAppClient struct {
	container    *sqlstore.Container
	clients      map[uuid.UUID]*whatsmeow.Client
	clientsMutex sync.RWMutex
	killChannels map[uuid.UUID]chan bool
	killMutex    sync.RWMutex
	sessionRepo  interface {
		GetActiveSessions(ctx context.Context) ([]*session.Session, error)
		UpdateJID(ctx context.Context, sessionID uuid.UUID, jid string) error
		UpdateStatus(ctx context.Context, sessionID uuid.UUID, status session.WhatsAppSessionStatus) error
	}
	logger            *logger.Logger
	eventHandler      EventHandler
	minioClient       *storage.MinIOClient
	connectionManager *ConnectionManager
	messageSender     *MessageSender
}

// PairSuccessEvent representa o evento de pareamento bem-sucedido
type PairSuccessEvent struct {
	SessionID    uuid.UUID
	JID          string
	BusinessName string
	Platform     string
}

// EventHandler define a interface para manipular eventos do WhatsApp
type EventHandler interface {
	HandleEvent(sessionID uuid.UUID, event interface{})
}

// NewWhatsAppClient cria uma nova instância do cliente WhatsApp
func NewWhatsAppClient(dbContainer *sqlstore.Container, sessionRepo interface {
	GetActiveSessions(ctx context.Context) ([]*session.Session, error)
	UpdateJID(ctx context.Context, sessionID uuid.UUID, jid string) error
	UpdateStatus(ctx context.Context, sessionID uuid.UUID, status session.WhatsAppSessionStatus) error
}, eventHandler EventHandler, minioClient *storage.MinIOClient) *WhatsAppClient {
	client := &WhatsAppClient{
		container:    dbContainer,
		clients:      make(map[uuid.UUID]*whatsmeow.Client),
		killChannels: make(map[uuid.UUID]chan bool),
		sessionRepo:  sessionRepo,
		logger:       logger.Get(),
		eventHandler: eventHandler,
		minioClient:  minioClient,
	}

	// Inicializar componentes
	client.connectionManager = NewConnectionManager(client)
	client.messageSender = NewMessageSender(client)

	return client
}

// ConnectOnStartup reconecta automaticamente sessões ativas com JID
func (c *WhatsAppClient) ConnectOnStartup(ctx context.Context) error {
	return c.connectionManager.ConnectOnStartup(ctx)
}

// Connect estabelece conexão com o WhatsApp
func (c *WhatsAppClient) Connect(ctx context.Context, sessionID uuid.UUID) error {
	return c.connectionManager.Connect(ctx, sessionID)
}

// Disconnect encerra a conexão
func (c *WhatsAppClient) Disconnect(ctx context.Context, sessionID uuid.UUID) error {
	return c.connectionManager.Disconnect(ctx, sessionID)
}

// GetQRCode obtém QR Code (método simplificado)
func (c *WhatsAppClient) GetQRCode(ctx context.Context, sessionID uuid.UUID) (string, error) {
	c.clientsMutex.RLock()
	client, exists := c.clients[sessionID]
	c.clientsMutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("cliente não encontrado para sessão %s. Execute /connect primeiro", sessionID.String())
	}

	if client.Store.ID != nil {
		return "", fmt.Errorf("cliente já está autenticado")
	}

	// Retornar mensagem informativa
	return "", fmt.Errorf("QR Code sendo processado. Verifique o terminal do servidor")
}

// PairPhone emparelha com um número de telefone
func (c *WhatsAppClient) PairPhone(ctx context.Context, sessionID uuid.UUID, phoneNumber string, showPushNotification bool) error {
	c.clientsMutex.RLock()
	client, exists := c.clients[sessionID]
	c.clientsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("cliente não encontrado para sessão %s", sessionID.String())
	}

	// Implementar pareamento por telefone usando whatsmeow
	code, err := client.PairPhone(ctx, phoneNumber, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		return fmt.Errorf("erro ao emparelhar com telefone: %w", err)
	}

	c.logger.Info().Str("phone", phoneNumber).Str("code", code).Msg("Código de pareamento gerado")
	return nil
}

// GetStatus retorna o status da conexão
func (c *WhatsAppClient) GetStatus(ctx context.Context, sessionID uuid.UUID) (whatsapp.ConnectionStatus, error) {
	return c.connectionManager.GetStatus(ctx, sessionID)
}

// IsConnected verifica se está conectado
func (c *WhatsAppClient) IsConnected(ctx context.Context, sessionID uuid.UUID) bool {
	return c.connectionManager.IsConnected(sessionID)
}

// IsLoggedIn verifica se está logado
func (c *WhatsAppClient) IsLoggedIn(ctx context.Context, sessionID uuid.UUID) bool {
	return c.connectionManager.IsLoggedIn(sessionID)
}

// SendTextMessage envia mensagem de texto
func (c *WhatsAppClient) SendTextMessage(ctx context.Context, req *whatsapp.SendTextRequest) (*whatsapp.MessageResponse, error) {
	return c.messageSender.SendTextMessage(ctx, req)
}

// SendImageMessage envia imagem
func (c *WhatsAppClient) SendImageMessage(ctx context.Context, req *whatsapp.SendImageRequest) (*whatsapp.MessageResponse, error) {
	return c.messageSender.SendImageMessage(ctx, req)
}

// SendAudioMessage envia áudio
func (c *WhatsAppClient) SendAudioMessage(ctx context.Context, req *whatsapp.SendAudioRequest) (*whatsapp.MessageResponse, error) {
	return c.messageSender.SendAudioMessage(ctx, req)
}

// SendVideoMessage envia vídeo
func (c *WhatsAppClient) SendVideoMessage(ctx context.Context, req *whatsapp.SendVideoRequest) (*whatsapp.MessageResponse, error) {
	return c.messageSender.SendVideoMessage(ctx, req)
}

// SendDocumentMessage envia documento
func (c *WhatsAppClient) SendDocumentMessage(ctx context.Context, req *whatsapp.SendDocumentRequest) (*whatsapp.MessageResponse, error) {
	return c.messageSender.SendDocumentMessage(ctx, req)
}

// SendStickerMessage envia sticker
func (c *WhatsAppClient) SendStickerMessage(ctx context.Context, req *whatsapp.SendStickerRequest) (*whatsapp.MessageResponse, error) {
	return c.messageSender.SendStickerMessage(ctx, req)
}

// SendLocationMessage envia localização
func (c *WhatsAppClient) SendLocationMessage(ctx context.Context, req *whatsapp.SendLocationRequest) (*whatsapp.MessageResponse, error) {
	return nil, fmt.Errorf("SendLocationMessage não implementado ainda")
}

// SendContactMessage envia contato
func (c *WhatsAppClient) SendContactMessage(ctx context.Context, req *whatsapp.SendContactRequest) (*whatsapp.MessageResponse, error) {
	return nil, fmt.Errorf("SendContactMessage não implementado ainda")
}

// SendReactionMessage envia reação
func (c *WhatsAppClient) SendReactionMessage(ctx context.Context, req *whatsapp.SendReactionRequest) (*whatsapp.MessageResponse, error) {
	return nil, fmt.Errorf("SendReactionMessage não implementado ainda")
}

// SendPollMessage envia enquete
func (c *WhatsAppClient) SendPollMessage(ctx context.Context, req *whatsapp.SendPollRequest) (*whatsapp.MessageResponse, error) {
	return nil, fmt.Errorf("SendPollMessage não implementado ainda")
}

// EditMessage edita uma mensagem
func (c *WhatsAppClient) EditMessage(ctx context.Context, req *whatsapp.EditMessageRequest) (*whatsapp.MessageResponse, error) {
	return nil, fmt.Errorf("EditMessage não implementado ainda")
}

// RevokeMessage revoga uma mensagem
func (c *WhatsAppClient) RevokeMessage(ctx context.Context, req *whatsapp.RevokeMessageRequest) (*whatsapp.MessageResponse, error) {
	return nil, fmt.Errorf("RevokeMessage não implementado ainda")
}

// DownloadMedia faz download de mídia
func (c *WhatsAppClient) DownloadMedia(ctx context.Context, req *whatsapp.DownloadMediaRequest) ([]byte, error) {
	return nil, fmt.Errorf("DownloadMedia não implementado ainda")
}

// UploadMedia faz upload de mídia
func (c *WhatsAppClient) UploadMedia(ctx context.Context, req *whatsapp.UploadMediaRequest) (*whatsapp.UploadResponse, error) {
	return nil, fmt.Errorf("UploadMedia não implementado ainda")
}

// SetProxy configura proxy
func (c *WhatsAppClient) SetProxy(ctx context.Context, sessionID uuid.UUID, proxyURL string) error {
	return fmt.Errorf("SetProxy não implementado ainda")
}

// GetContacts obtém lista de contatos
func (c *WhatsAppClient) GetContacts(ctx context.Context, sessionID uuid.UUID) ([]*whatsapp.Contact, error) {
	return nil, fmt.Errorf("GetContacts não implementado ainda")
}

// GetChats obtém lista de chats
func (c *WhatsAppClient) GetChats(ctx context.Context, sessionID uuid.UUID) ([]*whatsapp.Chat, error) {
	return nil, fmt.Errorf("GetChats não implementado ainda")
}

// GetGroupInfo obtém informações do grupo
func (c *WhatsAppClient) GetGroupInfo(ctx context.Context, sessionID uuid.UUID, groupJID string) (*whatsapp.GroupInfo, error) {
	return nil, fmt.Errorf("GetGroupInfo não implementado ainda")
}

// MarkAsRead marca mensagem como lida
func (c *WhatsAppClient) MarkAsRead(ctx context.Context, req *whatsapp.MarkAsReadRequest) error {
	return fmt.Errorf("MarkAsRead não implementado ainda")
}

// SendPresence define presença (online/offline/typing)
func (c *WhatsAppClient) SendPresence(ctx context.Context, req *whatsapp.SendPresenceRequest) error {
	return fmt.Errorf("SendPresence não implementado ainda")
}

// SendChatPresence define presença no chat (typing/recording)
func (c *WhatsAppClient) SendChatPresence(ctx context.Context, req *whatsapp.SendChatPresenceRequest) error {
	return fmt.Errorf("SendChatPresence não implementado ainda")
}

// GetProfilePicture obtém foto de perfil
func (c *WhatsAppClient) GetProfilePicture(ctx context.Context, sessionID uuid.UUID, jid string) (*whatsapp.ProfilePictureInfo, error) {
	return nil, fmt.Errorf("GetProfilePicture não implementado ainda")
}

// SubscribePresence se inscreve para receber atualizações de presença
func (c *WhatsAppClient) SubscribePresence(ctx context.Context, sessionID uuid.UUID, jid string) error {
	return fmt.Errorf("SubscribePresence não implementado ainda")
}

// GetUserInfo obtém informações do usuário
func (c *WhatsAppClient) GetUserInfo(ctx context.Context, sessionID uuid.UUID, jids []string) ([]*whatsapp.UserInfo, error) {
	return nil, fmt.Errorf("GetUserInfo não implementado ainda")
}

// IsOnWhatsApp verifica se números estão no WhatsApp
func (c *WhatsAppClient) IsOnWhatsApp(ctx context.Context, sessionID uuid.UUID, phones []string) ([]*whatsapp.IsOnWhatsAppResponse, error) {
	return nil, fmt.Errorf("IsOnWhatsApp não implementado ainda")
}

// CreateGroup cria um grupo
func (c *WhatsAppClient) CreateGroup(ctx context.Context, req *whatsapp.CreateGroupRequest) (*whatsapp.GroupInfo, error) {
	return nil, fmt.Errorf("CreateGroup não implementado ainda")
}

// LeaveGroup sai do grupo
func (c *WhatsAppClient) LeaveGroup(ctx context.Context, sessionID uuid.UUID, groupJID string) error {
	return fmt.Errorf("LeaveGroup não implementado ainda")
}

// UpdateGroupParticipants atualiza participantes do grupo
func (c *WhatsAppClient) UpdateGroupParticipants(ctx context.Context, req *whatsapp.UpdateGroupParticipantsRequest) error {
	return fmt.Errorf("UpdateGroupParticipants não implementado ainda")
}

// SetGroupName define nome do grupo
func (c *WhatsAppClient) SetGroupName(ctx context.Context, sessionID uuid.UUID, groupJID, name string) error {
	return fmt.Errorf("SetGroupName não implementado ainda")
}

// SetGroupDescription define descrição do grupo
func (c *WhatsAppClient) SetGroupDescription(ctx context.Context, sessionID uuid.UUID, groupJID, description string) error {
	return fmt.Errorf("SetGroupDescription não implementado ainda")
}

// GetGroupInviteLink obtém link de convite do grupo
func (c *WhatsAppClient) GetGroupInviteLink(ctx context.Context, sessionID uuid.UUID, groupJID string, reset bool) (string, error) {
	return "", fmt.Errorf("GetGroupInviteLink não implementado ainda")
}

// JoinGroupWithLink entra no grupo via link
func (c *WhatsAppClient) JoinGroupWithLink(ctx context.Context, sessionID uuid.UUID, inviteCode string) (*whatsapp.GroupInfo, error) {
	return nil, fmt.Errorf("JoinGroupWithLink não implementado ainda")
}

// handleWhatsAppEvent manipula eventos do WhatsApp
func (c *WhatsAppClient) handleWhatsAppEvent(sessionID uuid.UUID, evt interface{}) {
	switch e := evt.(type) {
	case *events.Message:
		c.logger.Info().
			Str("session_id", sessionID.String()).
			Str("message_id", e.Info.ID).
			Str("from", e.Info.SourceString()).
			Str("pushname", e.Info.PushName).
			Msg("Mensagem recebida")

	case *events.UndecryptableMessage:
		c.logger.Warn().
			Str("session_id", sessionID.String()).
			Str("message_id", e.Info.ID).
			Str("from", e.Info.SourceString()).
			Str("pushname", e.Info.PushName).
			Bool("is_unavailable", e.IsUnavailable).
			Str("unavailable_type", string(e.UnavailableType)).
			Msg("Mensagem não descriptografável recebida")

	case *events.Receipt:
		c.logger.Info().
			Str("session_id", sessionID.String()).
			Strs("message_ids", e.MessageIDs).
			Str("type", string(e.Type)).
			Str("source", e.SourceString()).
			Msg("Recibo recebido")

	case *events.PairSuccess:
		c.logger.Info().
			Str("session_id", sessionID.String()).
			Str("jid", e.ID.String()).
			Str("business_name", e.BusinessName).
			Str("platform", e.Platform).
			Msg("Pareamento bem-sucedido")

		// Atualizar JID no banco de dados
		ctx := context.Background()
		if err := c.sessionRepo.UpdateJID(ctx, sessionID, e.ID.String()); err != nil {
			c.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao atualizar JID após pareamento")
		} else {
			c.logger.Info().Str("session_id", sessionID.String()).Str("jid", e.ID.String()).Msg("JID atualizado com sucesso após pareamento")
		}

	case *events.Connected:
		c.logger.Info().
			Str("session_id", sessionID.String()).
			Msg("WhatsApp conectado")

		// Atualizar status para connected no banco de dados
		ctx := context.Background()
		if err := c.sessionRepo.UpdateStatus(ctx, sessionID, session.WhatsAppStatusConnected); err != nil {
			c.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao atualizar status para connected")
		} else {
			c.logger.Info().Str("session_id", sessionID.String()).Msg("Status atualizado para connected com sucesso")
		}

	case *events.LoggedOut:
		c.logger.Info().
			Str("session_id", sessionID.String()).
			Int("reason", int(e.Reason)).
			Msg("WhatsApp desconectado")

	case *events.Presence:
		if e.Unavailable {
			if e.LastSeen.IsZero() {
				c.logger.Info().
					Str("session_id", sessionID.String()).
					Str("from", e.From.String()).
					Msg("Usuário ficou offline")
			} else {
				c.logger.Info().
					Str("session_id", sessionID.String()).
					Str("from", e.From.String()).
					Str("last_seen", e.LastSeen.String()).
					Msg("Usuário ficou offline")
			}
		} else {
			c.logger.Info().
				Str("session_id", sessionID.String()).
				Str("from", e.From.String()).
				Msg("Usuário ficou online")
		}

	default:
		c.logger.Debug().
			Str("session_id", sessionID.String()).
			Str("event_type", fmt.Sprintf("%T", evt)).
			Msg("Evento recebido")
	}

	// Chamar handler externo se configurado
	if c.eventHandler != nil {
		c.eventHandler.HandleEvent(sessionID, evt)
	}
}

// configureMediaDownloader configura o MediaDownloader para uma sessão específica
func (c *WhatsAppClient) configureMediaDownloader(sessionID uuid.UUID, client *whatsmeow.Client) {
	if c.minioClient == nil {
		return // MinIO não está habilitado
	}

	// Criar MediaDownloader para esta sessão
	mediaDownloader := NewMediaDownloader(client, c.minioClient)

	// Configurar o MediaDownloader no StorageHandler se possível
	if compositeHandler, ok := c.eventHandler.(*CompositeEventHandler); ok {
		compositeHandler.SetMediaDownloader(sessionID, mediaDownloader)
	}

	c.logger.Info().
		Str("session_id", sessionID.String()).
		Msg("MediaDownloader configurado com sucesso")
}
