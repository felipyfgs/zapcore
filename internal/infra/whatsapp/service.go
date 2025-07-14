package whatsapp

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
	waCommon "go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	entity "wamex/internal/domain/entity"
	domainRepo "wamex/internal/domain/repository"
	"wamex/internal/usecase/media"
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
	Status       entity.Status
	QRChannel    <-chan whatsmeow.QRChannelItem
	KillChannel  chan bool
	Repository   domainRepo.SessionRepository
}

// WhatsAppService implementa a interface SessionService
type WhatsAppService struct {
	repository   domainRepo.SessionRepository
	container    *sqlstore.Container
	clients      map[string]*WhatsAppClient
	clientsMutex sync.RWMutex
}

// NewWhatsAppService cria uma nova instância do serviço WhatsApp
func NewWhatsAppService(repo domainRepo.SessionRepository, dbDialect string, dbSource string) (*WhatsAppService, error) {
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
		go func(sess *entity.Session) {
			// Aguarda um tempo aleatório para evitar sobrecarga
			time.Sleep(time.Duration(1+len(sess.ID)%5) * time.Second)

			logger.WithComponent("whatsapp").Info().
				Str("session_name", sess.Session).
				Str("device_jid", sess.DeviceJID).
				Msg("Auto-reconnecting session")

			// Atualiza status para connecting
			s.UpdateSessionStatus(sess.ID, entity.StatusConnecting)

			// Inicia o cliente
			s.startClient(sess.ID, sess)
		}(session)
	}
}

