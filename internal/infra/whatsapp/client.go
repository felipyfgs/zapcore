package whatsapp

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"zapcore/internal/domain/session"
	"zapcore/internal/domain/whatsapp"
	"zapcore/internal/infra/repository"
	"zapcore/pkg/logger"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/mdp/qrterminal/v3"
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
	container       *sqlstore.Container
	clients         map[uuid.UUID]*whatsmeow.Client
	clientsMutex    sync.RWMutex
	httpClients     map[uuid.UUID]*resty.Client
	httpMutex       sync.RWMutex
	qrChannels      map[uuid.UUID]<-chan whatsmeow.QRChannelItem
	qrChannelsMutex sync.RWMutex
	killChannels    map[uuid.UUID]chan bool
	killMutex       sync.RWMutex
	sessionRepo     interface {
		GetActiveSessions(ctx context.Context) ([]*repository.SessionData, error)
		UpdateJID(ctx context.Context, sessionID uuid.UUID, jid string) error
		UpdateStatus(ctx context.Context, sessionID uuid.UUID, status session.WhatsAppSessionStatus) error
	}
	logger       *logger.Logger
	eventHandler EventHandler
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
	GetActiveSessions(ctx context.Context) ([]*repository.SessionData, error)
	UpdateJID(ctx context.Context, sessionID uuid.UUID, jid string) error
	UpdateStatus(ctx context.Context, sessionID uuid.UUID, status session.WhatsAppSessionStatus) error
}, zeroLogger zerolog.Logger, eventHandler EventHandler) *WhatsAppClient {
	return &WhatsAppClient{
		container:    dbContainer,
		clients:      make(map[uuid.UUID]*whatsmeow.Client),
		httpClients:  make(map[uuid.UUID]*resty.Client),
		qrChannels:   make(map[uuid.UUID]<-chan whatsmeow.QRChannelItem),
		killChannels: make(map[uuid.UUID]chan bool),
		sessionRepo:  sessionRepo,
		logger:       logger.NewFromZerolog(zeroLogger),
		eventHandler: eventHandler,
	}
}

// ConnectOnStartup reconecta automaticamente sessões ativas com JID
func (c *WhatsAppClient) ConnectOnStartup(ctx context.Context) error {
	c.logger.Info().Msg("Iniciando reconexão automática de sessões ativas")

	// Buscar sessões ativas com JID no banco
	sessions, err := c.sessionRepo.GetActiveSessions(ctx)
	if err != nil {
		c.logger.Error().Err(err).Msg("Erro ao buscar sessões ativas para reconexão")
		return fmt.Errorf("erro ao buscar sessões ativas: %w", err)
	}

	if len(sessions) == 0 {
		c.logger.Info().Msg("Nenhuma sessão ativa encontrada para reconexão")
		return nil
	}

	c.logger.Info().Int("count", len(sessions)).Msg("Sessões ativas encontradas para reconexão")

	// Reconectar cada sessão em paralelo
	var wg sync.WaitGroup
	reconnectedCount := 0
	failedCount := 0
	var mu sync.Mutex

	for _, session := range sessions {
		wg.Add(1)
		go func(sess *repository.SessionData) {
			defer wg.Done()

			if err := c.reconnectSession(ctx, sess); err != nil {
				c.logger.Error().
					Err(err).
					Str("session_id", sess.ID.String()).
					Str("session_name", sess.Name).
					Str("jid", sess.JID).
					Msg("Falha ao reconectar sessão")

				mu.Lock()
				failedCount++
				mu.Unlock()
			} else {
				c.logger.Info().
					Str("session_id", sess.ID.String()).
					Str("session_name", sess.Name).
					Str("jid", sess.JID).
					Msg("Sessão reconectada com sucesso")

				mu.Lock()
				reconnectedCount++
				mu.Unlock()
			}
		}(session)
	}

	// Aguardar todas as reconexões
	wg.Wait()

	c.logger.Info().
		Int("total", len(sessions)).
		Int("reconnected", reconnectedCount).
		Int("failed", failedCount).
		Msg("Processo de reconexão automática concluído")

	return nil
}

