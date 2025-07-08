package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)

// SessionManager gerencia todas as sessões do WhatsApp
type SessionManager struct {
	sessions map[string]*WhatsAppSession
	mutex    sync.RWMutex
	store    *sqlstore.Container
	db       *DatabaseManager
	logger   waLog.Logger
}

// NewSessionManager cria um novo gerenciador de sessões
func NewSessionManager() *SessionManager {
	// Configurar logger
	logger := waLog.Stdout("SessionManager", "INFO", true)

	// Configurar store SQLite para whatsmeow
	dbLog := waLog.Stdout("Database", "INFO", true)

	// Primeiro, criar uma conexão para habilitar foreign keys
	db, err := sql.Open("sqlite", "file:whatsapp.db?cache=shared&mode=rwc")
	if err != nil {
		panic(fmt.Sprintf("Erro ao abrir banco de dados: %v", err))
	}

	// Habilitar foreign keys
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		panic(fmt.Sprintf("Erro ao habilitar foreign keys: %v", err))
	}
	db.Close()

	// Agora criar o store do whatsmeow
	container, err := sqlstore.New(context.Background(), "sqlite", "file:whatsapp.db?cache=shared&mode=rwc&_pragma=foreign_keys(1)", dbLog)
	if err != nil {
		panic(fmt.Sprintf("Erro ao criar store: %v", err))
	}

	// Configurar banco de dados para sessões usando Bun ORM (mesmo banco do WhatsApp)
	dbManager, err := NewDatabaseManager("file:whatsapp.db?cache=shared&mode=rwc&_foreign_keys=on")
	if err != nil {
		panic(fmt.Sprintf("Erro ao criar database manager: %v", err))
	}

	sm := &SessionManager{
		sessions: make(map[string]*WhatsAppSession),
		store:    container,
		db:       dbManager,
		logger:   logger,
	}

	// Carregar sessões existentes do banco de dados
	if err := sm.loadExistingSessions(); err != nil {
		logger.Warnf("Erro ao carregar sessões existentes: %v", err)
	}

	// Conectar automaticamente sessões que estavam conectadas
	go sm.connectOnStartup()

	return sm
}

// loadExistingSessions carrega sessões existentes do banco de dados
func (sm *SessionManager) loadExistingSessions() error {
	ctx := context.Background()
	sessionsDB, err := sm.db.GetAllSessions(ctx)
	if err != nil {
		return fmt.Errorf("erro ao buscar sessões no banco: %v", err)
	}

	for _, sessionDB := range sessionsDB {
		// Recriar apenas a estrutura básica da sessão
		session := sessionDB.ToWhatsAppSession()

		// Tentar recuperar device store existente ou criar novo
		var deviceStore *store.Device
		var err error

		if session.DeviceJID != nil && *session.DeviceJID != "" {
			// Tentar recuperar device existente usando o JID
			jid, parseErr := types.ParseJID(*session.DeviceJID)
			if parseErr == nil {
				deviceStore, err = sm.store.GetDevice(ctx, jid)
				if err != nil {
					sm.logger.Warnf("Erro ao recuperar device existente para sessão %s: %v", session.ID, err)
					deviceStore = sm.store.NewDevice()
				} else {
					sm.logger.Infof("Device existente recuperado para sessão %s (JID: %s)", session.ID, *session.DeviceJID)
				}
			} else {
				sm.logger.Warnf("JID inválido para sessão %s: %v", session.ID, parseErr)
				deviceStore = sm.store.NewDevice()
			}
		} else {
			// Criar novo device se não houver JID
			sm.logger.Infof("Criando novo device para sessão %s (sem JID)", session.ID)
			deviceStore = sm.store.NewDevice()
		}

		session.Device = deviceStore

		// Criar cliente WhatsApp
		client := whatsmeow.NewClient(deviceStore, sm.logger)
		session.Client = client

		// Criar canal de eventos
		session.EventChan = make(chan interface{}, 100)

		// Criar contexto cancelável
		ctx, cancel := context.WithCancel(context.Background())
		session.CancelFunc = cancel

		// Adicionar event handlers
		client.AddEventHandler(sm.createEventHandler(session))

		// Armazenar sessão em memória
		sm.sessions[session.ID] = session

		// Iniciar goroutine para processar eventos
		go sm.processEvents(ctx, session)

		sm.logger.Infof("Sessão carregada do banco: %s (nome: %s, status: %s)",
			session.ID, session.Name, session.Status)
	}

	return nil
}