// CreateSession cria uma nova sessão WhatsApp
func (s *WhatsAppService) CreateSession(req *entity.CreateSessionRequest) (*entity.Session, error) {
	// Gera um ID único para a sessão
	sessionID := uuid.New().String()

	// Cria a sessão no domínio
	session := &entity.Session{
		ID:        sessionID,
		Session:   req.Session,
		Status:    entity.StatusDisconnected,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Salva no repositório
	if err := s.repository.Create(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	logger.WithComponent("whatsapp").Info().Str("session_name", req.Session).Msg("Session created successfully")
	return session, nil
}

// GetSession obtém uma sessão por nome
func (s *WhatsAppService) GetSession(sessionName string) (*entity.Session, error) {
	return s.repository.GetBySession(sessionName)
}

// GetSessionByID obtém uma sessão por ID ou nome
func (s *WhatsAppService) GetSessionByID(sessionID string) (*entity.Session, error) {
	// Primeiro tenta buscar por ID
	session, err := s.repository.GetByID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("error searching by ID: %w", err)
	}

	// Se não encontrou por ID, tenta buscar por nome da sessão
	if session == nil {
		session, err = s.repository.GetBySession(sessionID)
		if err != nil {
			return nil, fmt.Errorf("error searching by session name: %w", err)
		}

		// Se ainda não encontrou, retorna erro
		if session == nil {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
	}

	return session, nil
}

// ListSessions lista todas as sessões
func (s *WhatsAppService) ListSessions() ([]*entity.Session, error) {
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
func (s *WhatsAppService) UpdateSessionStatus(id string, status entity.Status) error {
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
	if err := s.UpdateSessionStatus(session.ID, entity.StatusConnecting); err != nil {
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
	if err := s.UpdateSessionStatus(session.ID, entity.StatusDisconnected); err != nil {
		logger.WithComponent("whatsapp").Error().Err(err).Str("session_name", sessionName).Msg("Failed to update session status")
	}

	logger.WithComponent("whatsapp").Info().Str("session_name", sessionName).Msg("Session disconnected")
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
func (s *WhatsAppService) GetSessionStatus(sessionName string) (*entity.StatusResponse, error) {
	session, err := s.repository.GetBySession(sessionName)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", sessionName)
	}

	return &entity.StatusResponse{
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

// SendImageMessage envia uma mensagem de imagem
func (s *WhatsAppService) SendImageMessage(sessionName, to, imageData, caption, mimeType string) error {
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

	// Processa a mídia
	mediaService := media.NewMediaService()
	processed, err := mediaService.ProcessMediaForUpload(imageData, entity.MessageTypeImage)
	if err != nil {
		return fmt.Errorf("failed to process image: %w", err)
	}

	// Faz upload para WhatsApp
	uploaded, err := mediaService.UploadMediaToWhatsApp(waClient.Client, processed.Data, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("failed to upload image: %w", err)
	}

	// Parse do JID de destino
	recipient, ok := s.parseJID(to)
	if !ok {
		return fmt.Errorf("invalid recipient JID: %s", to)
	}

	// Gera ID da mensagem
	msgID := waClient.Client.GenerateMessageID()

	// Cria a mensagem de imagem
	msg := &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			Caption:       &caption,
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			Mimetype:      &processed.MimeType,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(processed.Size)),
			JPEGThumbnail: processed.Thumbnail,
		},
	}

	// Envia a mensagem
	resp, err := waClient.Client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: msgID})
	if err != nil {
		logger.WithComponent("whatsapp").Error().
			Err(err).
			Str("session_name", sessionName).
			Str("recipient", to).
			Str("message_id", msgID).
			Msg("Failed to send image message")
		return fmt.Errorf("failed to send image message: %w", err)
	}

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Str("recipient", to).
		Str("message_id", msgID).
		Str("timestamp", fmt.Sprintf("%v", resp.Timestamp)).
		Str("mime_type", processed.MimeType).
		Int64("file_size", processed.Size).
		Msg("Image message sent successfully")

	return nil
}

// SendImageMessageMultiSource envia uma mensagem de imagem com múltiplas fontes
func (s *WhatsAppService) SendImageMessageMultiSource(sessionName, to, base64Data, filePath, url, minioID, caption, mimeType string) error {
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

	// Cria o serviço de fonte de mídia
	mediaSourceService := media.NewMediaSourceService(nil, media.GetProjectRoot())

	// Prepara a requisição de mídia
	sourceReq := media.MediaSourceRequest{
		Base64:   base64Data,
		FilePath: filePath,
		URL:      url,
		MinioID:  minioID,
		MimeType: mimeType,
	}

	// Processa a fonte de mídia
	sourceResult, err := mediaSourceService.ProcessMediaSource(sourceReq, entity.MessageTypeImage)
	if err != nil {
		return fmt.Errorf("failed to process media source: %w", err)
	}

	// Processa a mídia para upload
	mediaService := media.NewMediaService()
	processed, err := mediaService.ProcessMediaFromSource(sourceResult, entity.MessageTypeImage)
	if err != nil {
		return fmt.Errorf("failed to process media: %w", err)
	}

	// Faz upload para WhatsApp
	uploaded, err := mediaService.UploadMediaToWhatsApp(waClient.Client, processed.Data, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("failed to upload image: %w", err)
	}

	// Parse do JID de destino
	recipient, ok := s.parseJID(to)
	if !ok {
		return fmt.Errorf("invalid recipient JID: %s", to)
	}

	// Gera ID da mensagem
	msgID := waClient.Client.GenerateMessageID()

	// Cria a mensagem de imagem
	msg := &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			Caption:       &caption,
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			Mimetype:      &processed.MimeType,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(processed.Size)),
			JPEGThumbnail: processed.Thumbnail,
		},
	}

	// Envia a mensagem
	resp, err := waClient.Client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: msgID})
	if err != nil {
		logger.WithComponent("whatsapp").Error().
			Err(err).
			Str("session_name", sessionName).
			Str("recipient", to).
			Str("message_id", msgID).
			Str("source", sourceResult.Source).
			Msg("Failed to send image message")
		return fmt.Errorf("failed to send image message: %w", err)
	}

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Str("recipient", to).
		Str("message_id", msgID).
		Str("timestamp", fmt.Sprintf("%v", resp.Timestamp)).
		Str("mime_type", processed.MimeType).
		Int64("file_size", processed.Size).
		Str("source", sourceResult.Source).
		Str("filename", processed.Filename).
		Msg("Image message sent successfully")

	return nil
}

