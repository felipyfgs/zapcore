package whatsapp

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"zapcore/internal/domain/whatsapp"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// WhatsAppClient implementa a interface whatsapp.Client
type WhatsAppClient struct {
	container    *sqlstore.Container
	clients      map[uuid.UUID]*whatsmeow.Client
	clientsMutex sync.RWMutex
	httpClients  map[uuid.UUID]*resty.Client
	httpMutex    sync.RWMutex
	logger       zerolog.Logger
	eventHandler EventHandler
}

// EventHandler define a interface para manipular eventos do WhatsApp
type EventHandler interface {
	HandleEvent(sessionID uuid.UUID, event interface{})
}

// NewWhatsAppClient cria uma nova instância do cliente WhatsApp
func NewWhatsAppClient(dbContainer *sqlstore.Container, logger zerolog.Logger, eventHandler EventHandler) *WhatsAppClient {
	return &WhatsAppClient{
		container:    dbContainer,
		clients:      make(map[uuid.UUID]*whatsmeow.Client),
		httpClients:  make(map[uuid.UUID]*resty.Client),
		logger:       logger,
		eventHandler: eventHandler,
	}
}

// Connect estabelece conexão com o WhatsApp
func (c *WhatsAppClient) Connect(ctx context.Context, sessionID uuid.UUID) error {
	c.clientsMutex.Lock()
	defer c.clientsMutex.Unlock()

	// Verificar se já existe um cliente conectado
	if client, exists := c.clients[sessionID]; exists {
		if client.IsConnected() {
			return nil
		}
		// Remover cliente desconectado
		delete(c.clients, sessionID)
	}

	// Criar novo device store
	deviceStore := c.container.NewDevice()

	// Configurar propriedades do device
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_UNKNOWN.Enum()
	store.DeviceProps.Os = proto.String("ZapCore")

	// Criar logger para o cliente
	clientLog := waLog.Stdout("Client", "INFO", true)

	// Criar cliente WhatsApp
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// Adicionar event handler
	client.AddEventHandler(func(evt interface{}) {
		c.handleWhatsAppEvent(sessionID, evt)
	})

	// Criar cliente HTTP
	httpClient := c.createHTTPClient()
	c.httpMutex.Lock()
	c.httpClients[sessionID] = httpClient
	c.httpMutex.Unlock()

	// Se não está autenticado, obter canal QR antes de conectar
	if client.Store.ID == nil {
		_, err := client.GetQRChannel(ctx)
		if err != nil {
			c.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao obter canal QR")
			return fmt.Errorf("erro ao obter canal QR: %w", err)
		}
	}

	// Conectar
	err := client.Connect()
	if err != nil {
		c.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao conectar cliente WhatsApp")
		return fmt.Errorf("erro ao conectar: %w", err)
	}

	// Armazenar cliente
	c.clients[sessionID] = client

	c.logger.Info().Str("session_id", sessionID.String()).Msg("Cliente WhatsApp conectado com sucesso")
	return nil
}

// Disconnect encerra a conexão
func (c *WhatsAppClient) Disconnect(ctx context.Context, sessionID uuid.UUID) error {
	c.clientsMutex.Lock()
	defer c.clientsMutex.Unlock()

	client, exists := c.clients[sessionID]
	if !exists {
		return fmt.Errorf("cliente não encontrado para sessão %s", sessionID.String())
	}

	client.Disconnect()
	delete(c.clients, sessionID)

	// Remover cliente HTTP
	c.httpMutex.Lock()
	delete(c.httpClients, sessionID)
	c.httpMutex.Unlock()

	c.logger.Info().Str("session_id", sessionID.String()).Msg("Cliente WhatsApp desconectado")
	return nil
}