// reconnectSession reconecta uma sessão específica usando o JID armazenado
func (c *WhatsAppClient) reconnectSession(ctx context.Context, session *repository.SessionData) error {
	// Parse do JID
	jid, err := types.ParseJID(session.JID)
	if err != nil {
		return fmt.Errorf("erro ao fazer parse do JID %s: %w", session.JID, err)
	}

	// Obter device store usando o JID
	deviceStore, err := c.container.GetDevice(ctx, jid)
	if err != nil {
		return fmt.Errorf("erro ao obter device store para JID %s: %w", session.JID, err)
	}

	// Verificar se o device está autenticado
	if deviceStore.ID == nil {
		return fmt.Errorf("device store não possui ID válido para sessão %s", session.ID.String())
	}

	// Configurar propriedades do device
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_UNKNOWN.Enum()
	store.DeviceProps.Os = proto.String("ZapCore")

	// Criar logger para o cliente
	clientLog := waLog.Stdout("Client", "INFO", true)

	// Criar cliente WhatsApp
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// Adicionar event handler
	client.AddEventHandler(func(evt interface{}) {
		c.handleWhatsAppEvent(session.ID, evt)
	})

	// Conectar (sem QR code, pois já está autenticado)
	err = client.Connect()
	if err != nil {
		return fmt.Errorf("erro ao conectar cliente para sessão %s: %w", session.ID.String(), err)
	}

	// Aguardar um momento para garantir que a conexão seja estabelecida
	time.Sleep(100 * time.Millisecond)

	// Verificar se o cliente está realmente conectado antes de armazenar
	if !client.IsConnected() {
		return fmt.Errorf("cliente não conseguiu se conectar para sessão %s", session.ID.String())
	}

	// Armazenar cliente com verificação de integridade
	c.clientsMutex.Lock()
	c.clients[session.ID] = client

	// Verificar se o cliente foi realmente armazenado
	if storedClient, exists := c.clients[session.ID]; exists && storedClient == client {
		c.logger.Info().
			Str("session_id", session.ID.String()).
			Int("total_clients_after", len(c.clients)).
			Bool("is_connected", client.IsConnected()).
			Msg("✅ Cliente armazenado com sucesso no mapa durante reconexão")
	} else {
		c.logger.Error().
			Str("session_id", session.ID.String()).
			Msg("❌ Falha ao armazenar cliente no mapa")
	}
	c.clientsMutex.Unlock()

	// Criar cliente HTTP
	httpClient := c.createHTTPClient()
	c.httpMutex.Lock()
	c.httpClients[session.ID] = httpClient
	c.httpMutex.Unlock()

	// Criar canal de kill para controlar a conexão
	c.killMutex.Lock()
	c.killChannels[session.ID] = make(chan bool)
	c.killMutex.Unlock()

	// Iniciar goroutine para manter cliente vivo
	go c.keepClientAlive(session.ID)

	return nil
}

// Connect estabelece conexão com o WhatsApp seguindo o padrão do wuzapi
func (c *WhatsAppClient) Connect(ctx context.Context, sessionID uuid.UUID) error {
	c.logger.Info().Str("session_id", sessionID.String()).Msg("Iniciando conexão com WhatsApp")

	// Atualizar status para "connecting"
	if err := c.sessionRepo.UpdateStatus(ctx, sessionID, session.WhatsAppStatusConnecting); err != nil {
		c.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao atualizar status para connecting")
	}

	// Verificar se já existe um cliente conectado
	c.clientsMutex.RLock()
	if client, exists := c.clients[sessionID]; exists && client.IsConnected() {
		c.clientsMutex.RUnlock()
		c.logger.Info().Str("session_id", sessionID.String()).Msg("Cliente já conectado")
		return nil
	}
	c.clientsMutex.RUnlock()

	// Verificar se a sessão já tem JID (está autenticada)
	sessions, err := c.sessionRepo.GetActiveSessions(ctx)
	if err != nil {
		c.logger.Error().Err(err).Msg("Erro ao buscar sessões para verificar JID")
		return fmt.Errorf("erro ao verificar estado da sessão: %w", err)
	}

	// Procurar a sessão específica
	var sessionData *repository.SessionData
	for _, session := range sessions {
		if session.ID == sessionID {
			sessionData = session
			break
		}
	}

	// Criar canal de kill para controlar a conexão
	c.killMutex.Lock()
	c.killChannels[sessionID] = make(chan bool)
	c.killMutex.Unlock()

	// Decidir como conectar baseado no estado da sessão
	if sessionData != nil && sessionData.JID != "" {
		// Sessão já autenticada, usar reconnectSession
		c.logger.Info().
			Str("session_id", sessionID.String()).
			Str("jid", sessionData.JID).
			Msg("Sessão já autenticada, reconectando")

		go func() {
			if err := c.reconnectSession(ctx, sessionData); err != nil {
				c.logger.Error().Err(err).Msg("Erro na reconexão")
			}
		}()
	} else {
		// Nova sessão, usar startClient para QR code
		c.logger.Info().
			Str("session_id", sessionID.String()).
			Msg("Nova sessão, iniciando processo de autenticação")

		go c.startClient(sessionID)
	}

	c.logger.Info().Str("session_id", sessionID.String()).Msg("Processo de conexão iniciado")
	return nil
}