// connectOnStartup conecta automaticamente sessões que estavam conectadas
func (sm *SessionManager) connectOnStartup() {
	ctx := context.Background()

	// Buscar sessões que estavam conectadas
	sessionsDB, err := sm.db.GetConnectedSessions(ctx)
	if err != nil {
		sm.logger.Errorf("Erro ao buscar sessões conectadas: %v", err)
		return
	}

	for _, sessionDB := range sessionsDB {
		sm.logger.Infof("Tentando reconectar sessão: %s (nome: %s)", sessionDB.ID, sessionDB.Name)

		// Buscar sessão em memória
		session, exists := sm.GetSessionByID(sessionDB.ID)
		if !exists {
			sm.logger.Warnf("Sessão %s não encontrada em memória", sessionDB.ID)
			continue
		}

		// Verificar se tem device JID
		if session.DeviceJID == nil || *session.DeviceJID == "" {
			sm.logger.Warnf("Sessão %s não tem device JID, pulando reconexão", sessionDB.ID)
			continue
		}

		// Tentar conectar em goroutine separada
		go func(s *WhatsAppSession) {
			sm.logger.Infof("Iniciando reconexão para sessão %s", s.ID)

			// Verificar se já está conectado
			if s.Client.IsConnected() {
				sm.logger.Infof("Sessão %s já está conectada", s.ID)
				return
			}

			// Tentar conectar
			err := s.Client.Connect()
			if err != nil {
				sm.logger.Errorf("Erro ao reconectar sessão %s: %v", s.ID, err)
				// Atualizar status no banco
				s.UpdateStatus(StatusDisconnected)
				now := time.Now()
				sm.db.UpdateSessionStatus(context.Background(), s.ID, StatusDisconnected, nil, &now)
			} else {
				sm.logger.Infof("Sessão %s reconectada com sucesso", s.ID)
				s.UpdateStatus(StatusConnected)
				now := time.Now()
				sm.db.UpdateSessionStatus(context.Background(), s.ID, StatusConnected, &now, &now)
			}
		}(session)
	}
}

// CreateNewSession cria uma nova sessão
func (sm *SessionManager) CreateNewSession(name string) (*WhatsAppSession, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Gerar ID único para a sessão
	sessionID := uuid.New().String()

	// Criar device store
	deviceStore := sm.store.NewDevice()

	// Criar cliente WhatsApp
	client := whatsmeow.NewClient(deviceStore, sm.logger)

	// Obter JID do device se estiver logado
	var deviceJID *string
	if client.IsLoggedIn() {
		jid := client.Store.ID.String()
		deviceJID = &jid
	}

	// Criar canal de eventos
	eventChan := make(chan interface{}, 100)

	// Criar contexto cancelável
	ctx, cancel := context.WithCancel(context.Background())

	// Criar sessão
	session := &WhatsAppSession{
		ID:         sessionID,
		Name:       name,
		Status:     StatusDisconnected,
		DeviceJID:  deviceJID,
		CreatedAt:  time.Now(),
		Client:     client,
		Device:     deviceStore,
		EventChan:  eventChan,
		CancelFunc: cancel,
	}

	// Adicionar event handlers
	client.AddEventHandler(sm.createEventHandler(session))

	// Armazenar sessão em memória
	sm.sessions[sessionID] = session

	// Salvar sessão no banco de dados
	if err := sm.db.SaveSession(context.Background(), session); err != nil {
		// Se falhar ao salvar no banco, remover da memória
		delete(sm.sessions, sessionID)
		return nil, fmt.Errorf("erro ao salvar sessão no banco: %v", err)
	}

	// Iniciar goroutine para processar eventos
	go sm.processEvents(ctx, session)

	sm.logger.Infof("Sessão criada: %s (nome: %s)", sessionID, name)
	return session, nil
}

