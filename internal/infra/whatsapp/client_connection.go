package whatsapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"sync"
	"time"

	"zapcore/internal/domain/session"
	"zapcore/internal/domain/whatsapp"

	"github.com/google/uuid"
	"github.com/mdp/qrterminal/v3"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/store"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// ConnectionManager gerencia conexões WhatsApp
type ConnectionManager struct {
	client *WhatsAppClient
}

// NewConnectionManager cria novo gerenciador de conexões
func NewConnectionManager(client *WhatsAppClient) *ConnectionManager {
	return &ConnectionManager{client: client}
}

// Connect estabelece conexão com o WhatsApp
func (cm *ConnectionManager) Connect(ctx context.Context, sessionID uuid.UUID) error {
	cm.client.logger.Info().Str("session_id", sessionID.String()).Msg("Iniciando conexão com WhatsApp")

	// Atualizar status para "connecting"
	if err := cm.client.sessionRepo.UpdateStatus(ctx, sessionID, session.WhatsAppStatusConnecting); err != nil {
		cm.client.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao atualizar status para connecting")
		return fmt.Errorf("erro ao atualizar status da sessão %s: %w", sessionID.String(), err)
	}

	// Buscar dados da sessão
	sessions, err := cm.client.sessionRepo.GetActiveSessions(ctx)
	if err != nil {
		return fmt.Errorf("erro ao buscar sessões para sessão %s: %w", sessionID.String(), err)
	}

	var sessionData *session.Session
	for _, s := range sessions {
		if s.ID == sessionID {
			sessionData = s
			break
		}
	}

	// Criar canal de kill para controlar a conexão
	cm.client.killMutex.Lock()
	cm.client.killChannels[sessionID] = make(chan bool)
	cm.client.killMutex.Unlock()

	// Decidir como conectar baseado no estado da sessão
	if sessionData != nil && sessionData.JID != "" {
		// Sessão já autenticada, usar reconnectSession
		cm.client.logger.Info().
			Str("session_id", sessionID.String()).
			Str("jid", sessionData.JID).
			Msg("Sessão já autenticada, reconectando")

		go func() {
			if err := cm.reconnectSession(ctx, sessionData); err != nil {
				cm.client.logger.Error().Err(err).Msg("Erro na reconexão")
			}
		}()
	} else {
		// Nova sessão, usar startClient para QR code
		cm.client.logger.Info().
			Str("session_id", sessionID.String()).
			Msg("Nova sessão, iniciando processo de autenticação")

		go cm.startClient(sessionID)
	}

	return nil
}

// ConnectOnStartup reconecta automaticamente sessões ativas com JID
func (cm *ConnectionManager) ConnectOnStartup(ctx context.Context) error {
	cm.client.logger.WithFields(map[string]interface{}{
		"component": "whatsapp",
		"operation": "reconnection",
		"phase":     "checking_sessions",
	}).Info().Msg("🔄 Verificando sessões ativas")

	// Buscar sessões ativas com JID no banco
	sessions, err := cm.client.sessionRepo.GetActiveSessions(ctx)
	if err != nil {
		cm.client.logger.Error().Err(err).Msg("Erro ao buscar sessões ativas para reconexão")
		return fmt.Errorf("erro ao buscar sessões ativas: %w", err)
	}

	if len(sessions) == 0 {
		return nil
	}

	cm.client.logger.Info().Int("count", len(sessions)).Msg("Sessões ativas encontradas para reconexão")

	// Reconectar cada sessão em paralelo
	var wg sync.WaitGroup
	reconnectedCount := 0
	failedCount := 0
	var mu sync.Mutex

	for _, sess := range sessions {
		wg.Add(1)
		go func(s *session.Session) {
			defer wg.Done()

			if err := cm.reconnectSession(ctx, s); err != nil {
				cm.client.logger.Error().
					Err(err).
					Str("session_id", s.ID.String()).
					Str("session_name", s.Name).
					Str("jid", s.JID).
					Msg("Falha ao reconectar sessão")

				mu.Lock()
				failedCount++
				mu.Unlock()
			} else {
				cm.client.logger.Info().
					Str("session_id", s.ID.String()).
					Str("session_name", s.Name).
					Str("jid", s.JID).
					Msg("Sessão reconectada com sucesso")

				mu.Lock()
				reconnectedCount++
				mu.Unlock()
			}
		}(sess)
	}

	// Aguardar todas as reconexões
	wg.Wait()

	cm.client.logger.Info().
		Int("reconnected", reconnectedCount).
		Int("failed", failedCount).
		Int("total", len(sessions)).
		Msg("Processo de reconexão automática finalizado")

	return nil
}