// startClient inicia o cliente WhatsApp (baseado no wuzapi)
func (c *WhatsAppClient) startClient(sessionID uuid.UUID) {
	c.logger.Info().Str("session_id", sessionID.String()).Msg("Iniciando cliente WhatsApp")

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

	// Verificar se precisa de autenticação
	if client.Store.ID == nil {
		// Não está autenticado, precisa de QR code
		c.logger.Info().Str("session_id", sessionID.String()).Msg("Cliente não autenticado, gerando QR code")

		qrChan, err := client.GetQRChannel(context.Background())
		if err != nil {
			c.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao obter canal QR")
			return
		}

		// Conectar primeiro para poder gerar QR
		err = client.Connect()
		if err != nil {
			c.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao conectar cliente")
			return
		}

		// Armazenar cliente
		c.clientsMutex.Lock()
		c.clients[sessionID] = client
		c.clientsMutex.Unlock()

		// Processar eventos QR
		c.processQREvents(sessionID, qrChan)
	} else {
		// Já está autenticado, apenas conectar
		c.logger.Info().Str("session_id", sessionID.String()).Msg("Cliente já autenticado, conectando")

		err := client.Connect()
		if err != nil {
			c.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao conectar cliente")
			return
		}

		// Armazenar cliente
		c.clientsMutex.Lock()
		c.clients[sessionID] = client
		c.clientsMutex.Unlock()
	}

	// Loop para manter cliente vivo
	c.keepClientAlive(sessionID)
}

// Disconnect encerra a conexão
func (c *WhatsAppClient) Disconnect(ctx context.Context, sessionID uuid.UUID) error {
	c.logger.Info().Str("session_id", sessionID.String()).Msg("Desconectando cliente WhatsApp")

	// Enviar sinal de kill
	c.killMutex.RLock()
	if killChan, exists := c.killChannels[sessionID]; exists {
		select {
		case killChan <- true:
		default:
		}
	}
	c.killMutex.RUnlock()

	// Remover cliente
	c.clientsMutex.Lock()
	if client, exists := c.clients[sessionID]; exists {
		client.Disconnect()
		delete(c.clients, sessionID)
	}
	c.clientsMutex.Unlock()

	// Remover cliente HTTP
	c.httpMutex.Lock()
	delete(c.httpClients, sessionID)
	c.httpMutex.Unlock()

	// Remover canais
	c.qrChannelsMutex.Lock()
	delete(c.qrChannels, sessionID)
	c.qrChannelsMutex.Unlock()

	c.killMutex.Lock()
	delete(c.killChannels, sessionID)
	c.killMutex.Unlock()

	c.logger.Info().Str("session_id", sessionID.String()).Msg("Cliente WhatsApp desconectado")
	return nil
}