// GetQRCode gera QR Code para autenticação
func (c *WhatsAppClient) GetQRCode(ctx context.Context, sessionID uuid.UUID) (string, error) {
	c.clientsMutex.RLock()
	client, exists := c.clients[sessionID]
	c.clientsMutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("cliente não encontrado para sessão %s", sessionID.String())
	}

	if client.Store.ID != nil {
		return "", fmt.Errorf("cliente já está autenticado")
	}

	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		return "", fmt.Errorf("erro ao obter canal QR: %w", err)
	}

	// Aguardar pelo QR code
	select {
	case evt := <-qrChan:
		if evt.Event == "code" {
			// Gerar QR code em base64
			image, err := qrcode.Encode(evt.Code, qrcode.Medium, 256)
			if err != nil {
				return "", fmt.Errorf("erro ao gerar QR code: %w", err)
			}

			base64QR := "data:image/png;base64," + base64.StdEncoding.EncodeToString(image)
			return base64QR, nil
		}
		return "", fmt.Errorf("evento QR inesperado: %s", evt.Event)
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(30 * time.Second):
		return "", fmt.Errorf("timeout ao aguardar QR code")
	}
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
	c.clientsMutex.RLock()
	client, exists := c.clients[sessionID]
	c.clientsMutex.RUnlock()

	if !exists {
		return whatsapp.StatusDisconnected, nil
	}

	if client.IsConnected() {
		if client.Store.ID != nil {
			return whatsapp.StatusConnected, nil
		}
		return whatsapp.StatusQRCode, nil
	}

	return whatsapp.StatusDisconnected, nil
}

// SendTextMessage envia mensagem de texto
func (c *WhatsAppClient) SendTextMessage(ctx context.Context, req *whatsapp.SendTextRequest) (*whatsapp.MessageResponse, error) {
	client, err := c.getClient(req.SessionID)
	if err != nil {
		return nil, err
	}

	jid, err := c.parseJID(req.ToJID)
	if err != nil {
		return nil, fmt.Errorf("JID inválido: %w", err)
	}

	message := &waProto.Message{
		Conversation: proto.String(req.Content),
	}

	// Adicionar contexto de resposta se especificado
	if req.ReplyToID != "" {
		message.ExtendedTextMessage = &waProto.ExtendedTextMessage{
			Text: proto.String(req.Content),
			ContextInfo: &waProto.ContextInfo{
				StanzaID: proto.String(req.ReplyToID),
			},
		}
		message.Conversation = nil
	}

	resp, err := client.SendMessage(ctx, jid, message)
	if err != nil {
		return nil, fmt.Errorf("erro ao enviar mensagem: %w", err)
	}

	return &whatsapp.MessageResponse{
		MessageID: resp.ID,
		Status:    "sent",
		Timestamp: resp.Timestamp.Unix(),
	}, nil
}

// SendImageMessage envia imagem
func (c *WhatsAppClient) SendImageMessage(ctx context.Context, req *whatsapp.SendImageRequest) (*whatsapp.MessageResponse, error) {
	client, err := c.getClient(req.SessionID)
	if err != nil {
		return nil, err
	}

	jid, err := c.parseJID(req.ToJID)
	if err != nil {
		return nil, fmt.Errorf("JID inválido: %w", err)
	}

	// Ler dados da imagem
	imageData, err := io.ReadAll(req.ImageData)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler dados da imagem: %w", err)
	}

	// Fazer upload da imagem
	uploaded, err := client.Upload(ctx, imageData, whatsmeow.MediaImage)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer upload da imagem: %w", err)
	}

	// Criar mensagem de imagem
	message := &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(req.MimeType),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(imageData))),
			Caption:       proto.String(req.Caption),
		},
	}

	// Adicionar contexto de resposta se especificado
	if req.ReplyToID != "" {
		message.ImageMessage.ContextInfo = &waProto.ContextInfo{
			StanzaID: proto.String(req.ReplyToID),
		}
	}

	resp, err := client.SendMessage(ctx, jid, message)
	if err != nil {
		return nil, fmt.Errorf("erro ao enviar imagem: %w", err)
	}

	return &whatsapp.MessageResponse{
		MessageID: resp.ID,
		Status:    "sent",
		Timestamp: resp.Timestamp.Unix(),
	}, nil
}