// configureDeviceProps configura propriedades do device (elimina duplicação)
func (cm *ConnectionManager) configureDeviceProps() {
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_UNKNOWN.Enum()
	store.DeviceProps.Os = proto.String("ZapCore")
}

// createWhatsmeowClient cria cliente whatsmeow (elimina duplicação)
func (cm *ConnectionManager) createWhatsmeowClient(deviceStore *store.Device, sessionID uuid.UUID) *whatsmeow.Client {
	// Configurar propriedades do device
	cm.configureDeviceProps()

	// Criar logger para o cliente
	clientLog := waLog.Stdout("Client", "INFO", true)

	// Criar cliente WhatsApp
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// Adicionar event handler
	client.AddEventHandler(func(evt any) {
		cm.client.handleWhatsAppEvent(sessionID, evt)
	})

	// Configurar MediaDownloader se MinIO estiver habilitado
	if cm.client.minioClient != nil {
		cm.client.configureMediaDownloader(sessionID, client)
	}

	return client
}

// GetStatus retorna o status da conexão
func (cm *ConnectionManager) GetStatus(ctx context.Context, sessionID uuid.UUID) (whatsapp.ConnectionStatus, error) {
	cm.client.clientsMutex.RLock()
	client, exists := cm.client.clients[sessionID]
	cm.client.clientsMutex.RUnlock()

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

// IsConnected verifica se está conectado
func (cm *ConnectionManager) IsConnected(sessionID uuid.UUID) bool {
	cm.client.clientsMutex.RLock()
	client, exists := cm.client.clients[sessionID]
	cm.client.clientsMutex.RUnlock()

	return exists && client.IsConnected()
}

// IsLoggedIn verifica se está logado
func (cm *ConnectionManager) IsLoggedIn(sessionID uuid.UUID) bool {
	cm.client.clientsMutex.RLock()
	client, exists := cm.client.clients[sessionID]
	cm.client.clientsMutex.RUnlock()

	return exists && client.IsLoggedIn()
}

// GetQRCode obtém QR Code
func (cm *ConnectionManager) GetQRCode(ctx context.Context, sessionID uuid.UUID) (string, error) {
	cm.client.clientsMutex.RLock()
	client, exists := cm.client.clients[sessionID]
	cm.client.clientsMutex.RUnlock()

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
func (cm *ConnectionManager) PairPhone(ctx context.Context, sessionID uuid.UUID, phoneNumber string, showPushNotification bool) error {
	cm.client.clientsMutex.RLock()
	client, exists := cm.client.clients[sessionID]
	cm.client.clientsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("cliente não encontrado para sessão %s", sessionID.String())
	}

	// Implementar pareamento por telefone usando whatsmeow
	code, err := client.PairPhone(ctx, phoneNumber, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		return fmt.Errorf("erro ao emparelhar com telefone: %w", err)
	}

	cm.client.logger.Info().Str("phone", phoneNumber).Str("code", code).Msg("Código de pareamento gerado")
	return nil
}

// reconnectSession reconecta uma sessão específica
func (cm *ConnectionManager) reconnectSession(ctx context.Context, sessionData *session.Session) error {
	cm.client.logger.Info().
		Str("session_id", sessionData.ID.String()).
		Str("jid", sessionData.JID).
		Msg("Iniciando reconexão de sessão autenticada")

	// Obter device store existente para a sessão usando o JID
	deviceStore, err := cm.client.container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("erro ao obter device store para sessão %s: %w", sessionData.ID.String(), err)
	}
	if deviceStore == nil {
		return fmt.Errorf("device store não encontrado para sessão %s", sessionData.ID.String())
	}

	// Criar cliente WhatsApp
	client := cm.createWhatsmeowClient(deviceStore, sessionData.ID)

	// Conectar (sem QR code, pois já está autenticado)
	err = client.Connect()
	if err != nil {
		return fmt.Errorf("erro ao conectar cliente para sessão %s: %w", sessionData.ID.String(), err)
	}

	// Aguardar um momento para garantir que a conexão seja estabelecida
	time.Sleep(100 * time.Millisecond)

	// Verificar se o cliente está realmente conectado antes de armazenar
	if !client.IsConnected() {
		return fmt.Errorf("cliente não conseguiu se conectar para sessão %s", sessionData.ID.String())
	}

	// Armazenar cliente com verificação de integridade
	cm.client.clientsMutex.Lock()
	cm.client.clients[sessionData.ID] = client

	// Verificar se o cliente foi realmente armazenado
	if storedClient, exists := cm.client.clients[sessionData.ID]; exists && storedClient == client {
		cm.client.logger.Info().
			Str("session_id", sessionData.ID.String()).
			Int("total_clients_after", len(cm.client.clients)).
			Bool("is_connected", client.IsConnected()).
			Msg("✅ Cliente armazenado com sucesso no mapa durante reconexão")
	} else {
		cm.client.logger.Error().
			Str("session_id", sessionData.ID.String()).
			Msg("❌ Falha ao armazenar cliente no mapa")
	}
	cm.client.clientsMutex.Unlock()

	// Atualizar status da sessão para "connected"
	if err := cm.client.sessionRepo.UpdateStatus(ctx, sessionData.ID, session.WhatsAppStatusConnected); err != nil {
		cm.client.logger.Error().Err(err).Str("session_id", sessionData.ID.String()).Msg("Erro ao atualizar status para connected")
	}

	// Garantir que o canal de kill existe antes de iniciar keep-alive
	cm.client.killMutex.Lock()
	if _, exists := cm.client.killChannels[sessionData.ID]; !exists {
		cm.client.killChannels[sessionData.ID] = make(chan bool)
		cm.client.logger.Debug().Str("session_id", sessionData.ID.String()).Msg("Canal de kill criado para reconexão")
	}
	cm.client.killMutex.Unlock()

	// Iniciar loop para manter cliente vivo
	go cm.keepClientAlive(sessionData.ID)

	return nil
}