// processQREvents processa eventos do canal QR (baseado no wuzapi)
func (c *WhatsAppClient) processQREvents(sessionID uuid.UUID, qrChan <-chan whatsmeow.QRChannelItem) {
	c.logger.Info().Str("session_id", sessionID.String()).Msg("Processando eventos QR")

	for evt := range qrChan {
		switch evt.Event {
		case "code":
			c.logger.Info().Str("session_id", sessionID.String()).Msg("QR Code gerado")

			// Exibir QR code no terminal (como no wuzapi)
			c.logger.Info().
				Str("session_id", sessionID.String()).
				Str("qr_code", evt.Code).
				Msg("=== QR CODE GERADO ===")

			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)

			c.logger.Info().
				Str("session_id", sessionID.String()).
				Msg("=== Escaneie o código acima com seu WhatsApp ===")

			// Gerar QR code em base64 para armazenar
			image, err := qrcode.Encode(evt.Code, qrcode.Medium, 256)
			if err != nil {
				c.logger.Error().Err(err).Msg("Erro ao gerar QR code em base64")
			} else {
				base64QR := "data:image/png;base64," + base64.StdEncoding.EncodeToString(image)

				// Armazenar QR code
				c.qrChannelsMutex.Lock()
				// Aqui você pode salvar o QR code no banco se necessário
				// Por enquanto, apenas logamos
				c.logger.Info().Str("session_id", sessionID.String()).Str("qr_base64", base64QR).Msg("QR Code base64 gerado")
				c.qrChannelsMutex.Unlock()
			}

		case "timeout":
			c.logger.Warn().
				Str("session_id", sessionID.String()).
				Msg("⏰ QR Code expirou - Tente conectar novamente")

			// Limpar cliente
			c.clientsMutex.Lock()
			if client, exists := c.clients[sessionID]; exists {
				client.Disconnect()
				delete(c.clients, sessionID)
			}
			c.clientsMutex.Unlock()

			// Enviar sinal de kill
			c.killMutex.RLock()
			if killChan, exists := c.killChannels[sessionID]; exists {
				select {
				case killChan <- true:
				default:
				}
			}
			c.killMutex.RUnlock()
			return

		case "success":
			c.logger.Info().
				Str("session_id", sessionID.String()).
				Msg("✅ QR Code escaneado com sucesso")

			// QR foi escaneado, sair do loop
			return

		default:
			c.logger.Info().Str("session_id", sessionID.String()).Str("event", evt.Event).Msg("Evento QR recebido")
		}
	}
}

// keepClientAlive mantém o cliente vivo (baseado no wuzapi)
func (c *WhatsAppClient) keepClientAlive(sessionID uuid.UUID) {
	c.logger.Info().Str("session_id", sessionID.String()).Msg("Iniciando loop de manutenção do cliente")

	c.killMutex.RLock()
	killChan, exists := c.killChannels[sessionID]
	c.killMutex.RUnlock()

	if !exists {
		c.logger.Error().Str("session_id", sessionID.String()).Msg("Canal de kill não encontrado")
		return
	}

	c.logger.Debug().Str("session_id", sessionID.String()).Msg("Loop de manutenção iniciado, aguardando sinais")

	for {
		select {
		case <-killChan:
			c.logger.Info().Str("session_id", sessionID.String()).Msg("Recebido sinal de kill, encerrando cliente")

			// Limpar cliente
			c.clientsMutex.Lock()
			if client, exists := c.clients[sessionID]; exists {
				c.logger.Debug().Str("session_id", sessionID.String()).Msg("Desconectando e removendo cliente do mapa")
				client.Disconnect()
				delete(c.clients, sessionID)
			} else {
				c.logger.Warn().Str("session_id", sessionID.String()).Msg("Cliente não encontrado no mapa durante kill")
			}
			c.clientsMutex.Unlock()

			c.logger.Info().Str("session_id", sessionID.String()).Msg("Cliente removido, encerrando loop de manutenção")
			return
		default:
			// Simples como no wuzapi - apenas sleep
			time.Sleep(1000 * time.Millisecond)
			// Log comentado para evitar spam (como no wuzapi)
			// c.logger.Debug().Str("session_id", sessionID.String()).Msg("Loop the loop")
		}
	}
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
		return whatsapp.StatusConnecting, nil
	}

	return whatsapp.StatusDisconnected, nil
}