// GetSessionByID retorna uma sessão pelo ID
func (sm *SessionManager) GetSessionByID(sessionID string) (*WhatsAppSession, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	return session, exists
}

// GetAllSessions retorna todas as sessões
func (sm *SessionManager) GetAllSessions() []*WhatsAppSession {
	// Buscar sessões do banco de dados para ter dados atualizados
	ctx := context.Background()
	sessionsDB, err := sm.db.GetAllSessions(ctx)
	if err != nil {
		sm.logger.Errorf("Erro ao buscar sessões do banco: %v", err)
		// Fallback para sessões em memória
		sm.mutex.RLock()
		defer sm.mutex.RUnlock()

		sessions := make([]*WhatsAppSession, 0, len(sm.sessions))
		for _, session := range sm.sessions {
			sessions = append(sessions, session.GetInfo())
		}
		return sessions
	}

	sessions := make([]*WhatsAppSession, 0, len(sessionsDB))
	for _, sessionDB := range sessionsDB {
		// Combinar dados do banco com dados em memória (se existir)
		sm.mutex.RLock()
		memorySession, exists := sm.sessions[sessionDB.ID]
		sm.mutex.RUnlock()

		if exists {
			// Atualizar dados da sessão em memória com dados do banco
			memorySession.Name = sessionDB.Name
			memorySession.Status = SessionStatus(sessionDB.Status)
			memorySession.CreatedAt = sessionDB.CreatedAt
			memorySession.ConnectedAt = sessionDB.ConnectedAt
			memorySession.LastSeen = sessionDB.LastSeen
			sessions = append(sessions, memorySession.GetInfo())
		} else {
			// Sessão existe no banco mas não na memória
			sessions = append(sessions, sessionDB.ToWhatsAppSession().GetInfo())
		}
	}

	return sessions
}

// RemoveSession remove uma sessão
func (sm *SessionManager) RemoveSession(sessionID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("sessão não encontrada: %s", sessionID)
	}

	// Desconectar se estiver conectado
	if session.Client.IsConnected() {
		session.Client.Disconnect()
	}

	// Cancelar contexto
	if session.CancelFunc != nil {
		session.CancelFunc()
	}

	// Fechar canal de eventos
	close(session.EventChan)

	// Remover da memória
	delete(sm.sessions, sessionID)

	// Remover sessão e device do banco de dados
	if err := sm.db.DeleteSessionAndDevice(context.Background(), sessionID); err != nil {
		sm.logger.Errorf("Erro ao remover sessão e device do banco: %v", err)
		// Não retornar erro pois a sessão já foi removida da memória
	}

	sm.logger.Infof("Sessão removida: %s", sessionID)
	return nil
}

// ConnectSessionToWhatsApp conecta uma sessão ao WhatsApp
func (sm *SessionManager) ConnectSessionToWhatsApp(sessionID string) error {
	session, exists := sm.GetSessionByID(sessionID)
	if !exists {
		return fmt.Errorf("sessão não encontrada: %s", sessionID)
	}

	if session.Client.IsConnected() {
		return fmt.Errorf("sessão já está conectada")
	}

	session.UpdateStatus(StatusConnecting)

	// Se não estiver logado, obter QR code
	if !session.Client.IsLoggedIn() {
		qrChan, err := session.Client.GetQRChannel(context.Background())
		if err != nil {
			session.UpdateStatus(StatusError)
			return fmt.Errorf("erro ao obter canal QR: %v", err)
		}
		session.QRChan = qrChan

		// Processar QR codes em goroutine
		go sm.processQRCodes(session)
	}

	// Conectar
	err := session.Client.Connect()
	if err != nil {
		session.UpdateStatus(StatusError)
		return fmt.Errorf("erro ao conectar: %v", err)
	}

	sm.logger.Infof("Sessão conectando: %s", sessionID)
	return nil
}

