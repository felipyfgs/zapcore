package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/mdp/qrterminal/v3"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	"wamex/internal/domain"
	"wamex/pkg/logger"
)

func init() {
	// OBRIGATÓRIO para PostgreSQL - deve ser configurado ANTES de criar o container
	sqlstore.PostgresArrayWrapper = pq.Array
}

// WhatsAppClient representa um cliente WhatsApp ativo
type WhatsAppClient struct {
	Client       *whatsmeow.Client
	EventHandler uint32
	SessionID    string
	Status       domain.Status
	QRChannel    <-chan whatsmeow.QRChannelItem
	KillChannel  chan bool
	Repository   domain.SessionRepository
}

// WhatsAppService implementa a interface SessionService
type WhatsAppService struct {
	repository   domain.SessionRepository
	container    *sqlstore.Container
	clients      map[string]*WhatsAppClient
	clientsMutex sync.RWMutex
}

// NewWhatsAppService cria uma nova instância do serviço WhatsApp
func NewWhatsAppService(repo domain.SessionRepository, dbDialect string, dbSource string) (*WhatsAppService, error) {
	// Cria logger para o whatsmeow
	waLogger := waLog.Stdout("WhatsApp", "INFO", true)

	// Inicializa o container do sqlstore para PostgreSQL
	ctx := context.Background()
	container, err := sqlstore.New(ctx, dbDialect, dbSource, waLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create sqlstore container: %w", err)
	}

	// Upgrade schema automaticamente
	err = container.Upgrade(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade database schema: %w", err)
	}

	// Configura propriedades do device
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_UNKNOWN.Enum()
	osName := "WAMEX"
	store.DeviceProps.Os = &osName

	logger.WithComponent("whatsapp").Info().
		Str("dialect", dbDialect).
		Msg("WhatsApp service initialized successfully")

	service := &WhatsAppService{
		repository: repo,
		container:  container,
		clients:    make(map[string]*WhatsAppClient),
	}

	// Inicia auto-reconexão em background
	logger.WithComponent("whatsapp").Info().Msg("Scheduling auto-reconnection process...")
	go service.connectOnStartup()

	return service, nil
}

// connectOnStartup reconecta automaticamente sessões que estavam conectadas
func (s *WhatsAppService) connectOnStartup() {
	logger.WithComponent("whatsapp").Info().Msg("Auto-reconnection goroutine started")

	// Aguarda um pouco para garantir que o sistema está totalmente inicializado
	time.Sleep(2 * time.Second)

	logger.WithComponent("whatsapp").Info().Msg("Starting auto-reconnection process...")

	// Busca sessões que têm DeviceJID (foram conectadas anteriormente)
	sessions, err := s.repository.GetConnectedSessions()
	if err != nil {
		logger.WithComponent("whatsapp").Error().Err(err).Msg("Failed to get connected sessions for auto-reconnection")
		return
	}

	logger.WithComponent("whatsapp").Info().
		Int("sessions_found", len(sessions)).
		Msg("Sessions query completed")

	if len(sessions) == 0 {
		logger.WithComponent("whatsapp").Info().Msg("No sessions found for auto-reconnection")
		return
	}

	logger.WithComponent("whatsapp").Info().
		Int("session_count", len(sessions)).
		Msg("Found sessions for auto-reconnection")

	// Reconecta cada sessão em paralelo
	for _, session := range sessions {
		go func(sess *domain.Session) {
			// Aguarda um tempo aleatório para evitar sobrecarga
			time.Sleep(time.Duration(1+len(sess.ID)%5) * time.Second)

			logger.WithSession(sess.ID).Info().
				Str("session_name", sess.Session).
				Str("device_jid", sess.DeviceJID).
				Msg("Auto-reconnecting session")

			// Atualiza status para connecting
			s.UpdateSessionStatus(sess.ID, domain.StatusConnecting)

			// Inicia o cliente
			s.startClient(sess.ID, sess)
		}(session)
	}
}