// SendAudioMessage envia uma mensagem de áudio
func (s *WhatsAppService) SendAudioMessage(sessionName, to, audioData string) error {
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

	// Processa a mídia
	mediaService := media.NewMediaService()
	processed, err := mediaService.ProcessMediaForUpload(audioData, entity.MessageTypeAudio)
	if err != nil {
		return fmt.Errorf("failed to process audio: %w", err)
	}

	// Faz upload para WhatsApp
	uploaded, err := mediaService.UploadMediaToWhatsApp(waClient.Client, processed.Data, whatsmeow.MediaAudio)
	if err != nil {
		return fmt.Errorf("failed to upload audio: %w", err)
	}

	// Parse do JID de destino
	recipient, ok := s.parseJID(to)
	if !ok {
		return fmt.Errorf("invalid recipient JID: %s", to)
	}

	// Gera ID da mensagem
	msgID := waClient.Client.GenerateMessageID()

	// Cria a mensagem de áudio (PTT = Push-to-Talk para mensagem de voz)
	ptt := true
	msg := &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			Mimetype:      &processed.MimeType,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(processed.Size)),
			PTT:           &ptt,
		},
	}

	// Envia a mensagem
	resp, err := waClient.Client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: msgID})
	if err != nil {
		logger.WithComponent("whatsapp").Error().
			Err(err).
			Str("session_name", sessionName).
			Str("recipient", to).
			Str("message_id", msgID).
			Msg("Failed to send audio message")
		return fmt.Errorf("failed to send audio message: %w", err)
	}

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Str("recipient", to).
		Str("message_id", msgID).
		Str("timestamp", fmt.Sprintf("%v", resp.Timestamp)).
		Str("mime_type", processed.MimeType).
		Int64("file_size", processed.Size).
		Msg("Audio message sent successfully")

	return nil
}

// SendAudioMessageMultiSource envia uma mensagem de áudio com múltiplas fontes
func (s *WhatsAppService) SendAudioMessageMultiSource(sessionName, to, base64Data, filePath, url, minioID string) error {
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

	// Cria o serviço de fonte de mídia
	mediaSourceService := media.NewMediaSourceService(nil, media.GetProjectRoot())

	// Prepara a requisição de mídia
	sourceReq := media.MediaSourceRequest{
		Base64:   base64Data,
		FilePath: filePath,
		URL:      url,
		MinioID:  minioID,
	}

	// Processa a fonte de mídia
	sourceResult, err := mediaSourceService.ProcessMediaSource(sourceReq, entity.MessageTypeAudio)
	if err != nil {
		return fmt.Errorf("failed to process media source: %w", err)
	}

	// Processa a mídia para upload
	mediaService := media.NewMediaService()
	processed, err := mediaService.ProcessMediaFromSource(sourceResult, entity.MessageTypeAudio)
	if err != nil {
		return fmt.Errorf("failed to process media: %w", err)
	}

	// Faz upload para WhatsApp
	uploaded, err := mediaService.UploadMediaToWhatsApp(waClient.Client, processed.Data, whatsmeow.MediaAudio)
	if err != nil {
		return fmt.Errorf("failed to upload audio: %w", err)
	}

	// Parse do JID de destino
	recipient, ok := s.parseJID(to)
	if !ok {
		return fmt.Errorf("invalid recipient JID: %s", to)
	}

	// Gera ID da mensagem
	msgID := waClient.Client.GenerateMessageID()

	// Cria a mensagem de áudio seguindo a implementação de referência
	ptt := true
	mimeType := "audio/ogg; codecs=opus" // MIME type específico como na referência

	msg := &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			Mimetype:      &mimeType, // Usar MIME type específico
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(processed.Data))), // Tamanho dos dados originais
			PTT:           &ptt,
		},
	}

	// Envia a mensagem
	resp, err := waClient.Client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: msgID})
	if err != nil {
		logger.WithComponent("whatsapp").Error().
			Err(err).
			Str("session_name", sessionName).
			Str("recipient", to).
			Str("message_id", msgID).
			Str("source", sourceResult.Source).
			Msg("Failed to send audio message")
		return fmt.Errorf("failed to send audio message: %w", err)
	}

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Str("recipient", to).
		Str("message_id", msgID).
		Str("timestamp", fmt.Sprintf("%v", resp.Timestamp)).
		Str("mime_type", processed.MimeType).
		Int64("file_size", processed.Size).
		Str("source", sourceResult.Source).
		Str("filename", processed.Filename).
		Msg("Audio message sent successfully")

	return nil
}