// startClient inicia o cliente WhatsApp para nova sessão
func (cm *ConnectionManager) startClient(sessionID uuid.UUID) {
	cm.client.logger.Info().Str("session_id", sessionID.String()).Msg("Iniciando cliente WhatsApp")

	// Criar novo device store
	deviceStore := cm.client.container.NewDevice()

	// Criar cliente WhatsApp
	client := cm.createWhatsmeowClient(deviceStore, sessionID)

	// Verificar se precisa de autenticação
	if client.Store.ID == nil {
		// Não está autenticado, precisa de QR code
		cm.client.logger.Info().Str("session_id", sessionID.String()).Msg("Cliente não autenticado, gerando QR code")

		qrChan, err := client.GetQRChannel(context.Background())
		if err != nil {
			cm.client.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao obter canal QR")
			return
		}

		// Conectar primeiro para poder gerar QR
		err = client.Connect()
		if err != nil {
			cm.client.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao conectar cliente")
			return
		}

		// Armazenar cliente
		cm.client.clientsMutex.Lock()
		cm.client.clients[sessionID] = client
		cm.client.clientsMutex.Unlock()

		// Processar eventos QR
		cm.processQREvents(sessionID, qrChan)
	} else {
		// Já está autenticado, apenas conectar
		cm.client.logger.Info().Str("session_id", sessionID.String()).Msg("Cliente já autenticado, conectando")

		err := client.Connect()
		if err != nil {
			cm.client.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao conectar cliente")
			return
		}

		// Armazenar cliente
		cm.client.clientsMutex.Lock()
		cm.client.clients[sessionID] = client
		cm.client.clientsMutex.Unlock()
	}

	// Loop para manter cliente vivo
	cm.keepClientAlive(sessionID)
}

// processQREvents processa eventos do canal QR
func (cm *ConnectionManager) processQREvents(sessionID uuid.UUID, qrChan <-chan whatsmeow.QRChannelItem) {
	cm.client.logger.Info().Str("session_id", sessionID.String()).Msg("Iniciando processamento de eventos QR")

	for evt := range qrChan {
		switch evt.Event {
		case "code":
			cm.handleQRCode(sessionID, evt.Code)
		case "timeout":
			cm.handleQRTimeout(sessionID)
			return
		case "success":
			cm.handleQRSuccess(sessionID)
			return
		default:
			cm.client.logger.Info().Str("session_id", sessionID.String()).Str("event", evt.Event).Msg("Evento QR recebido")
		}
	}
}

// handleQRCode processa evento de código QR
func (cm *ConnectionManager) handleQRCode(sessionID uuid.UUID, code string) {
	cm.client.logger.Info().Str("session_id", sessionID.String()).Msg("QR Code gerado")

	// Exibir QR code no terminal
	cm.client.logger.Info().
		Str("session_id", sessionID.String()).
		Str("qr_code", code).
		Msg("=== QR CODE GERADO ===")

	qrterminal.GenerateHalfBlock(code, qrterminal.L, os.Stdout)

	cm.client.logger.Info().
		Str("session_id", sessionID.String()).
		Msg("=== Escaneie o código acima com seu WhatsApp ===")

	// Gerar QR code em base64 para armazenar
	image, err := qrcode.Encode(code, qrcode.Medium, 256)
	if err != nil {
		cm.client.logger.Error().Err(err).Msg("Erro ao gerar QR code em base64")
	} else {
		base64QR := "data:image/png;base64," + base64.StdEncoding.EncodeToString(image)
		cm.client.logger.Info().Str("session_id", sessionID.String()).Str("qr_base64", base64QR).Msg("QR Code base64 gerado")
	}
}