// CreateSession cria uma nova sessão WhatsApp
func (s *WhatsAppService) CreateSession(req *domain.CreateSessionRequest) (*domain.Session, error) {
	// Gera um ID único para a sessão
	sessionID := uuid.New().String()

	// Cria a sessão no domínio
	session := &domain.Session{
		ID:        sessionID,
		Session:   req.Session,
		Status:    domain.StatusDisconnected,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Salva no repositório
	if err := s.repository.Create(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	logger.WithComponent("whatsapp").Info().Str("session_id", sessionID).Str("session_name", req.Session).Msg("Session created successfully")
	return session, nil
}

// GetSession obtém uma sessão por nome
func (s *WhatsAppService) GetSession(sessionName string) (*domain.Session, error) {
	return s.repository.GetBySession(sessionName)
}

// ListSessions lista todas as sessões
func (s *WhatsAppService) ListSessions() ([]*domain.Session, error) {
	return s.repository.List()
}

// DeleteSession remove uma sessão
func (s *WhatsAppService) DeleteSession(sessionName string) error {
	// Primeiro busca a sessão para obter o ID
	session, err := s.repository.GetBySession(sessionName)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session not found: %s", sessionName)
	}

	// Desconecta o cliente se estiver ativo
	s.clientsMutex.Lock()
	if client, exists := s.clients[session.ID]; exists {
		client.KillChannel <- true
		delete(s.clients, session.ID)
	}
	s.clientsMutex.Unlock()

	// Remove do repositório
	return s.repository.DeleteBySession(sessionName)
}

// UpdateSessionStatus atualiza o status de uma sessão
func (s *WhatsAppService) UpdateSessionStatus(id string, status domain.Status) error {
	session, err := s.repository.GetByID(id)
	if err != nil {
		return err
	}

	if session == nil {
		return fmt.Errorf("session not found: %s", id)
	}

	session.Status = status
	session.UpdatedAt = time.Now()

	return s.repository.Update(session)
}

// ConnectSession estabelece conexão com WhatsApp
func (s *WhatsAppService) ConnectSession(sessionName string) error {
	session, err := s.repository.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return fmt.Errorf("session not found: %s", sessionName)
	}

	// Verifica se já está conectado
	s.clientsMutex.RLock()
	if _, exists := s.clients[session.ID]; exists {
		s.clientsMutex.RUnlock()
		return errors.New("session already connected")
	}
	s.clientsMutex.RUnlock()

	// Atualiza status para connecting
	if err := s.UpdateSessionStatus(session.ID, domain.StatusConnecting); err != nil {
		return err
	}

	// Inicia o cliente em uma goroutine
	go s.startClient(session.ID, session)

	logger.WithComponent("whatsapp").Info().Str("session_name", sessionName).Str("session_id", session.ID).Msg("Starting WhatsApp connection")
	return nil
}

// DisconnectSession desconecta uma sessão
func (s *WhatsAppService) DisconnectSession(sessionName string) error {
	// Primeiro busca a sessão para obter o ID
	session, err := s.repository.GetBySession(sessionName)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session not found: %s", sessionName)
	}

	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	client, exists := s.clients[session.ID]
	if !exists {
		return errors.New("session not connected")
	}

	// Envia sinal de kill
	client.KillChannel <- true
	delete(s.clients, session.ID)

	// Atualiza status no banco
	if err := s.UpdateSessionStatus(session.ID, domain.StatusDisconnected); err != nil {
		logger.WithComponent("whatsapp").Error().Err(err).Str("session_name", sessionName).Str("session_id", session.ID).Msg("Failed to update session status")
	}

	logger.WithComponent("whatsapp").Info().Str("session_name", sessionName).Str("session_id", session.ID).Msg("Session disconnected")
	return nil
}