// SendDocumentMessage envia uma mensagem de documento
func (s *WhatsAppService) SendDocumentMessage(sessionName, to, documentData, filename, mimeType string) error {
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

	// Processa a mídia
	mediaService := media.NewMediaService()
	processed, err := mediaService.ProcessMediaForUpload(documentData, entity.MessageTypeDocument)
	if err != nil {
		return fmt.Errorf("failed to process document: %w", err)
	}

	// Faz upload para WhatsApp
	uploaded, err := mediaService.UploadMediaToWhatsApp(waClient.Client, processed.Data, whatsmeow.MediaDocument)
	if err != nil {
		return fmt.Errorf("failed to upload document: %w", err)
	}

	// Parse do JID de destino
	recipient, ok := s.parseJID(to)
	if !ok {
		return fmt.Errorf("invalid recipient JID: %s", to)
	}

	// Gera ID da mensagem
	msgID := waClient.Client.GenerateMessageID()

	// Usa o MIME type fornecido ou o detectado
	finalMimeType := mimeType
	if finalMimeType == "" {
		finalMimeType = processed.MimeType
	}

	// Cria a mensagem de documento
	msg := &waProto.Message{
		DocumentMessage: &waProto.DocumentMessage{
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			Mimetype:      &finalMimeType,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(processed.Size)),
			FileName:      &filename,
		},
	}

	// Envia a mensagem
	resp, err := waClient.Client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: msgID})
	if err != nil {
		logger.WithComponent("whatsapp").Error().
			Err(err).
			Str("session_name", sessionName).
			Str("recipient", to).
			Str("message_id", msgID).
			Str("filename", filename).
			Msg("Failed to send document message")
		return fmt.Errorf("failed to send document message: %w", err)
	}

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Str("recipient", to).
		Str("message_id", msgID).
		Str("timestamp", fmt.Sprintf("%v", resp.Timestamp)).
		Str("mime_type", finalMimeType).
		Str("filename", filename).
		Int64("file_size", processed.Size).
		Msg("Document message sent successfully")

	return nil
}

// SendDocumentMessageMultiSource envia uma mensagem de documento com múltiplas fontes
func (s *WhatsAppService) SendDocumentMessageMultiSource(sessionName, to, base64Data, filePath, url, minioID, filename, mimeType string) error {
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

	// Cria o serviço de fonte de mídia
	mediaSourceService := media.NewMediaSourceService(nil, media.GetProjectRoot())

	// Prepara a requisição de mídia
	sourceReq := media.MediaSourceRequest{
		Base64:   base64Data,
		FilePath: filePath,
		URL:      url,
		MinioID:  minioID,
		MimeType: mimeType,
		Filename: filename,
	}

	// Processa a fonte de mídia
	sourceResult, err := mediaSourceService.ProcessMediaSource(sourceReq, entity.MessageTypeDocument)
	if err != nil {
		return fmt.Errorf("failed to process media source: %w", err)
	}

	// Processa a mídia para upload
	mediaService := media.NewMediaService()
	processed, err := mediaService.ProcessMediaFromSource(sourceResult, entity.MessageTypeDocument)
	if err != nil {
		return fmt.Errorf("failed to process media: %w", err)
	}

	// Faz upload para WhatsApp
	uploaded, err := mediaService.UploadMediaToWhatsApp(waClient.Client, processed.Data, whatsmeow.MediaDocument)
	if err != nil {
		return fmt.Errorf("failed to upload document: %w", err)
	}

	// Parse do JID de destino
	recipient, ok := s.parseJID(to)
	if !ok {
		return fmt.Errorf("invalid recipient JID: %s", to)
	}

	// Gera ID da mensagem
	msgID := waClient.Client.GenerateMessageID()

	// Cria a mensagem de documento seguindo a implementação de referência
	caption := "" // Caption vazia por padrão
	msg := &waProto.Message{
		DocumentMessage: &waProto.DocumentMessage{
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			Mimetype:      &processed.MimeType,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(processed.Data))), // Tamanho dos dados originais
			FileName:      &filename,
			Caption:       &caption, // Campo caption como na referência
		},
	}

	// Envia a mensagem
	resp, err := waClient.Client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: msgID})
	if err != nil {
		logger.WithComponent("whatsapp").Error().
			Err(err).
			Str("session_name", sessionName).
			Str("recipient", to).
			Str("message_id", msgID).
			Str("source", sourceResult.Source).
			Msg("Failed to send document message")
		return fmt.Errorf("failed to send document message: %w", err)
	}

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Str("recipient", to).
		Str("message_id", msgID).
		Str("timestamp", fmt.Sprintf("%v", resp.Timestamp)).
		Str("mime_type", processed.MimeType).
		Int64("file_size", processed.Size).
		Str("source", sourceResult.Source).
		Str("filename", filename).
		Msg("Document message sent successfully")

	return nil
}