// SendAudioMessage envia áudio
func (c *WhatsAppClient) SendAudioMessage(ctx context.Context, req *whatsapp.SendAudioRequest) (*whatsapp.MessageResponse, error) {
	client, err := c.getClient(req.SessionID)
	if err != nil {
		return nil, err
	}

	jid, err := c.parseJID(req.ToJID)
	if err != nil {
		return nil, fmt.Errorf("JID inválido: %w", err)
	}

	// Ler dados do áudio
	audioData, err := io.ReadAll(req.AudioData)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler dados do áudio: %w", err)
	}

	// Fazer upload do áudio
	uploaded, err := client.Upload(ctx, audioData, whatsmeow.MediaAudio)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer upload do áudio: %w", err)
	}

	// Criar mensagem de áudio
	message := &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(req.MimeType),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(audioData))),
			PTT:           proto.Bool(req.IsVoice),
		},
	}

	// Adicionar contexto de resposta se especificado
	if req.ReplyToID != "" {
		message.AudioMessage.ContextInfo = &waProto.ContextInfo{
			StanzaID: proto.String(req.ReplyToID),
		}
	}

	resp, err := client.SendMessage(ctx, jid, message)
	if err != nil {
		return nil, fmt.Errorf("erro ao enviar áudio: %w", err)
	}

	return &whatsapp.MessageResponse{
		MessageID: resp.ID,
		Status:    "sent",
		Timestamp: resp.Timestamp.Unix(),
	}, nil
}

// SendVideoMessage envia vídeo (não implementado)
func (c *WhatsAppClient) SendVideoMessage(ctx context.Context, req *whatsapp.SendVideoRequest) (*whatsapp.MessageResponse, error) {
	return nil, fmt.Errorf("SendVideoMessage não implementado ainda")
}

// SendStickerMessage envia sticker (não implementado)
func (c *WhatsAppClient) SendStickerMessage(ctx context.Context, req *whatsapp.SendStickerRequest) (*whatsapp.MessageResponse, error) {
	return nil, fmt.Errorf("SendStickerMessage não implementado ainda")
}

// SendDocumentMessage envia documento (não implementado)
func (c *WhatsAppClient) SendDocumentMessage(ctx context.Context, req *whatsapp.SendDocumentRequest) (*whatsapp.MessageResponse, error) {
	return nil, fmt.Errorf("SendDocumentMessage não implementado ainda")
}

// CreateGroup cria um grupo (implementação temporária)
func (c *WhatsAppClient) CreateGroup(ctx context.Context, req *whatsapp.CreateGroupRequest) (*whatsapp.GroupInfo, error) {
	// TODO: Implementar criação de grupo
	return nil, fmt.Errorf("CreateGroup não implementado ainda")
}

// LeaveGroup sai do grupo (implementação temporária)
func (c *WhatsAppClient) LeaveGroup(ctx context.Context, sessionID uuid.UUID, groupJID string) error {
	// TODO: Implementar saída do grupo
	return fmt.Errorf("LeaveGroup não implementado ainda")
}

// UpdateGroupParticipants atualiza participantes do grupo (implementação temporária)
func (c *WhatsAppClient) UpdateGroupParticipants(ctx context.Context, req *whatsapp.UpdateGroupParticipantsRequest) error {
	// TODO: Implementar atualização de participantes
	return fmt.Errorf("UpdateGroupParticipants não implementado ainda")
}

// GetGroupInfo obtém informações do grupo (implementação temporária)
func (c *WhatsAppClient) GetGroupInfo(ctx context.Context, sessionID uuid.UUID, groupJID string) (*whatsapp.GroupInfo, error) {
	// TODO: Implementar obtenção de informações do grupo
	return nil, fmt.Errorf("GetGroupInfo não implementado ainda")
}

// GetUserInfo obtém informações do usuário (implementação temporária)
func (c *WhatsAppClient) GetUserInfo(ctx context.Context, sessionID uuid.UUID, jids []string) ([]*whatsapp.UserInfo, error) {
	// TODO: Implementar obtenção de informações do usuário
	return nil, fmt.Errorf("GetUserInfo não implementado ainda")
}

// IsOnWhatsApp verifica se números estão no WhatsApp (implementação temporária)
func (c *WhatsAppClient) IsOnWhatsApp(ctx context.Context, sessionID uuid.UUID, phones []string) ([]*whatsapp.IsOnWhatsAppResponse, error) {
	// TODO: Implementar verificação de números no WhatsApp
	return nil, fmt.Errorf("IsOnWhatsApp não implementado ainda")
}