// SendTextMessage envia mensagem de texto
func (c *WhatsAppClient) SendTextMessage(ctx context.Context, req *whatsapp.SendTextRequest) (*whatsapp.MessageResponse, error) {
	c.logger.Debug().
		Str("session_id", req.SessionID.String()).
		Str("to_jid", req.ToJID).
		Int("content_length", len(req.Content)).
		Msg("SendTextMessage chamado")

	client, err := c.getClient(req.SessionID)
	if err != nil {
		c.logger.Error().Err(err).Str("session_id", req.SessionID.String()).Msg("Erro ao obter cliente")
		return nil, err
	}

	c.logger.Debug().Str("session_id", req.SessionID.String()).Msg("Cliente obtido com sucesso")

	jid, err := c.parseJID(req.ToJID)
	if err != nil {
		c.logger.Error().Err(err).Str("to_jid", req.ToJID).Msg("Erro ao fazer parse do JID")
		return nil, fmt.Errorf("JID inválido: %w", err)
	}

	c.logger.Debug().Str("parsed_jid", jid.String()).Msg("JID parseado com sucesso")

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

	// Obter dados da imagem (de io.Reader ou URL)
	mediaReader, err := c.getMediaData(ctx, req.ImageData, req.ImageURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter dados da imagem: %w", err)
	}

	// Ler dados da imagem
	imageData, err := io.ReadAll(mediaReader)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler dados da imagem: %w", err)
	}

	// Validar mídia de imagem
	if err := validateImageMedia(imageData, req.MimeType); err != nil {
		return nil, fmt.Errorf("validação de imagem falhou: %w", err)
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

	// Obter dados do áudio (de io.Reader ou URL)
	mediaReader, err := c.getMediaData(ctx, req.AudioData, req.AudioURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter dados do áudio: %w", err)
	}

	// Ler dados do áudio
	audioData, err := io.ReadAll(mediaReader)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler dados do áudio: %w", err)
	}

	// Validar mídia de áudio
	if err := validateAudioMedia(audioData, req.MimeType); err != nil {
		return nil, fmt.Errorf("validação de áudio falhou: %w", err)
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

// SendVideoMessage envia vídeo
func (c *WhatsAppClient) SendVideoMessage(ctx context.Context, req *whatsapp.SendVideoRequest) (*whatsapp.MessageResponse, error) {
	client, err := c.getClient(req.SessionID)
	if err != nil {
		return nil, err
	}

	jid, err := c.parseJID(req.ToJID)
	if err != nil {
		return nil, fmt.Errorf("JID inválido: %w", err)
	}

	// Obter dados do vídeo (de io.Reader ou URL)
	mediaReader, err := c.getMediaData(ctx, req.VideoData, req.VideoURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter dados do vídeo: %w", err)
	}

	// Ler dados do vídeo
	videoData, err := io.ReadAll(mediaReader)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler dados do vídeo: %w", err)
	}

	// Validar mídia de vídeo
	if err := validateVideoMedia(videoData, req.MimeType); err != nil {
		return nil, fmt.Errorf("validação de vídeo falhou: %w", err)
	}

	// Fazer upload do vídeo
	uploaded, err := client.Upload(ctx, videoData, whatsmeow.MediaVideo)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer upload do vídeo: %w", err)
	}

	// Criar mensagem de vídeo
	message := &waProto.Message{
		VideoMessage: &waProto.VideoMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(req.MimeType),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(videoData))),
			Caption:       proto.String(req.Caption),
		},
	}

	// Adicionar contexto de resposta se especificado
	if req.ReplyToID != "" {
		message.VideoMessage.ContextInfo = &waProto.ContextInfo{
			StanzaID: proto.String(req.ReplyToID),
		}
	}

	resp, err := client.SendMessage(ctx, jid, message)
	if err != nil {
		return nil, fmt.Errorf("erro ao enviar vídeo: %w", err)
	}

	return &whatsapp.MessageResponse{
		MessageID: resp.ID,
		Status:    "sent",
		Timestamp: resp.Timestamp.Unix(),
	}, nil
}