// SendLocationMessage envia uma mensagem de localização
func (s *WhatsAppService) SendLocationMessage(sessionName, to string, latitude, longitude float64, name string) error {
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

	// Cria a mensagem de localização
	msg := &waProto.Message{
		LocationMessage: &waProto.LocationMessage{
			DegreesLatitude:  &latitude,
			DegreesLongitude: &longitude,
			Name:             &name,
		},
	}

	// Envia a mensagem
	resp, err := waClient.Client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: msgID})
	if err != nil {
		logger.WithComponent("whatsapp").Error().
			Err(err).
			Str("session_name", sessionName).
			Str("recipient", to).
			Str("message_id", msgID).
			Float64("latitude", latitude).
			Float64("longitude", longitude).
			Str("location_name", name).
			Msg("Failed to send location message")
		return fmt.Errorf("failed to send location message: %w", err)
	}

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Str("recipient", to).
		Str("message_id", msgID).
		Str("timestamp", fmt.Sprintf("%v", resp.Timestamp)).
		Float64("latitude", latitude).
		Float64("longitude", longitude).
		Str("location_name", name).
		Msg("Location message sent successfully")

	return nil
}

// SendContactMessage envia uma mensagem de contato
func (s *WhatsAppService) SendContactMessage(sessionName, to, name, vcard string) error {
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

	// Cria a mensagem de contato
	msg := &waProto.Message{
		ContactMessage: &waProto.ContactMessage{
			DisplayName: &name,
			Vcard:       &vcard,
		},
	}

	// Envia a mensagem
	resp, err := waClient.Client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: msgID})
	if err != nil {
		logger.WithComponent("whatsapp").Error().
			Err(err).
			Str("session_name", sessionName).
			Str("recipient", to).
			Str("message_id", msgID).
			Str("contact_name", name).
			Msg("Failed to send contact message")
		return fmt.Errorf("failed to send contact message: %w", err)
	}

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Str("recipient", to).
		Str("message_id", msgID).
		Str("timestamp", fmt.Sprintf("%v", resp.Timestamp)).
		Str("contact_name", name).
		Msg("Contact message sent successfully")

	return nil
}

// ReactToMessage reage a uma mensagem existente
func (s *WhatsAppService) ReactToMessage(sessionName, to, messageID, reaction string) error {
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

	// Processa a reação (remove se for "remove")
	reactionText := reaction
	if reaction == "remove" {
		reactionText = ""
	}

	// Determina se a mensagem é nossa ou não baseado no prefixo
	fromMe := false
	actualMessageID := messageID
	if strings.HasPrefix(messageID, "me:") {
		fromMe = true
		actualMessageID = messageID[len("me:"):]
	}

	// Cria a mensagem de reação
	msg := &waProto.Message{
		ReactionMessage: &waProto.ReactionMessage{
			Key: &waCommon.MessageKey{
				RemoteJID: proto.String(recipient.String()),
				FromMe:    proto.Bool(fromMe),
				ID:        proto.String(actualMessageID),
			},
			Text:              proto.String(reactionText),
			GroupingKey:       proto.String(reactionText),
			SenderTimestampMS: proto.Int64(time.Now().UnixMilli()),
		},
	}

	// Envia a reação
	resp, err := waClient.Client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: actualMessageID})
	if err != nil {
		logger.WithComponent("whatsapp").Error().
			Err(err).
			Str("session_name", sessionName).
			Str("recipient", to).
			Str("message_id", actualMessageID).
			Str("reaction", reaction).
			Bool("from_me", fromMe).
			Msg("Failed to send reaction")
		return fmt.Errorf("failed to send reaction: %w", err)
	}

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Str("recipient", to).
		Str("message_id", actualMessageID).
		Str("timestamp", fmt.Sprintf("%v", resp.Timestamp)).
		Str("reaction", reaction).
		Bool("from_me", fromMe).
		Msg("Reaction sent successfully")

	return nil
}