// createHTTPClient cria um cliente HTTP configurado
func (c *WhatsAppClient) createHTTPClient() *resty.Client {
	httpClient := resty.New()
	httpClient.SetRedirectPolicy(resty.FlexibleRedirectPolicy(15))
	httpClient.SetTimeout(30 * time.Second)
	httpClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	httpClient.OnError(func(req *resty.Request, err error) {
		if v, ok := err.(*resty.ResponseError); ok {
			c.logger.Debug().Str("response", v.Response.String()).Msg("resty error")
			c.logger.Error().Err(v.Err).Msg("resty error")
		}
	})

	return httpClient
}

// getClient obtém o cliente WhatsApp para uma sessão
func (c *WhatsAppClient) getClient(sessionID uuid.UUID) (*whatsmeow.Client, error) {
	c.clientsMutex.RLock()
	defer c.clientsMutex.RUnlock()

	client, exists := c.clients[sessionID]
	if !exists {
		return nil, fmt.Errorf("cliente não encontrado para sessão %s", sessionID.String())
	}

	if !client.IsConnected() {
		return nil, fmt.Errorf("cliente não está conectado para sessão %s", sessionID.String())
	}

	return client, nil
}

// parseJID converte string para types.JID
func (c *WhatsAppClient) parseJID(jidStr string) (types.JID, error) {
	if jidStr[0] == '+' {
		jidStr = jidStr[1:]
	}

	if !strings.ContainsRune(jidStr, '@') {
		return types.NewJID(jidStr, types.DefaultUserServer), nil
	}

	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return jid, fmt.Errorf("JID inválido: %w", err)
	}

	if jid.User == "" {
		return jid, fmt.Errorf("JID inválido: servidor não especificado")
	}

	return jid, nil
}