// GenerateQRCode gera QR code para uma sessão
func (s *WhatsAppService) GenerateQRCode(sessionName string) (string, error) {
	session, err := s.repository.GetBySession(sessionName)
	if err != nil {
		return "", err
	}
	if session == nil {
		return "", fmt.Errorf("session not found: %s", sessionName)
	}

	s.clientsMutex.RLock()
	_, exists := s.clients[session.ID]
	s.clientsMutex.RUnlock()

	if !exists {
		return "", errors.New("session not connected or QR not available")
	}

	return session.QRCode, nil
}

// PairPhone emparelha um telefone com a sessão
func (s *WhatsAppService) PairPhone(sessionName string, phone string) error {
	// TODO: Implementar pareamento por telefone
	return errors.New("phone pairing not implemented yet")
}

// GetSessionStatus obtém o status atual da sessão
func (s *WhatsAppService) GetSessionStatus(sessionName string) (*domain.StatusResponse, error) {
	session, err := s.repository.GetBySession(sessionName)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", sessionName)
	}

	return &domain.StatusResponse{
		Success:   true,
		SessionID: session.ID,
		Status:    session.Status,
		Message:   fmt.Sprintf("Session status: %s", session.Status),
	}, nil
}

// GetConnectedSessionsCount retorna o número de sessões conectadas atualmente
func (s *WhatsAppService) GetConnectedSessionsCount() int {
	s.clientsMutex.RLock()
	defer s.clientsMutex.RUnlock()
	return len(s.clients)
}

// SendTextMessage envia uma mensagem de texto
func (s *WhatsAppService) SendTextMessage(sessionName, to, message string) error {
	// Busca a sessão
	session, err := s.repository.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return fmt.Errorf("session not found: %s", sessionName)
	}

	// Verifica se o cliente está conectado
	s.clientsMutex.RLock()
	waClient, exists := s.clients[session.ID]
	s.clientsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("session not connected: %s", sessionName)
	}

	// Parse do JID de destino
	recipient, ok := s.parseJID(to)
	if !ok {
		return fmt.Errorf("invalid recipient JID: %s", to)
	}

	// Gera ID da mensagem
	msgID := waClient.Client.GenerateMessageID()

	// Cria a mensagem usando ExtendedTextMessage (como no wuzapi)
	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: &message,
		},
	}

	// Envia a mensagem
	resp, err := waClient.Client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: msgID})
	if err != nil {
		logger.WithSession(session.ID).Error().
			Err(err).
			Str("recipient", to).
			Str("message_id", msgID).
			Msg("Failed to send message")
		return fmt.Errorf("failed to send message: %w", err)
	}

	logger.WithSession(session.ID).Info().
		Str("recipient", to).
		Str("message_id", msgID).
		Str("timestamp", fmt.Sprintf("%v", resp.Timestamp)).
		Str("message_preview", message[:min(50, len(message))]).
		Msg("Message sent successfully")

	return nil
}

// startClient inicia o cliente WhatsApp para uma sessão
func (s *WhatsAppService) startClient(sessionID string, session *domain.Session) {
	logger.WithSession(sessionID).Info().Msg("Starting WhatsApp client")

	// Cria ou obtém device store
	var deviceStore *store.Device
	var err error

	if session.DeviceJID != "" {
		// Tenta recuperar device existente
		jid, ok := s.parseJID(session.DeviceJID)
		if ok {
			deviceStore, err = s.container.GetDevice(context.Background(), jid)
			if err != nil {
				logger.WithSession(sessionID).Error().Err(err).Msg("Failed to get existing device")
				deviceStore = s.container.NewDevice()
			}
		} else {
			deviceStore = s.container.NewDevice()
		}
	} else {
		deviceStore = s.container.NewDevice()
	}

	// Cria cliente WhatsApp
	clientLog := waLog.Noop
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// Cria estrutura do cliente
	waClient := &WhatsAppClient{
		Client:      client,
		SessionID:   sessionID,
		Status:      domain.StatusConnecting,
		KillChannel: make(chan bool),
		Repository:  s.repository,
	}

	// Adiciona event handler
	waClient.EventHandler = client.AddEventHandler(s.createEventHandler(waClient))

	// Adiciona cliente ao mapa
	s.clientsMutex.Lock()
	s.clients[sessionID] = waClient
	s.clientsMutex.Unlock()

	// Verifica se precisa fazer login
	if client.Store.ID == nil {
		// Novo login - gera QR code
		s.handleNewLogin(waClient)
	} else {
		// Já logado - apenas conecta
		s.handleExistingLogin(waClient)
	}

	// Loop principal para manter cliente vivo
	s.keepClientAlive(waClient)
}