// SendVideoMessageMultiSource envia uma mensagem de vídeo com múltiplas fontes
func (s *WhatsAppService) SendVideoMessageMultiSource(sessionName, to, base64Data, filePath, url, minioID, caption, mimeType string, jpegThumbnail []byte) error {
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

	// Cria o serviço de fonte de mídia
	mediaSourceService := media.NewMediaSourceService(nil, media.GetProjectRoot())

	// Prepara a requisição de mídia
	sourceReq := media.MediaSourceRequest{
		Base64:   base64Data,
		FilePath: filePath,
		URL:      url,
		MinioID:  minioID,
		MimeType: mimeType,
	}

	// Processa a fonte de mídia
	sourceResult, err := mediaSourceService.ProcessMediaSource(sourceReq, entity.MessageTypeVideo)
	if err != nil {
		return fmt.Errorf("failed to process media source: %w", err)
	}

	// Processa a mídia para upload
	mediaService := media.NewMediaService()
	processed, err := mediaService.ProcessMediaFromSource(sourceResult, entity.MessageTypeVideo)
	if err != nil {
		return fmt.Errorf("failed to process media: %w", err)
	}

	// Faz upload para WhatsApp
	uploaded, err := mediaService.UploadMediaToWhatsApp(waClient.Client, processed.Data, whatsmeow.MediaVideo)
	if err != nil {
		return fmt.Errorf("failed to upload video: %w", err)
	}

	// Parse do JID de destino
	recipient, ok := s.parseJID(to)
	if !ok {
		return fmt.Errorf("invalid recipient JID: %s", to)
	}

	// Gera ID da mensagem
	msgID := waClient.Client.GenerateMessageID()

	// Cria a mensagem de vídeo seguindo a implementação de referência
	msg := &waProto.Message{
		VideoMessage: &waProto.VideoMessage{
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			Mimetype:      &processed.MimeType,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(processed.Data))), // Tamanho dos dados originais
			Caption:       &caption,
			JPEGThumbnail: jpegThumbnail, // Thumbnail opcional
		},
	}

	// Envia a mensagem
	resp, err := waClient.Client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: msgID})
	if err != nil {
		logger.WithComponent("whatsapp").Error().
			Err(err).
			Str("session_name", sessionName).
			Str("recipient", to).
			Str("message_id", msgID).
			Str("source", sourceResult.Source).
			Msg("Failed to send video message")
		return fmt.Errorf("failed to send video message: %w", err)
	}

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Str("recipient", to).
		Str("message_id", msgID).
		Str("timestamp", fmt.Sprintf("%v", resp.Timestamp)).
		Str("mime_type", processed.MimeType).
		Int64("file_size", processed.Size).
		Str("source", sourceResult.Source).
		Str("caption", caption).
		Msg("Video message sent successfully")

	return nil
}

// EditMessage edita uma mensagem de texto existente
func (s *WhatsAppService) EditMessage(sessionName, to, messageID, newText string) error {
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

	// Determina se a mensagem é nossa ou não baseado no prefixo
	fromMe := false
	actualMessageID := messageID
	if strings.HasPrefix(messageID, "me:") {
		fromMe = true
		actualMessageID = messageID[len("me:"):]
	}

	// Cria a mensagem de texto para edição
	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(newText),
		},
	}

	// Usa o método BuildEdit do whatsmeow para criar a mensagem de edição
	editMsg := waClient.Client.BuildEdit(recipient, actualMessageID, msg)

	// Envia a edição
	resp, err := waClient.Client.SendMessage(context.Background(), recipient, editMsg, whatsmeow.SendRequestExtra{})
	if err != nil {
		logger.WithComponent("whatsapp").Error().
			Err(err).
			Str("session_name", sessionName).
			Str("recipient", to).
			Str("message_id", actualMessageID).
			Str("new_text", newText).
			Bool("from_me", fromMe).
			Msg("Failed to edit message")
		return fmt.Errorf("failed to edit message: %w", err)
	}

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Str("recipient", to).
		Str("message_id", actualMessageID).
		Str("timestamp", fmt.Sprintf("%v", resp.Timestamp)).
		Str("new_text", newText).
		Bool("from_me", fromMe).
		Msg("Message edited successfully")

	return nil
}