// handleQRTimeout processa timeout do QR code
func (cm *ConnectionManager) handleQRTimeout(sessionID uuid.UUID) {
	cm.client.logger.Warn().
		Str("session_id", sessionID.String()).
		Msg("⏰ QR Code expirou - Tente conectar novamente")

	// Limpar cliente
	cm.client.clientsMutex.Lock()
	if client, exists := cm.client.clients[sessionID]; exists {
		client.Disconnect()
		delete(cm.client.clients, sessionID)
	}
	cm.client.clientsMutex.Unlock()

	// Atualizar status no banco para disconnected para permitir nova conexão
	ctx := context.Background()
	if err := cm.client.sessionRepo.UpdateStatus(ctx, sessionID, session.WhatsAppStatusDisconnected); err != nil {
		cm.client.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao atualizar status para disconnected após timeout do QR")
	} else {
		cm.client.logger.Info().Str("session_id", sessionID.String()).Msg("Status atualizado para disconnected após timeout do QR")
	}

	// Enviar sinal de kill
	cm.sendKillSignal(sessionID)
}

// handleQRSuccess processa sucesso do QR code
func (cm *ConnectionManager) handleQRSuccess(sessionID uuid.UUID) {
	cm.client.logger.Info().
		Str("session_id", sessionID.String()).
		Msg("✅ QR Code escaneado com sucesso")
}

// sendKillSignal envia sinal de kill para sessão
func (cm *ConnectionManager) sendKillSignal(sessionID uuid.UUID) {
	cm.client.killMutex.RLock()
	if killChan, exists := cm.client.killChannels[sessionID]; exists {
		select {
		case killChan <- true:
		default:
		}
	}
	cm.client.killMutex.RUnlock()
}

// keepClientAlive mantém cliente vivo
func (cm *ConnectionManager) keepClientAlive(sessionID uuid.UUID) {
	cm.client.logger.Debug().Str("session_id", sessionID.String()).Msg("Iniciando loop keep-alive")

	// Verificação defensiva: garantir que o canal de kill existe
	cm.client.killMutex.Lock()
	killChan, exists := cm.client.killChannels[sessionID]
	if !exists {
		// Criar canal se não existir (recuperação defensiva)
		cm.client.killChannels[sessionID] = make(chan bool)
		killChan = cm.client.killChannels[sessionID]
		cm.client.logger.Warn().Str("session_id", sessionID.String()).Msg("Canal de kill não encontrado, criando defensivamente")
	}
	cm.client.killMutex.Unlock()

	for {
		select {
		case <-killChan:
			cm.client.logger.Info().Str("session_id", sessionID.String()).Msg("Sinal de kill recebido, encerrando keep-alive")
			return
		case <-time.After(30 * time.Second):
			// Verificar se cliente ainda está conectado
			cm.client.clientsMutex.RLock()
			client, exists := cm.client.clients[sessionID]
			cm.client.clientsMutex.RUnlock()

			if !exists {
				cm.client.logger.Warn().Str("session_id", sessionID.String()).Msg("Cliente não encontrado, encerrando keep-alive")
				return
			}

			if !client.IsConnected() {
				cm.client.logger.Warn().Str("session_id", sessionID.String()).Msg("Cliente desconectado, encerrando keep-alive")
				return
			}

			cm.client.logger.Debug().Str("session_id", sessionID.String()).Msg("Cliente ainda conectado")
		}
	}
}

// Disconnect encerra a conexão
func (cm *ConnectionManager) Disconnect(ctx context.Context, sessionID uuid.UUID) error {
	cm.client.logger.Info().Str("session_id", sessionID.String()).Msg("Desconectando cliente WhatsApp")

	// Enviar sinal de kill
	cm.sendKillSignal(sessionID)

	// Remover e desconectar cliente
	cm.client.clientsMutex.Lock()
	if client, exists := cm.client.clients[sessionID]; exists {
		client.Disconnect()
		delete(cm.client.clients, sessionID)
		cm.client.logger.Info().Str("session_id", sessionID.String()).Msg("Cliente removido e desconectado")
	}
	cm.client.clientsMutex.Unlock()

	// Limpar canal de kill
	cm.client.killMutex.Lock()
	if killChan, exists := cm.client.killChannels[sessionID]; exists {
		close(killChan)
		delete(cm.client.killChannels, sessionID)
	}
	cm.client.killMutex.Unlock()

	// Atualizar status no banco
	if err := cm.client.sessionRepo.UpdateStatus(ctx, sessionID, session.WhatsAppStatusDisconnected); err != nil {
		cm.client.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao atualizar status para disconnected")
	}

	return nil
}
