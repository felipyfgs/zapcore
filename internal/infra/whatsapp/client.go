package whatsapp

import (
	"context"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"

	"wamex/pkg/logger"
)

// Client representa um cliente WhatsApp
type Client struct {
	*whatsmeow.Client
	SessionName string
	DeviceJID   types.JID
	Connected   bool
}

// ClientManager gerencia múltiplos clientes WhatsApp
type ClientManager struct {
	clients map[string]*Client
	store   *sqlstore.Container
}

// NewClientManager cria um novo gerenciador de clientes
func NewClientManager(dbDialect, dsn string) (*ClientManager, error) {
	// Configurar store do WhatsApp
	dbLog := waLog.Stdout("Database", "INFO", true)
	container, err := sqlstore.New(context.Background(), dbDialect, dsn, dbLog)
	if err != nil {
		return nil, fmt.Errorf("failed to create WhatsApp store: %w", err)
	}

	return &ClientManager{
		clients: make(map[string]*Client),
		store:   container,
	}, nil
}

// CreateClient cria um novo cliente WhatsApp
func (cm *ClientManager) CreateClient(sessionName string) (*Client, error) {
	// Verificar se cliente já existe
	if client, exists := cm.clients[sessionName]; exists {
		return client, nil
	}

	// Criar device store
	deviceStore, err := cm.store.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get device store: %w", err)
	}

	// Criar cliente WhatsApp
	clientLog := waLog.Stdout("Client", "INFO", true)
	whatsappClient := whatsmeow.NewClient(deviceStore, clientLog)

	client := &Client{
		Client:      whatsappClient,
		SessionName: sessionName,
		Connected:   false,
	}

	// Adicionar ao mapa de clientes
	cm.clients[sessionName] = client

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Msg("WhatsApp client created")

	return client, nil
}

// GetClient obtém um cliente existente
func (cm *ClientManager) GetClient(sessionName string) (*Client, bool) {
	client, exists := cm.clients[sessionName]
	return client, exists
}

// ConnectClient conecta um cliente
func (cm *ClientManager) ConnectClient(sessionName string) error {
	client, exists := cm.clients[sessionName]
	if !exists {
		return fmt.Errorf("client %s not found", sessionName)
	}

	if client.Connected {
		return fmt.Errorf("client %s is already connected", sessionName)
	}

	// Conectar cliente
	err := client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect client %s: %w", sessionName, err)
	}

	client.Connected = true
	client.DeviceJID = *client.Store.ID

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Str("device_jid", client.DeviceJID.String()).
		Msg("WhatsApp client connected")

	return nil
}

// DisconnectClient desconecta um cliente
func (cm *ClientManager) DisconnectClient(sessionName string) error {
	client, exists := cm.clients[sessionName]
	if !exists {
		return fmt.Errorf("client %s not found", sessionName)
	}

	if !client.Connected {
		return fmt.Errorf("client %s is not connected", sessionName)
	}

	// Desconectar cliente
	client.Disconnect()
	client.Connected = false

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Msg("WhatsApp client disconnected")

	return nil
}

// DeleteClient remove um cliente
func (cm *ClientManager) DeleteClient(sessionName string) error {
	client, exists := cm.clients[sessionName]
	if !exists {
		return fmt.Errorf("client %s not found", sessionName)
	}

	// Desconectar se estiver conectado
	if client.Connected {
		client.Disconnect()
	}

	// Remover do mapa
	delete(cm.clients, sessionName)

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Msg("WhatsApp client deleted")

	return nil
}

// GetConnectedClients retorna o número de clientes conectados
func (cm *ClientManager) GetConnectedClients() int {
	count := 0
	for _, client := range cm.clients {
		if client.Connected {
			count++
		}
	}
	return count
}

// IsClientConnected verifica se um cliente está conectado
func (cm *ClientManager) IsClientConnected(sessionName string) bool {
	client, exists := cm.clients[sessionName]
	if !exists {
		return false
	}
	return client.Connected
}

// GenerateQRCode gera QR code para pareamento
func (cm *ClientManager) GenerateQRCode(sessionName string) (string, error) {
	client, exists := cm.clients[sessionName]
	if !exists {
		return "", fmt.Errorf("client %s not found", sessionName)
	}

	if client.Store.ID != nil {
		return "", fmt.Errorf("client %s is already logged in", sessionName)
	}

	// Gerar QR code
	qrChan, err := client.GetQRChannel(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get QR channel: %w", err)
	}

	err = client.Connect()
	if err != nil {
		return "", fmt.Errorf("failed to connect for QR: %w", err)
	}

	// Aguardar QR code
	select {
	case evt := <-qrChan:
		if evt.Event == "code" {
			return evt.Code, nil
		}
	case <-time.After(30 * time.Second):
		return "", fmt.Errorf("timeout waiting for QR code")
	}

	return "", fmt.Errorf("failed to generate QR code")
}

// PairPhone faz pareamento via código de telefone
func (cm *ClientManager) PairPhone(sessionName, phone string) error {
	client, exists := cm.clients[sessionName]
	if !exists {
		return fmt.Errorf("client %s not found", sessionName)
	}

	if client.Store.ID != nil {
		return fmt.Errorf("client %s is already logged in", sessionName)
	}

	// Conectar para pareamento
	err := client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect for pairing: %w", err)
	}

	// Solicitar código de pareamento
	code, err := client.PairPhone(context.Background(), phone, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		return fmt.Errorf("failed to pair phone: %w", err)
	}

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Str("phone", phone).
		Str("code", code).
		Msg("Phone pairing code generated")

	return nil
}