// parseJID converte string para types.JID
func (s *WhatsAppService) parseJID(arg string) (types.JID, bool) {
	if len(arg) > 0 && arg[0] == '+' {
		arg = arg[1:]
	}

	if !strings.ContainsRune(arg, '@') {
		return types.NewJID(arg, types.DefaultUserServer), true
	}

	recipient, err := types.ParseJID(arg)
	if err != nil {
		logger.WithComponent("whatsapp").Error().Err(err).Msg("Invalid JID")
		return recipient, false
	}

	if recipient.User == "" {
		logger.WithComponent("whatsapp").Error().Msg("Invalid JID no server specified")
		return recipient, false
	}

	return recipient, true
}

// handleNewLogin gerencia novo login com QR code
func (s *WhatsAppService) handleNewLogin(waClient *WhatsAppClient) {
	qrChan, err := waClient.Client.GetQRChannel(context.Background())
	if err != nil {
		if !errors.Is(err, whatsmeow.ErrQRStoreContainsID) {
			logger.WithSession(waClient.SessionID).Error().Err(err).Msg("Failed to get QR channel")
			return
		}
	}

	// Conecta o cliente primeiro
	err = waClient.Client.Connect()
	if err != nil {
		logger.WithSession(waClient.SessionID).Error().Err(err).Msg("Failed to connect client")
		return
	}

	waClient.QRChannel = qrChan

	// Processa eventos do QR
	go s.processQREvents(waClient, qrChan)

	// Inicia o loop de manter vivo
	go s.keepClientAlive(waClient)
}

// handleExistingLogin gerencia login existente
func (s *WhatsAppService) handleExistingLogin(waClient *WhatsAppClient) {
	logger.WithSession(waClient.SessionID).Info().Msg("Already logged in, connecting...")

	err := waClient.Client.Connect()
	if err != nil {
		logger.WithSession(waClient.SessionID).Error().Err(err).Msg("Failed to connect existing client")
		s.UpdateSessionStatus(waClient.SessionID, domain.StatusDisconnected)

		// Remove cliente do mapa se falhou
		s.clientsMutex.Lock()
		delete(s.clients, waClient.SessionID)
		s.clientsMutex.Unlock()
		return
	}

	// Inicia o loop de manter vivo
	go s.keepClientAlive(waClient)
}