// DisconnectSessionFromWhatsApp desconecta uma sessão
func (sm *SessionManager) DisconnectSessionFromWhatsApp(sessionID string) error {
	session, exists := sm.GetSessionByID(sessionID)
	if !exists {
		return fmt.Errorf("sessão não encontrada: %s", sessionID)
	}

	if !session.Client.IsConnected() {
		return fmt.Errorf("sessão não está conectada")
	}

	session.Client.Disconnect()
	session.UpdateStatus(StatusDisconnected)

	sm.logger.Infof("Sessão desconectada: %s", sessionID)
	return nil
}

// GetSessionStatusByID retorna o status de uma sessão
func (sm *SessionManager) GetSessionStatusByID(sessionID string) (SessionStatus, error) {
	session, exists := sm.GetSessionByID(sessionID)
	if !exists {
		return "", fmt.Errorf("sessão não encontrada: %s", sessionID)
	}

	// Atualizar status baseado no cliente
	if session.Client.IsConnected() {
		session.UpdateStatus(StatusConnected)
	} else if session.Status == StatusConnected {
		session.UpdateStatus(StatusDisconnected)
	}

	return session.Status, nil
}

// GetQRCodeBySessionID retorna o QR code atual de uma sessão
func (sm *SessionManager) GetQRCodeBySessionID(sessionID string) (string, error) {
	session, exists := sm.GetSessionByID(sessionID)
	if !exists {
		return "", fmt.Errorf("sessão não encontrada: %s", sessionID)
	}

	if session.Client.IsLoggedIn() {
		return "", fmt.Errorf("sessão já está logada")
	}

	return session.QRCode, nil
}

// processEvents processa eventos de uma sessão
func (sm *SessionManager) processEvents(ctx context.Context, session *WhatsAppSession) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-session.EventChan:
			sm.handleSessionEvent(session, evt)
		}
	}
}

// processQRCodes processa QR codes de uma sessão
func (sm *SessionManager) processQRCodes(session *WhatsAppSession) {
	for evt := range session.QRChan {
		if evt.Event == "code" {
			session.QRCode = evt.Code
			sm.logger.Infof("QR Code gerado para sessão %s (%s)", session.ID, session.Name)

			// Exibir QR Code no terminal
			sm.displayQRCodeInTerminal(session.Name, evt.Code)
		}
	}
}

// displayQRCodeInTerminal exibe o QR Code no terminal
func (sm *SessionManager) displayQRCodeInTerminal(sessionName, qrData string) {
	separator := strings.Repeat("=", 80)

	fmt.Printf("\n%s\n", separator)
	fmt.Printf("🔗 QR CODE PARA SESSÃO: %s\n", sessionName)
	fmt.Printf("%s\n", separator)

	// Gerar QR Code para o terminal
	qr, err := qrcode.New(qrData, qrcode.Medium)
	if err != nil {
		sm.logger.Errorf("Erro ao gerar QR Code: %v", err)
		return
	}

	// Exibir QR Code no terminal
	fmt.Print(qr.ToSmallString(false))

	fmt.Printf("\n%s\n", separator)
	fmt.Printf("📱 Escaneie o QR Code acima com seu WhatsApp\n")
	fmt.Printf("🔄 Aguardando conexão...\n")
	fmt.Printf("%s\n\n", separator)
}