// SendStickerMessage envia sticker
func (c *WhatsAppClient) SendStickerMessage(ctx context.Context, req *whatsapp.SendStickerRequest) (*whatsapp.MessageResponse, error) {
	client, err := c.getClient(req.SessionID)
	if err != nil {
		return nil, err
	}

	jid, err := c.parseJID(req.ToJID)
	if err != nil {
		return nil, fmt.Errorf("JID inválido: %w", err)
	}

	// Obter dados do sticker (de io.Reader ou URL)
	mediaReader, err := c.getMediaData(ctx, req.StickerData, req.StickerURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter dados do sticker: %w", err)
	}

	// Ler dados do sticker
	stickerData, err := io.ReadAll(mediaReader)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler dados do sticker: %w", err)
	}

	// Validar mídia de sticker
	if err := validateStickerMedia(stickerData, req.MimeType); err != nil {
		return nil, fmt.Errorf("validação de sticker falhou: %w", err)
	}

	// Fazer upload do sticker
	uploaded, err := client.Upload(ctx, stickerData, whatsmeow.MediaImage)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer upload do sticker: %w", err)
	}

	// Criar mensagem de sticker
	message := &waProto.Message{
		StickerMessage: &waProto.StickerMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(req.MimeType),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(stickerData))),
		},
	}

	// Adicionar contexto de resposta se especificado
	if req.ReplyToID != "" {
		message.StickerMessage.ContextInfo = &waProto.ContextInfo{
			StanzaID: proto.String(req.ReplyToID),
		}
	}

	resp, err := client.SendMessage(ctx, jid, message)
	if err != nil {
		return nil, fmt.Errorf("erro ao enviar sticker: %w", err)
	}

	return &whatsapp.MessageResponse{
		MessageID: resp.ID,
		Status:    "sent",
		Timestamp: resp.Timestamp.Unix(),
	}, nil
}

// SendDocumentMessage envia documento
func (c *WhatsAppClient) SendDocumentMessage(ctx context.Context, req *whatsapp.SendDocumentRequest) (*whatsapp.MessageResponse, error) {
	client, err := c.getClient(req.SessionID)
	if err != nil {
		return nil, err
	}

	jid, err := c.parseJID(req.ToJID)
	if err != nil {
		return nil, fmt.Errorf("JID inválido: %w", err)
	}

	// Obter dados do documento (de io.Reader ou URL)
	mediaReader, err := c.getMediaData(ctx, req.DocumentData, req.DocumentURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter dados do documento: %w", err)
	}

	// Ler dados do documento
	documentData, err := io.ReadAll(mediaReader)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler dados do documento: %w", err)
	}

	// Validar mídia de documento
	if err := validateDocumentMedia(documentData, req.MimeType); err != nil {
		return nil, fmt.Errorf("validação de documento falhou: %w", err)
	}

	// Fazer upload do documento
	uploaded, err := client.Upload(ctx, documentData, whatsmeow.MediaDocument)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer upload do documento: %w", err)
	}

	// Criar mensagem de documento
	documentMsg := &waProto.DocumentMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(req.MimeType),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(documentData))),
		FileName:      proto.String(req.FileName),
	}

	// Adicionar caption se fornecido
	if req.Caption != "" {
		documentMsg.Caption = proto.String(req.Caption)
	}

	message := &waProto.Message{
		DocumentMessage: documentMsg,
	}

	// Adicionar contexto de resposta se especificado
	if req.ReplyToID != "" {
		message.DocumentMessage.ContextInfo = &waProto.ContextInfo{
			StanzaID: proto.String(req.ReplyToID),
		}
	}

	resp, err := client.SendMessage(ctx, jid, message)
	if err != nil {
		return nil, fmt.Errorf("erro ao enviar documento: %w", err)
	}

	return &whatsapp.MessageResponse{
		MessageID: resp.ID,
		Status:    "sent",
		Timestamp: resp.Timestamp.Unix(),
	}, nil
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

	c.logger.Debug().
		Str("session_id", sessionID.String()).
		Int("total_clients", len(c.clients)).
		Msg("🔍 Buscando cliente para sessão")

	// Listar todas as sessões disponíveis para debug
	if len(c.clients) == 0 {
		c.logger.Warn().Msg("❌ Mapa de clientes está vazio")
	} else {
		for id := range c.clients {
			c.logger.Debug().
				Str("available_session", id.String()).
				Bool("is_connected", c.clients[id].IsConnected()).
				Msg("📱 Cliente disponível")
		}
	}

	client, exists := c.clients[sessionID]
	if !exists {
		c.logger.Error().
			Str("session_id", sessionID.String()).
			Int("total_clients", len(c.clients)).
			Msg("❌ Cliente não encontrado no mapa")

		// Tentar buscar todas as sessões ativas para debug
		ctx := context.Background()
		if sessions, err := c.sessionRepo.GetActiveSessions(ctx); err == nil {
			c.logger.Debug().Int("active_sessions_in_db", len(sessions)).Msg("Sessões ativas no banco")
			for _, session := range sessions {
				c.logger.Debug().
					Str("db_session_id", session.ID.String()).
					Str("db_session_jid", session.JID).
					Msg("Sessão ativa no banco")
			}
		}

		return nil, fmt.Errorf("cliente não encontrado para sessão %s", sessionID.String())
	}

	if !client.IsConnected() {
		c.logger.Error().
			Str("session_id", sessionID.String()).
			Msg("⚠️ Cliente encontrado mas não está conectado")
		return nil, fmt.Errorf("cliente não está conectado para sessão %s", sessionID.String())
	}

	c.logger.Debug().
		Str("session_id", sessionID.String()).
		Msg("✅ Cliente encontrado e conectado")

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