// processQREvents processa eventos do QR code
func (s *WhatsAppService) processQREvents(waClient *WhatsAppClient, qrChan <-chan whatsmeow.QRChannelItem) {
	for evt := range qrChan {
		switch evt.Event {
		case "code":
			// Imprime QR code no terminal (útil para desenvolvimento)
			logger.WithSession(waClient.SessionID).Info().Msg("QR Code gerado - escaneie com seu WhatsApp:")
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			fmt.Printf("\nQR Code String: %s\n\n", evt.Code)

			// Gera QR code em base64 para API
			image, err := qrcode.Encode(evt.Code, qrcode.Medium, 256)
			if err != nil {
				logger.WithSession(waClient.SessionID).Error().Err(err).Msg("Failed to generate QR code")
				continue
			}

			base64QR := "data:image/png;base64," + base64.StdEncoding.EncodeToString(image)

			// Atualiza sessão com QR code
			session, err := s.repository.GetByID(waClient.SessionID)
			if err != nil {
				logger.WithSession(waClient.SessionID).Error().Err(err).Msg("Failed to get session")
				continue
			}

			session.QRCode = base64QR
			session.UpdatedAt = time.Now()

			if err := s.repository.Update(session); err != nil {
				logger.WithSession(waClient.SessionID).Error().Err(err).Msg("Failed to update session with QR")
			} else {
				logger.WithSession(waClient.SessionID).Info().Msg("QR code generated and saved")
			}

		case "timeout":
			logger.WithSession(waClient.SessionID).Warn().Msg("QR code timeout")
			s.UpdateSessionStatus(waClient.SessionID, domain.StatusDisconnected)

			// Limpa QR code
			session, err := s.repository.GetByID(waClient.SessionID)
			if err == nil {
				session.QRCode = ""
				session.UpdatedAt = time.Now()
				s.repository.Update(session)
			}

			// Mata o cliente
			waClient.KillChannel <- true

		case "success":
			logger.WithSession(waClient.SessionID).Info().Msg("QR pairing successful")
			s.UpdateSessionStatus(waClient.SessionID, domain.StatusConnected)

			// Limpa QR code após sucesso
			session, err := s.repository.GetByID(waClient.SessionID)
			if err == nil {
				session.QRCode = ""
				session.UpdatedAt = time.Now()
				s.repository.Update(session)
			}

		default:
			logger.WithSession(waClient.SessionID).Info().Str("event", evt.Event).Msg("QR event received")
		}
	}
}

// createEventHandler cria o handler de eventos para o cliente WhatsApp
func (s *WhatsAppService) createEventHandler(waClient *WhatsAppClient) func(interface{}) {
	return func(rawEvt interface{}) {
		switch evt := rawEvt.(type) {
		case *events.Connected:
			logger.WithSession(waClient.SessionID).Info().Msg("WhatsApp connected")
			s.UpdateSessionStatus(waClient.SessionID, domain.StatusConnected)

			// Envia presença disponível
			if err := waClient.Client.SendPresence(types.PresenceAvailable); err != nil {
				logger.WithSession(waClient.SessionID).Warn().Err(err).Msg("Failed to send available presence")
			}

		case *events.PairSuccess:
			logger.WithSession(waClient.SessionID).Info().
				Str("jid", evt.ID.String()).
				Str("business_name", evt.BusinessName).
				Str("platform", evt.Platform).
				Msg("QR Pair Success")

			// Atualiza sessão com JID
			session, err := s.repository.GetByID(waClient.SessionID)
			if err == nil {
				session.DeviceJID = evt.ID.String()
				session.Status = domain.StatusConnected
				session.UpdatedAt = time.Now()
				s.repository.Update(session)
			}

		case *events.LoggedOut:
			logger.WithSession(waClient.SessionID).Info().
				Str("reason", evt.Reason.String()).
				Msg("Logged out from WhatsApp")

			s.UpdateSessionStatus(waClient.SessionID, domain.StatusDisconnected)
			waClient.KillChannel <- true

		case *events.StreamReplaced:
			logger.WithSession(waClient.SessionID).Info().Msg("Stream replaced")

		default:
			logger.WithSession(waClient.SessionID).Debug().
				Str("event_type", fmt.Sprintf("%T", evt)).
				Msg("Unhandled WhatsApp event")
		}
	}
}

// keepClientAlive mantém o cliente vivo até receber sinal de kill
func (s *WhatsAppService) keepClientAlive(waClient *WhatsAppClient) {
	defer func() {
		// Cleanup quando sair do loop
		waClient.Client.Disconnect()

		s.clientsMutex.Lock()
		delete(s.clients, waClient.SessionID)
		s.clientsMutex.Unlock()

		s.UpdateSessionStatus(waClient.SessionID, domain.StatusDisconnected)

		logger.WithSession(waClient.SessionID).Info().Msg("Client cleanup completed")
	}()

	for {
		select {
		case <-waClient.KillChannel:
			logger.WithSession(waClient.SessionID).Info().Msg("Received kill signal")
			return
		default:
			time.Sleep(1 * time.Second)
		}
	}
}