// SendPollMessage envia uma mensagem de enquete (apenas para grupos)
func (s *WhatsAppService) SendPollMessage(sessionName, to, header string, options []string, maxSelections int) error {
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

	// Verifica se é um grupo (enquetes só funcionam em grupos)
	if recipient.Server != "g.us" {
		return fmt.Errorf("polls can only be sent to groups")
	}

	// Define maxSelections padrão se não fornecido
	if maxSelections <= 0 {
		maxSelections = 1
	}

	// Gera ID da mensagem
	msgID := waClient.Client.GenerateMessageID()

	// Usa o método BuildPollCreation do whatsmeow
	pollMsg := waClient.Client.BuildPollCreation(header, options, maxSelections)

	// Envia a enquete
	resp, err := waClient.Client.SendMessage(context.Background(), recipient, pollMsg, whatsmeow.SendRequestExtra{ID: msgID})
	if err != nil {
		logger.WithComponent("whatsapp").Error().
			Err(err).
			Str("session_name", sessionName).
			Str("recipient", to).
			Str("message_id", msgID).
			Str("header", header).
			Int("options_count", len(options)).
			Int("max_selections", maxSelections).
			Msg("Failed to send poll message")
		return fmt.Errorf("failed to send poll message: %w", err)
	}

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Str("recipient", to).
		Str("message_id", msgID).
		Str("timestamp", fmt.Sprintf("%v", resp.Timestamp)).
		Str("header", header).
		Int("options_count", len(options)).
		Int("max_selections", maxSelections).
		Msg("Poll message sent successfully")

	return nil
}

// SendListMessage envia uma mensagem de lista interativa
func (s *WhatsAppService) SendListMessage(sessionName, to, header, body, footer, buttonText string, sections []entity.ListSection) error {
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

	// Converte as seções para o formato do whatsmeow
	var listSections []*waProto.ListMessage_Section
	for _, section := range sections {
		var rows []*waProto.ListMessage_Row
		for _, item := range section.Rows {
			rows = append(rows, &waProto.ListMessage_Row{
				Title:       proto.String(item.Title),
				Description: proto.String(item.Description),
				RowID:       proto.String(item.RowID),
			})
		}

		listSections = append(listSections, &waProto.ListMessage_Section{
			Title: proto.String(section.Title),
			Rows:  rows,
		})
	}

	// Gera ID da mensagem
	msgID := waClient.Client.GenerateMessageID()

	// Cria a mensagem de lista
	msg := &waProto.Message{
		ListMessage: &waProto.ListMessage{
			Title:       proto.String(header),
			Description: proto.String(body),
			FooterText:  proto.String(footer),
			ButtonText:  proto.String(buttonText),
			ListType:    waProto.ListMessage_SINGLE_SELECT.Enum(),
			Sections:    listSections,
		},
	}

	// Envia a lista
	resp, err := waClient.Client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: msgID})
	if err != nil {
		logger.WithComponent("whatsapp").Error().
			Err(err).
			Str("session_name", sessionName).
			Str("recipient", to).
			Str("message_id", msgID).
			Str("header", header).
			Int("sections_count", len(sections)).
			Msg("Failed to send list message")
		return fmt.Errorf("failed to send list message: %w", err)
	}

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Str("recipient", to).
		Str("message_id", msgID).
		Str("timestamp", fmt.Sprintf("%v", resp.Timestamp)).
		Str("header", header).
		Int("sections_count", len(sections)).
		Msg("List message sent successfully")

	return nil
}