// createEventHandler cria um handler de eventos para uma sessão
func (sm *SessionManager) createEventHandler(session *WhatsAppSession) func(interface{}) {
	return func(evt interface{}) {
		select {
		case session.EventChan <- evt:
		default:
			// Canal cheio, descartar evento
		}
	}
}

// handleSessionEvent manipula eventos de uma sessão
func (sm *SessionManager) handleSessionEvent(session *WhatsAppSession, evt interface{}) {
	switch v := evt.(type) {
	case *events.Connected:
		session.UpdateStatus(StatusConnected)

		// Atualizar device_jid se estiver logado
		if session.Client.IsLoggedIn() {
			jid := session.Client.Store.ID.String()
			session.DeviceJID = &jid

			// Vincular sessão ao device no banco
			if err := sm.db.LinkSessionToDevice(context.Background(), session.ID, jid); err != nil {
				sm.logger.Errorf("Erro ao vincular sessão ao device: %v", err)
			}
		}

		sm.updateSessionInDB(session)
		sm.logger.Infof("Sessão conectada: %s", session.ID)

	case *events.Disconnected:
		session.UpdateStatus(StatusDisconnected)
		sm.updateSessionInDB(session)
		sm.logger.Infof("Sessão desconectada: %s", session.ID)

	case *events.LoggedOut:
		session.UpdateStatus(StatusLoggedOut)
		session.QRCode = ""
		session.DeviceJID = nil

		// Desvincular sessão do device
		if err := sm.db.UnlinkSessionFromDevice(context.Background(), session.ID); err != nil {
			sm.logger.Errorf("Erro ao desvincular sessão do device: %v", err)
		}

		sm.updateSessionInDB(session)
		sm.logger.Infof("Sessão deslogada: %s", session.ID)

	default:
		// Outros eventos podem ser processados aqui
		sm.logger.Debugf("Evento recebido na sessão %s: %T", session.ID, v)
	}
}

// updateSessionInDB atualiza uma sessão no banco de dados
func (sm *SessionManager) updateSessionInDB(session *WhatsAppSession) {
	ctx := context.Background()
	if err := sm.db.SaveSession(ctx, session); err != nil {
		sm.logger.Errorf("Erro ao atualizar sessão no banco: %v", err)
	}
}

// GetSessionByNameOrID retorna uma sessão pelo nome ou ID
func (sm *SessionManager) GetSessionByNameOrID(identifier string) (*WhatsAppSession, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// Tentar buscar por ID primeiro
	if session, exists := sm.sessions[identifier]; exists {
		return session, true
	}

	// Se não encontrar por ID, tentar por nome
	for _, session := range sm.sessions {
		if session.Name == identifier {
			return session, true
		}
	}

	return nil, false
}

// GetSessionByName retorna uma sessão pelo nome
func (sm *SessionManager) GetSessionByName(name string) (*WhatsAppSession, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// Buscar na memória primeiro
	for _, session := range sm.sessions {
		if session.Name == name {
			return session, true
		}
	}

	return nil, false
}

// SendTextMessage envia uma mensagem de texto
func (sm *SessionManager) SendTextMessage(sessionID, to, text string) error {
	session, exists := sm.GetSessionByID(sessionID)
	if !exists {
		return fmt.Errorf("sessão não encontrada: %s", sessionID)
	}

	if !session.Client.IsConnected() {
		return fmt.Errorf("sessão não está conectada")
	}

	// Criar JID do destinatário
	targetJID, err := types.ParseJID(to)
	if err != nil {
		return fmt.Errorf("JID inválido: %v", err)
	}

	// Criar mensagem
	message := &waE2E.Message{
		Conversation: proto.String(text),
	}

	// Enviar mensagem
	resp, err := session.Client.SendMessage(context.Background(), targetJID, message)
	if err != nil {
		return fmt.Errorf("erro ao enviar mensagem: %v", err)
	}

	sm.logger.Infof("Mensagem enviada para %s (timestamp: %s)", to, resp.Timestamp)
	return nil
}