// handleWhatsAppEvent manipula eventos do WhatsApp (baseado no wuzapi)
func (c *WhatsAppClient) handleWhatsAppEvent(sessionID uuid.UUID, evt interface{}) {
	switch e := evt.(type) {
	case *events.Message:
		c.logger.Info().
			Str("session_id", sessionID.String()).
			Str("message_id", e.Info.ID).
			Str("from", e.Info.SourceString()).
			Str("pushname", e.Info.PushName).
			Msg("Mensagem recebida")

	case *events.Receipt:
		c.logger.Info().
			Str("session_id", sessionID.String()).
			Strs("message_ids", e.MessageIDs).
			Str("type", string(e.Type)).
			Str("source", e.SourceString()).
			Msg("Recibo recebido")

	case *events.Connected:
		c.logger.Info().
			Str("session_id", sessionID.String()).
			Msg("Cliente conectado ao WhatsApp")

		// Atualizar status para "connected"
		ctx := context.Background()
		if err := c.sessionRepo.UpdateStatus(ctx, sessionID, session.WhatsAppStatusConnected); err != nil {
			c.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao atualizar status para connected")
		}

		// Enviar presença disponível
		c.clientsMutex.RLock()
		if client, exists := c.clients[sessionID]; exists {
			if len(client.Store.PushName) > 0 {
				err := client.SendPresence(types.PresenceAvailable)
				if err != nil {
					c.logger.Warn().Err(err).Msg("Falha ao enviar presença disponível")
				} else {
					c.logger.Info().Msg("Presença marcada como disponível")
				}
			}
		}
		c.clientsMutex.RUnlock()

	case *events.PairSuccess:
		c.logger.Info().
			Str("session_id", sessionID.String()).
			Str("jid", e.ID.String()).
			Str("business_name", e.BusinessName).
			Str("platform", e.Platform).
			Msg("Pareamento QR realizado com sucesso")

		c.logger.Info().
			Str("session_id", sessionID.String()).
			Str("jid", e.ID.String()).
			Str("business_name", e.BusinessName).
			Str("platform", e.Platform).
			Msg("🎉 Pareamento realizado com sucesso")

		// Salvar JID no banco de dados para reconexão futura
		if c.eventHandler != nil {
			c.eventHandler.HandleEvent(sessionID, &PairSuccessEvent{
				SessionID:    sessionID,
				JID:          e.ID.String(),
				BusinessName: e.BusinessName,
				Platform:     e.Platform,
			})
		}

	case *events.LoggedOut:
		c.logger.Info().
			Str("session_id", sessionID.String()).
			Str("reason", e.Reason.String()).
			Msg("Cliente deslogado")

		// Enviar sinal de kill
		c.killMutex.RLock()
		if killChan, exists := c.killChannels[sessionID]; exists {
			select {
			case killChan <- true:
			default:
			}
		}
		c.killMutex.RUnlock()

	case *events.StreamReplaced:
		c.logger.Info().
			Str("session_id", sessionID.String()).
			Msg("Stream substituído")

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

// downloadFromURL baixa mídia de uma URL pública
func (c *WhatsAppClient) downloadFromURL(ctx context.Context, mediaURL string) (io.Reader, error) {
	// Validar URL
	parsedURL, err := url.Parse(mediaURL)
	if err != nil {
		return nil, fmt.Errorf("URL inválida: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("esquema de URL não suportado: %s", parsedURL.Scheme)
	}

	// Criar cliente HTTP com timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Fazer requisição HTTP
	req, err := http.NewRequestWithContext(ctx, "GET", mediaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %w", err)
	}

	// Adicionar User-Agent
	req.Header.Set("User-Agent", "ZapCore/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao baixar mídia: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erro HTTP: %d %s", resp.StatusCode, resp.Status)
	}

	// Ler dados da resposta
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler dados da resposta: %w", err)
	}

	return bytes.NewReader(data), nil
}

// validateMimeType valida se o tipo MIME é suportado para o tipo de mídia
func validateMimeType(mimeType string, supportedMimes []string) error {
	if mimeType == "" {
		return fmt.Errorf("tipo MIME não especificado")
	}

	for _, supported := range supportedMimes {
		if strings.EqualFold(mimeType, supported) {
			return nil
		}
	}

	return fmt.Errorf("tipo MIME não suportado: %s", mimeType)
}

// validateMediaSize valida se o tamanho da mídia está dentro do limite
func validateMediaSize(data []byte, maxSize int64, mediaType string) error {
	size := int64(len(data))
	if size > maxSize {
		return fmt.Errorf("arquivo %s muito grande: %d bytes (máximo: %d bytes)",
			mediaType, size, maxSize)
	}
	return nil
}

// validateImageMedia valida mídia de imagem
func validateImageMedia(data []byte, mimeType string) error {
	if err := validateMimeType(mimeType, SupportedImageMimes); err != nil {
		return err
	}
	return validateMediaSize(data, MaxImageSize, "imagem")
}

// validateVideoMedia valida mídia de vídeo
func validateVideoMedia(data []byte, mimeType string) error {
	if err := validateMimeType(mimeType, SupportedVideoMimes); err != nil {
		return err
	}
	return validateMediaSize(data, MaxVideoSize, "vídeo")
}

// validateAudioMedia valida mídia de áudio
func validateAudioMedia(data []byte, mimeType string) error {
	if err := validateMimeType(mimeType, SupportedAudioMimes); err != nil {
		return err
	}
	return validateMediaSize(data, MaxAudioSize, "áudio")
}

// validateDocumentMedia valida mídia de documento
func validateDocumentMedia(data []byte, mimeType string) error {
	if err := validateMimeType(mimeType, SupportedDocumentMimes); err != nil {
		return err
	}
	return validateMediaSize(data, MaxDocumentSize, "documento")
}

// validateStickerMedia valida mídia de sticker
func validateStickerMedia(data []byte, mimeType string) error {
	if err := validateMimeType(mimeType, SupportedStickerMimes); err != nil {
		return err
	}
	return validateMediaSize(data, MaxStickerSize, "sticker")
}

// getMediaData obtém dados de mídia de io.Reader, URL ou caminho local
func (c *WhatsAppClient) getMediaData(ctx context.Context, mediaData io.Reader, mediaURL string) (io.Reader, error) {
	if mediaData != nil {
		return mediaData, nil
	}

	if mediaURL != "" {
		// Verificar se é URL HTTP/HTTPS ou caminho local
		if strings.HasPrefix(mediaURL, "http://") || strings.HasPrefix(mediaURL, "https://") {
			// É uma URL pública - fazer download
			return c.downloadFromURL(ctx, mediaURL)
		} else {
			// É um caminho local - abrir arquivo
			return c.openLocalFile(mediaURL)
		}
	}

	return nil, fmt.Errorf("nenhuma fonte de mídia fornecida (dados ou URL)")
}

// openLocalFile abre um arquivo local e retorna um Reader
func (c *WhatsAppClient) openLocalFile(filePath string) (io.Reader, error) {
	// Verificar se o arquivo existe
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("arquivo não encontrado: %s", filePath)
		}
		return nil, fmt.Errorf("erro ao acessar arquivo: %w", err)
	}

	// Abrir o arquivo
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir arquivo: %w", err)
	}

	// Ler todo o conteúdo do arquivo
	data, err := io.ReadAll(file)
	file.Close() // Fechar o arquivo imediatamente após ler
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo: %w", err)
	}

	// Retornar um Reader com os dados
	return bytes.NewReader(data), nil
}