// handleWhatsAppEvent manipula eventos do WhatsApp
func (c *WhatsAppClient) handleWhatsAppEvent(sessionID uuid.UUID, evt interface{}) {
	switch e := evt.(type) {
	case *events.Message:
		c.logger.Info().
			Str("session_id", sessionID.String()).
			Str("message_id", e.Info.ID).
			Str("from", e.Info.SourceString()).
			Msg("Mensagem recebida")

	case *events.Receipt:
		c.logger.Info().
			Str("session_id", sessionID.String()).
			Strs("message_ids", e.MessageIDs).
			Str("type", string(e.Type)).
			Msg("Recibo recebido")

	case *events.Connected:
		c.logger.Info().
			Str("session_id", sessionID.String()).
			Msg("Cliente conectado")

	case *events.LoggedOut:
		c.logger.Info().
			Str("session_id", sessionID.String()).
			Str("reason", e.Reason.String()).
			Msg("Cliente deslogado")

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

// Métodos não implementados da interface whatsapp.Client
// TODO: Implementar estes métodos conforme necessário

// SendLocationMessage envia localização (não implementado)
func (c *WhatsAppClient) SendLocationMessage(ctx context.Context, req *whatsapp.SendLocationRequest) (*whatsapp.MessageResponse, error) {
	return nil, fmt.Errorf("SendLocationMessage não implementado ainda")
}

// SendContactMessage envia contato (não implementado)
func (c *WhatsAppClient) SendContactMessage(ctx context.Context, req *whatsapp.SendContactRequest) (*whatsapp.MessageResponse, error) {
	return nil, fmt.Errorf("SendContactMessage não implementado ainda")
}

// SendReactionMessage envia reação (não implementado)
func (c *WhatsAppClient) SendReactionMessage(ctx context.Context, req *whatsapp.SendReactionRequest) (*whatsapp.MessageResponse, error) {
	return nil, fmt.Errorf("SendReactionMessage não implementado ainda")
}

// SendPollMessage envia enquete (não implementado)
func (c *WhatsAppClient) SendPollMessage(ctx context.Context, req *whatsapp.SendPollRequest) (*whatsapp.MessageResponse, error) {
	return nil, fmt.Errorf("SendPollMessage não implementado ainda")
}

// EditMessage edita uma mensagem (não implementado)
func (c *WhatsAppClient) EditMessage(ctx context.Context, req *whatsapp.EditMessageRequest) (*whatsapp.MessageResponse, error) {
	return nil, fmt.Errorf("EditMessage não implementado ainda")
}

// RevokeMessage revoga uma mensagem (não implementado)
func (c *WhatsAppClient) RevokeMessage(ctx context.Context, req *whatsapp.RevokeMessageRequest) (*whatsapp.MessageResponse, error) {
	return nil, fmt.Errorf("RevokeMessage não implementado ainda")
}

// DownloadMedia faz download de mídia (não implementado)
func (c *WhatsAppClient) DownloadMedia(ctx context.Context, req *whatsapp.DownloadMediaRequest) ([]byte, error) {
	return nil, fmt.Errorf("DownloadMedia não implementado ainda")
}

// UploadMedia faz upload de mídia (não implementado)
func (c *WhatsAppClient) UploadMedia(ctx context.Context, req *whatsapp.UploadMediaRequest) (*whatsapp.UploadResponse, error) {
	return nil, fmt.Errorf("UploadMedia não implementado ainda")
}

// SetProxy configura proxy (não implementado)
func (c *WhatsAppClient) SetProxy(ctx context.Context, sessionID uuid.UUID, proxyURL string) error {
	return fmt.Errorf("SetProxy não implementado ainda")
}

// GetContacts obtém lista de contatos (não implementado)
func (c *WhatsAppClient) GetContacts(ctx context.Context, sessionID uuid.UUID) ([]*whatsapp.Contact, error) {
	return nil, fmt.Errorf("GetContacts não implementado ainda")
}

// GetChats obtém lista de chats (não implementado)
func (c *WhatsAppClient) GetChats(ctx context.Context, sessionID uuid.UUID) ([]*whatsapp.Chat, error) {
	return nil, fmt.Errorf("GetChats não implementado ainda")
}

// MarkAsRead marca mensagem como lida (não implementado)
func (c *WhatsAppClient) MarkAsRead(ctx context.Context, req *whatsapp.MarkAsReadRequest) error {
	return fmt.Errorf("MarkAsRead não implementado ainda")
}

// SendPresence define presença (não implementado)
func (c *WhatsAppClient) SendPresence(ctx context.Context, req *whatsapp.SendPresenceRequest) error {
	return fmt.Errorf("SendPresence não implementado ainda")
}

// GetGroupInviteLink obtém link de convite do grupo (não implementado)
func (c *WhatsAppClient) GetGroupInviteLink(ctx context.Context, sessionID uuid.UUID, groupJID string, reset bool) (string, error) {
	return "", fmt.Errorf("GetGroupInviteLink não implementado ainda")
}

// GetProfilePicture obtém foto de perfil (não implementado)
func (c *WhatsAppClient) GetProfilePicture(ctx context.Context, sessionID uuid.UUID, jid string) (*whatsapp.ProfilePictureInfo, error) {
	return nil, fmt.Errorf("GetProfilePicture não implementado ainda")
}

// IsConnected verifica se a sessão está conectada (não implementado)
func (c *WhatsAppClient) IsConnected(ctx context.Context, sessionID uuid.UUID) bool {
	return false
}

// IsLoggedIn verifica se a sessão está logada (não implementado)
func (c *WhatsAppClient) IsLoggedIn(ctx context.Context, sessionID uuid.UUID) bool {
	return false
}

// JoinGroupWithLink entra em grupo via link (não implementado)
func (c *WhatsAppClient) JoinGroupWithLink(ctx context.Context, sessionID uuid.UUID, link string) (*whatsapp.GroupInfo, error) {
	return nil, fmt.Errorf("JoinGroupWithLink não implementado ainda")
}

// SendChatPresence envia presença no chat (não implementado)
func (c *WhatsAppClient) SendChatPresence(ctx context.Context, req *whatsapp.SendChatPresenceRequest) error {
	return fmt.Errorf("SendChatPresence não implementado ainda")
}

// SubscribePresence se inscreve para receber atualizações de presença (não implementado)
func (c *WhatsAppClient) SubscribePresence(ctx context.Context, sessionID uuid.UUID, jid string) error {
	return fmt.Errorf("SubscribePresence não implementado ainda")
}

// SetGroupName define nome do grupo (não implementado)
func (c *WhatsAppClient) SetGroupName(ctx context.Context, sessionID uuid.UUID, groupJID, name string) error {
	return fmt.Errorf("SetGroupName não implementado ainda")
}

// SetGroupDescription define descrição do grupo (não implementado)
func (c *WhatsAppClient) SetGroupDescription(ctx context.Context, sessionID uuid.UUID, groupJID, description string) error {
	return fmt.Errorf("SetGroupDescription não implementado ainda")
}