// SendStickerMessage envia uma mensagem de sticker
func (s *WhatsAppService) SendStickerMessage(sessionName, to, stickerData string) error {
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

	// Processa a mídia
	mediaService := media.NewMediaService()
	processed, err := mediaService.ProcessMediaForUpload(stickerData, entity.MessageTypeSticker)
	if err != nil {
		return fmt.Errorf("failed to process sticker: %w", err)
	}

	// Faz upload para WhatsApp
	uploaded, err := mediaService.UploadMediaToWhatsApp(waClient.Client, processed.Data, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("failed to upload sticker: %w", err)
	}

	// Parse do JID de destino
	recipient, ok := s.parseJID(to)
	if !ok {
		return fmt.Errorf("invalid recipient JID: %s", to)
	}

	// Gera ID da mensagem
	msgID := waClient.Client.GenerateMessageID()

	// Cria a mensagem de sticker
	msg := &waProto.Message{
		StickerMessage: &waProto.StickerMessage{
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			Mimetype:      &processed.MimeType,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(processed.Size)),
		},
	}

	// Envia a mensagem
	resp, err := waClient.Client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: msgID})
	if err != nil {
		logger.WithComponent("whatsapp").Error().
			Err(err).
			Str("session_name", sessionName).
			Str("recipient", to).
			Str("message_id", msgID).
			Msg("Failed to send sticker message")
		return fmt.Errorf("failed to send sticker message: %w", err)
	}

	logger.WithComponent("whatsapp").Info().
		Str("session_name", sessionName).
		Str("recipient", to).
		Str("message_id", msgID).
		Str("timestamp", fmt.Sprintf("%v", resp.Timestamp)).
		Str("mime_type", processed.MimeType).
		Int64("file_size", processed.Size).
		Msg("Sticker message sent successfully")

	return nil
}

// startClient inicia o cliente WhatsApp para uma sessão
func (s *WhatsAppService) startClient(sessionID string, session *entity.Session) {
	logger.WithComponent("whatsapp").Info().Str("session_name", session.Session).Msg("Starting WhatsApp client")

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
		Status:      entity.StatusConnecting,
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
	// Busca informações da sessão para log mais detalhado
	if session, err := s.repository.GetByID(waClient.SessionID); err == nil {
		logger.WithComponent("whatsapp").Info().Str("session_name", session.Session).Msg("Already logged in, connecting...")
	} else {
		logger.WithComponent("whatsapp").Info().Msg("Already logged in, connecting...")
	}

	err := waClient.Client.Connect()
	if err != nil {
		logger.WithSession(waClient.SessionID).Error().Err(err).Msg("Failed to connect existing client")
		s.UpdateSessionStatus(waClient.SessionID, entity.StatusDisconnected)

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
			// Busca informações da sessão para log mais detalhado
			if session, err := s.repository.GetByID(waClient.SessionID); err == nil {
				logger.WithComponent("whatsapp").Info().Str("session_name", session.Session).Msg("QR Code generated - scan with your WhatsApp")
			} else {
				logger.WithComponent("whatsapp").Info().Msg("QR Code generated - scan with your WhatsApp")
			}
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
				logger.WithComponent("whatsapp").Error().Err(err).Str("session_name", session.Session).Msg("Failed to update session with QR code")
			} else {
				logger.WithComponent("whatsapp").Info().Str("session_name", session.Session).Msg("QR code generated and saved")
			}

		case "timeout":
			logger.WithSession(waClient.SessionID).Warn().Msg("QR code timeout")
			s.UpdateSessionStatus(waClient.SessionID, entity.StatusDisconnected)

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
			// Busca informações da sessão para log mais detalhado
			if session, err := s.repository.GetByID(waClient.SessionID); err == nil {
				logger.WithComponent("whatsapp").Info().Str("session_name", session.Session).Msg("QR pairing successful")
			} else {
				logger.WithComponent("whatsapp").Info().Msg("QR pairing successful")
			}
			s.UpdateSessionStatus(waClient.SessionID, entity.StatusConnected)

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
			// Busca informações da sessão para log mais detalhado
			if session, err := s.repository.GetByID(waClient.SessionID); err == nil {
				logger.WithComponent("whatsapp").Info().Str("session_name", session.Session).Msg("WhatsApp connected")
			} else {
				logger.WithComponent("whatsapp").Info().Msg("WhatsApp connected")
			}
			s.UpdateSessionStatus(waClient.SessionID, entity.StatusConnected)

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
				session.Status = entity.StatusConnected
				session.UpdatedAt = time.Now()
				s.repository.Update(session)
			}

		case *events.LoggedOut:
			logger.WithSession(waClient.SessionID).Info().
				Str("reason", evt.Reason.String()).
				Msg("Logged out from WhatsApp")

			s.UpdateSessionStatus(waClient.SessionID, entity.StatusDisconnected)
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

		s.UpdateSessionStatus(waClient.SessionID, entity.StatusDisconnected)

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
