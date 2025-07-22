package whatsapp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"zapcore/internal/domain/chat"
	"zapcore/internal/domain/contact"
	"zapcore/internal/domain/message"
	"zapcore/internal/infra/storage"
	"zapcore/pkg/logger"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// Constantes para padronização de logging
const (
	LogComponentEvents = "events"
	LogFieldSessionID  = "session_id"
	LogFieldMessageID  = "message_id"
	LogFieldChatJID    = "chat_jid"
	LogFieldEventType  = "event_type"
	LogFieldError      = "error"
)

// StorageHandler gerencia a persistência automática de eventos do WhatsApp
type StorageHandler struct {
	messageRepo     message.Repository
	chatRepo        chat.Repository
	contactRepo     contact.Repository
	mediaDownloader *MediaDownloader
	logger          *logger.Logger
	handlers        *EventHandlers
	storage         *StorageOperations
}

// NewStorageHandler cria uma nova instância do handler de storage
func NewStorageHandler(
	messageRepo message.Repository,
	chatRepo chat.Repository,
	contactRepo contact.Repository,
	mediaDownloader *MediaDownloader,
) *StorageHandler {
	handler := &StorageHandler{
		messageRepo:     messageRepo,
		chatRepo:        chatRepo,
		contactRepo:     contactRepo,
		mediaDownloader: mediaDownloader,
		logger:          logger.Get(),
	}

	// Inicializar componentes
	handler.storage = NewStorageOperations(handler)
	handler.handlers = NewEventHandlers(handler)

	return handler
}

// HandleEvent processa eventos do WhatsApp e os persiste no banco de dados
func (h *StorageHandler) HandleEvent(ctx context.Context, sessionID uuid.UUID, evt any) error {
	switch v := evt.(type) {
	case *events.Message:
		return h.handlers.HandleMessage(ctx, sessionID, v)
	case *events.UndecryptableMessage:
		return h.handlers.HandleUndecryptableMessage(ctx, sessionID, v)
	case *events.Receipt:
		return h.handlers.HandleReceipt(ctx, sessionID, v)
	case *events.HistorySync:
		return h.handleHistorySync(ctx, sessionID, v)
	case *events.OfflineSyncPreview:
		return h.handleOfflineSyncPreview(ctx, sessionID, v)
	case *events.OfflineSyncCompleted:
		return h.handleOfflineSyncCompleted(ctx, sessionID, v)
	case *events.Contact:
		return h.handlers.HandleContact(ctx, sessionID, v)
	case *events.PushName:
		return h.handlers.HandlePushName(ctx, sessionID, v)
	case *events.Presence:
		return h.handlers.HandlePresence(ctx, sessionID, v)
	case *events.ChatPresence:
		return h.handlers.HandleChatPresence(ctx, sessionID, v)
	case *events.Mute:
		return h.handlers.HandleMute(ctx, sessionID, v)
	case *events.Archive:
		return h.handlers.HandleArchive(ctx, sessionID, v)
	case *events.Pin:
		return h.handlers.HandlePin(ctx, sessionID, v)
	case *events.GroupInfo:
		return h.handlers.HandleGroupInfo(ctx, sessionID, v)
	case *events.Picture:
		return h.handlers.HandlePicture(ctx, sessionID, v)
	default:
		// Log eventos não tratados para debug
		h.logger.Debug().
			Str(LogFieldSessionID, sessionID.String()).
			Str(LogFieldEventType, fmt.Sprintf("%T", evt)).
			Str("component", LogComponentEvents).
			Msg("Evento não tratado pelo storage handler")
		return nil
	}
}

// Métodos de acesso aos componentes extraídos

// processMessageContent delega para StorageOperations
func (h *StorageHandler) processMessageContent(msg *message.Message, msgContent *waE2E.Message) error {
	return h.storage.ProcessMessageContent(msg, msgContent)
}

// processMediaMessage delega para StorageOperations
func (h *StorageHandler) processMediaMessage(ctx context.Context, msg *message.Message, evt *events.Message) error {
	return h.storage.ProcessMediaMessage(ctx, msg, evt)
}

// updateChatFromMessage delega para StorageOperations
func (h *StorageHandler) updateChatFromMessage(ctx context.Context, sessionID uuid.UUID, evt *events.Message) error {
	return h.storage.UpdateChatFromMessage(ctx, sessionID, evt)
}

// updateContactFromMessage delega para StorageOperations
func (h *StorageHandler) updateContactFromMessage(ctx context.Context, sessionID uuid.UUID, evt *events.Message) error {
	return h.storage.UpdateContactFromMessage(ctx, sessionID, evt)
}

// storeRawPayload delega para StorageOperations
func (h *StorageHandler) storeRawPayload(msg *message.Message, evt *events.Message) error {
	return h.storage.StoreRawPayload(msg, evt)
}

// storeUndecryptableRawPayload delega para StorageOperations
func (h *StorageHandler) storeUndecryptableRawPayload(msg *message.Message, evt *events.UndecryptableMessage) error {
	return h.storage.StoreUndecryptableRawPayload(msg, evt)
}

// Handlers para eventos que ainda não foram extraídos

// handleHistorySync processa eventos de sincronização de histórico
func (h *StorageHandler) handleHistorySync(ctx context.Context, sessionID uuid.UUID, evt *events.HistorySync) error {
	syncType := "unknown"
	if evt.Data.SyncType != nil {
		syncType = evt.Data.SyncType.String()
	}

	h.logger.Info().
		Str("session_id", sessionID.String()).
		Str("sync_type", syncType).
		Int("conversations_count", len(evt.Data.Conversations)).
		Msg("Processando sincronização de histórico")

	// Processar conversas do histórico
	for _, conversation := range evt.Data.Conversations {
		if conversation.ID == nil {
			continue
		}

		chatJID := *conversation.ID
		h.logger.Debug().
			Str("session_id", sessionID.String()).
			Str("chat_jid", chatJID).
			Int("messages_count", len(conversation.Messages)).
			Msg("Processando conversa do histórico")

		// Processar mensagens da conversa
		for _, historyMsg := range conversation.Messages {
			if historyMsg.Message == nil || historyMsg.Message.Key == nil {
				continue
			}

			msgKey := historyMsg.Message.Key
			if msgKey.ID == nil {
				continue
			}

			msgID := *msgKey.ID

			h.logger.Info().
				Str("session_id", sessionID.String()).
				Str("message_id", msgID).
				Msg("🔍 VERIFICANDO: Existência da mensagem histórica")

			// TEMPORÁRIO: Comentar verificação de existência para testar download de mídias
			// Verificar se a mensagem já existe para esta sessão específica
			exists, err := h.messageRepo.ExistsByMsgIDAndSessionID(ctx, msgID, sessionID)
			if err != nil {
				h.logger.Error().Err(err).
					Str("message_id", msgID).
					Str("session_id", sessionID.String()).
					Msg("❌ ERRO: Falha ao verificar existência da mensagem histórica")
				continue
			}

			if exists {
				h.logger.Info().
					Str("message_id", msgID).
					Str("session_id", sessionID.String()).
					Msg("⏭️ FORÇANDO REPROCESSAMENTO: Mensagem existe mas vamos processar para testar mídia")
				// continue // COMENTADO TEMPORARIAMENTE PARA TESTAR MÍDIAS
			}

			h.logger.Info().
				Str("session_id", sessionID.String()).
				Str("message_id", msgID).
				Msg("✅ PROCESSANDO MENSAGEM: Forçando processamento para testar mídia")

			// Processar mensagem histórica com dados completos
			if err := h.processCompleteHistoricalMessage(ctx, sessionID, historyMsg); err != nil {
				h.logger.Error().Err(err).
					Str("message_id", msgID).
					Msg("Erro ao processar mensagem histórica")
				continue
			}

			h.logger.Debug().
				Str("session_id", sessionID.String()).
				Str("message_id", msgID).
				Msg("Mensagem histórica processada com sucesso")
		}
	}

	h.logger.Info().
		Str("session_id", sessionID.String()).
		Str("sync_type", syncType).
		Msg("Sincronização de histórico concluída")

	return nil
}

// handleOfflineSyncPreview processa eventos de preview de sincronização offline
func (h *StorageHandler) handleOfflineSyncPreview(_ context.Context, sessionID uuid.UUID, evt *events.OfflineSyncPreview) error {
	h.logger.Info().
		Str("session_id", sessionID.String()).
		Int("total", evt.Total).
		Int("messages", evt.Messages).
		Int("notifications", evt.Notifications).
		Int("receipts", evt.Receipts).
		Int("app_data_changes", evt.AppDataChanges).
		Msg("Preview de sincronização offline recebido")

	// Log detalhado para debug
	h.logger.Debug().
		Str("session_id", sessionID.String()).
		Interface("raw_payload", evt).
		Msg("Payload completo do OfflineSyncPreview")

	return nil
}

// handleOfflineSyncCompleted processa eventos de conclusão de sincronização offline
func (h *StorageHandler) handleOfflineSyncCompleted(_ context.Context, sessionID uuid.UUID, evt *events.OfflineSyncCompleted) error {
	h.logger.Info().
		Str("session_id", sessionID.String()).
		Int("count", evt.Count).
		Msg("Sincronização offline concluída")

	// Log detalhado para debug
	h.logger.Debug().
		Str("session_id", sessionID.String()).
		Interface("raw_payload", evt).
		Msg("Payload completo do OfflineSyncCompleted")

	return nil
}

// processCompleteHistoricalMessage processa uma mensagem histórica com todos os dados disponíveis
func (h *StorageHandler) processCompleteHistoricalMessage(ctx context.Context, sessionID uuid.UUID, historyMsg any) error {
	h.logger.Info().
		Str("session_id", sessionID.String()).
		Str("type", fmt.Sprintf("%T", historyMsg)).
		Msg("🚀 INICIANDO: processCompleteHistoricalMessage")

	// Log otimizado para debug quando necessário
	h.logger.Debug().
		Str("session_id", sessionID.String()).
		Str("type", fmt.Sprintf("%T", historyMsg)).
		Interface("data", historyMsg).
		Msg("Dados brutos da mensagem histórica recebidos")

	h.logger.Info().
		Str("session_id", sessionID.String()).
		Msg("🔄 PASSO 1: Iniciando serialização JSON")

	// Tentar converter para JSON e depois para map para acessar os dados
	jsonData, err := json.Marshal(historyMsg)
	if err != nil {
		h.logger.Error().Err(err).
			Str("session_id", sessionID.String()).
			Msg("❌ ERRO: Falha na serialização JSON")
		return fmt.Errorf("erro ao serializar mensagem histórica: %w", err)
	}

	h.logger.Info().
		Str("session_id", sessionID.String()).
		Int("json_size", len(jsonData)).
		Msg("✅ PASSO 1: JSON serializado com sucesso")

	var msgData map[string]any
	if err := json.Unmarshal(jsonData, &msgData); err != nil {
		h.logger.Error().Err(err).
			Str("session_id", sessionID.String()).
			Msg("❌ ERRO: Falha na deserialização JSON")
		return fmt.Errorf("erro ao deserializar mensagem histórica: %w", err)
	}

	h.logger.Info().
		Str("session_id", sessionID.String()).
		Msg("✅ PASSO 2: JSON deserializado com sucesso")

	h.logger.Info().
		Str("session_id", sessionID.String()).
		Msg("🔄 PASSO 3: Verificando campo 'message'")

	messageField, ok := msgData["message"].(map[string]any)
	if !ok {
		h.logger.Error().
			Str("session_id", sessionID.String()).
			Interface("available_keys", getMapKeys(msgData)).
			Msg("❌ ERRO: Campo 'message' não encontrado")
		return fmt.Errorf("campo message não encontrado")
	}

	h.logger.Info().
		Str("session_id", sessionID.String()).
		Msg("✅ PASSO 3: Campo 'message' encontrado")

	h.logger.Info().
		Str("session_id", sessionID.String()).
		Msg("🔄 PASSO 4: Verificando campo 'key'")

	keyField, ok := messageField["key"].(map[string]any)
	if !ok {
		h.logger.Error().
			Str("session_id", sessionID.String()).
			Interface("available_keys", getMapKeys(messageField)).
			Msg("❌ ERRO: Campo 'key' não encontrado")
		return fmt.Errorf("campo key não encontrado")
	}

	h.logger.Info().
		Str("session_id", sessionID.String()).
		Msg("✅ PASSO 4: Campo 'key' encontrado")

	// Extrair dados básicos
	msgID, ok := keyField["ID"].(string)
	if !ok {
		return fmt.Errorf("ID da mensagem não encontrado")
	}

	remoteJID, ok := keyField["remoteJID"].(string)
	if !ok {
		return fmt.Errorf("remoteJID não encontrado")
	}

	fromMe, _ := keyField["fromMe"].(bool)

	// Determinar direção
	var direction message.MessageDirection
	if fromMe {
		direction = message.MessageDirectionOutbound
	} else {
		direction = message.MessageDirectionInbound
	}

	// Processar conteúdo e determinar tipo da mensagem
	msgType, content, mediaData := h.extractMessageContent(messageField)

	// Criar entidade de mensagem
	msg := message.NewMessage(sessionID, msgType, direction)
	msg.MsgID = msgID
	msg.ChatJID = remoteJID
	msg.Content = content

	// Definir sender baseado na direção
	if fromMe {
		msg.SenderJID = "me"
	} else {
		if participant, ok := keyField["participant"].(string); ok {
			msg.SenderJID = participant
		} else {
			msg.SenderJID = remoteJID
		}
	}

	// Extrair timestamp
	if timestamp, ok := messageField["messageTimestamp"].(float64); ok {
		msg.Timestamp = time.Unix(int64(timestamp), 0)
	} else {
		msg.Timestamp = time.Now()
	}

	// Extrair status da mensagem
	if statusField, ok := messageField["status"].(float64); ok {
		switch int(statusField) {
		case 1:
			msg.Status = message.MessageStatusSent
		case 2:
			msg.Status = message.MessageStatusDelivered
		case 3, 4:
			msg.Status = message.MessageStatusRead
		default:
			msg.Status = message.MessageStatusDelivered
		}
	} else {
		msg.Status = message.MessageStatusDelivered
	}

	// Definir se é de mim
	msg.IsFromMe = fromMe

	// Armazenar payload bruto completo com dados de mídia
	msg.RawPayload = map[string]any{
		"source":     "history_sync_complete",
		"message_id": msgID,
		"timestamp":  msg.Timestamp,
		"from_me":    fromMe,
		"status":     msg.Status,
		"content":    msg.Content,
		"type":       msgType,
		"media_data": mediaData,
		"raw_data":   historyMsg,
	}

	// Salvar mensagem
	if err := h.messageRepo.Create(ctx, msg); err != nil {
		return fmt.Errorf("erro ao salvar mensagem histórica completa: %w", err)
	}

	h.logger.Debug().
		Str("session_id", sessionID.String()).
		Str("message_id", msgID).
		Msg("✅ DEBUG: Mensagem histórica salva, iniciando verificação de mídia")

	// Debug: Verificar se há dados de mídia
	h.logger.Debug().
		Str("session_id", sessionID.String()).
		Str("message_id", msgID).
		Interface("media_data", mediaData).
		Bool("media_downloader_available", h.mediaDownloader != nil).
		Msg("🔍 DEBUG: Verificando dados de mídia")

	// Processar mídia se presente e MediaDownloader disponível
	if len(mediaData) > 0 {
		h.logger.Debug().
			Str("session_id", sessionID.String()).
			Str("message_id", msgID).
			Msg("🔍 DEBUG: Dados de mídia encontrados, verificando URL")

		hasURL := h.hasMediaURL(mediaData)
		h.logger.Debug().
			Str("session_id", sessionID.String()).
			Str("message_id", msgID).
			Bool("has_media_url", hasURL).
			Bool("media_downloader_available", h.mediaDownloader != nil).
			Msg("🔍 DEBUG: Resultado da verificação de URL")

		if hasURL && h.mediaDownloader != nil {
			// Verificar se a mensagem é recente o suficiente para tentar download
			// URLs do WhatsApp expiram após um tempo, então só tentamos baixar mídias recentes
			cutoffTime := time.Now().AddDate(0, 0, -7) // 7 dias atrás (URLs do WhatsApp expiram rapidamente)

			if msg.Timestamp.After(cutoffTime) {
				h.logger.Info().
					Str("session_id", sessionID.String()).
					Str("message_id", msgID).
					Time("message_timestamp", msg.Timestamp).
					Time("cutoff_time", cutoffTime).
					Msg("🎬 INICIANDO: Processamento de mídia histórica (mensagem recente)")

				if err := h.processHistoricalMedia(ctx, msg, mediaData); err != nil {
					h.logger.Error().Err(err).
						Str("message_id", msgID).
						Time("message_timestamp", msg.Timestamp).
						Msg("❌ ERRO: Falha ao processar mídia histórica")
					// Não retorna erro para não impedir o salvamento da mensagem
				} else {
					h.logger.Info().
						Str("session_id", sessionID.String()).
						Str("message_id", msgID).
						Msg("✅ SUCESSO: Mídia histórica processada")
				}
			} else {
				h.logger.Debug().
					Str("session_id", sessionID.String()).
					Str("message_id", msgID).
					Time("message_timestamp", msg.Timestamp).
					Time("cutoff_time", cutoffTime).
					Msg("⏭️ PULANDO: Mídia muito antiga, URLs provavelmente expiradas")
			}
		} else {
			h.logger.Debug().
				Str("session_id", sessionID.String()).
				Str("message_id", msgID).
				Bool("has_url", hasURL).
				Bool("has_downloader", h.mediaDownloader != nil).
				Msg("🚫 DEBUG: Mídia não processada - condições não atendidas")
		}
	} else {
		h.logger.Debug().
			Str("session_id", sessionID.String()).
			Str("message_id", msgID).
			Msg("🚫 DEBUG: Nenhum dado de mídia encontrado")
	}

	return nil
}

// hasMediaURL verifica se os dados de mídia contêm uma URL válida
func (h *StorageHandler) hasMediaURL(mediaData map[string]any) bool {
	h.logger.Debug().
		Interface("media_data_keys", getMapKeys(mediaData)).
		Msg("🔍 DEBUG: Iniciando verificação de URL de mídia")

	if mediaData == nil {
		h.logger.Debug().Msg("🚫 DEBUG: mediaData é nil")
		return false
	}

	if len(mediaData) == 0 {
		h.logger.Debug().Msg("🚫 DEBUG: mediaData está vazio")
		return false
	}

	// Verificar se há URL nos dados de mídia (tanto maiúsculo quanto minúsculo)
	urlKeys := []string{"url", "URL"}
	for _, key := range urlKeys {
		if url, exists := mediaData[key]; exists {
			h.logger.Debug().
				Str("key", key).
				Interface("url_value", url).
				Msg("🔍 DEBUG: Chave de URL encontrada")

			if urlStr, ok := url.(string); ok {
				h.logger.Debug().
					Str("key", key).
					Str("url", urlStr).
					Bool("is_whatsapp_url", strings.HasPrefix(urlStr, "https://mmg.whatsapp.net/")).
					Msg("🔍 DEBUG: URL convertida para string")

				if strings.HasPrefix(urlStr, "https://mmg.whatsapp.net/") {
					h.logger.Info().
						Str("url", urlStr).
						Str("found_in", "direct_key").
						Msg("✅ URL de mídia detectada nos dados extraídos")
					return true
				}
			} else {
				h.logger.Debug().
					Str("key", key).
					Interface("url_value", url).
					Msg("🚫 DEBUG: URL não é string")
			}
		}
	}

	// Verificar URLs em diferentes tipos de mensagem de mídia (fallback)
	mediaTypes := []string{"audioMessage", "imageMessage", "videoMessage", "documentMessage", "stickerMessage"}

	for _, mediaType := range mediaTypes {
		if mediaObj, exists := mediaData[mediaType]; exists {
			h.logger.Debug().
				Str("media_type", mediaType).
				Interface("media_obj", mediaObj).
				Msg("🔍 DEBUG: Tipo de mídia encontrado")

			if mediaMap, ok := mediaObj.(map[string]any); ok {
				h.logger.Debug().
					Str("media_type", mediaType).
					Interface("media_map_keys", getMapKeys(mediaMap)).
					Msg("🔍 DEBUG: Mapa de mídia convertido")

				for _, key := range urlKeys {
					if url, exists := mediaMap[key]; exists {
						h.logger.Debug().
							Str("media_type", mediaType).
							Str("key", key).
							Interface("url_value", url).
							Msg("🔍 DEBUG: URL encontrada em tipo específico")

						if urlStr, ok := url.(string); ok {
							h.logger.Debug().
								Str("media_type", mediaType).
								Str("key", key).
								Str("url", urlStr).
								Bool("is_whatsapp_url", strings.HasPrefix(urlStr, "https://mmg.whatsapp.net/")).
								Msg("🔍 DEBUG: URL de tipo específico convertida")

							if strings.HasPrefix(urlStr, "https://mmg.whatsapp.net/") {
								h.logger.Info().
									Str("url", urlStr).
									Str("media_type", mediaType).
									Str("found_in", "media_type").
									Msg("✅ URL de mídia detectada em tipo específico")
								return true
							}
						}
					}
				}
			} else {
				h.logger.Debug().
					Str("media_type", mediaType).
					Interface("media_obj", mediaObj).
					Msg("🚫 DEBUG: Objeto de mídia não é mapa")
			}
		}
	}

	h.logger.Debug().Msg("🚫 DEBUG: Nenhuma URL de mídia válida encontrada")
	return false
}

// getMapKeys retorna as chaves de um mapa para debug
func getMapKeys(m map[string]any) []string {
	if m == nil {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// processHistoricalMedia processa mídia de mensagens históricas usando DownloadMediaWithPath do whatsmeow
func (h *StorageHandler) processHistoricalMedia(ctx context.Context, msg *message.Message, mediaData map[string]any) error {
	msgIDStr := msg.ID.String()

	// Verificar se a mídia já foi processada para esta mensagem
	if h.storage.IsMediaAlreadyProcessed(msgIDStr) {
		h.logger.Debug().
			Str("message_id", msgIDStr).
			Msg("Mídia já processada para esta mensagem, pulando")
		return nil
	}

	h.logger.Info().
		Str("message_id", msgIDStr).
		Interface("media_data", mediaData).
		Msg("🎬 Iniciando processamento de mídia histórica")

	// Tentar extrair informações necessárias para download via whatsmeow
	directPath, mediaKey, fileEncSHA256, fileSHA256, fileLength, mimeType, fileName := h.extractMediaMetadata(mediaData)

	if directPath == "" || len(mediaKey) == 0 {
		h.logger.Warn().
			Str("message_id", msgIDStr).
			Str("direct_path", directPath).
			Int("media_key_len", len(mediaKey)).
			Msg("⚠️ Metadados insuficientes para download via whatsmeow - usando fallback")

		// Usar fallback se não temos metadados suficientes
		return h.processHistoricalMediaFallback(ctx, msg, mediaData)
	}

	// Obter cliente whatsmeow da sessão
	client, err := h.getWhatsmeowClient(msg.SessionID)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("message_id", msgIDStr).
			Msg("❌ Erro ao obter cliente whatsmeow - usando fallback")
		return h.processHistoricalMediaFallback(ctx, msg, mediaData)
	}

	// Se não há nome de arquivo, usar o ID da mensagem
	if fileName == "" {
		fileName = msgIDStr
	}

	h.logger.Info().
		Str("message_id", msgIDStr).
		Str("direct_path", directPath).
		Str("mime_type", mimeType).
		Str("file_name", fileName).
		Uint64("file_length", fileLength).
		Msg("🔑 Baixando mídia via DownloadMediaWithPath com chaves de descriptografia")

	// Usar DownloadMediaWithPath para download da mídia
	h.logger.Info().
		Str("message_id", msgIDStr).
		Str("direct_path", directPath).
		Msg("🔑 Baixando mídia via DownloadMediaWithPath com chaves de descriptografia")

	mediaBytes, err := client.DownloadMediaWithPath(ctx, directPath, fileEncSHA256, fileSHA256, mediaKey, int(fileLength), whatsmeow.MediaImage, "")
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("message_id", msgIDStr).
			Str("direct_path", directPath).
			Msg("❌ Erro ao baixar mídia via DownloadMediaWithPath - usando fallback")

		// Usar fallback se falhou
		return h.processHistoricalMediaFallback(ctx, msg, mediaData)
	}

	// Determinar extensão
	extension := h.storage.GetExtensionFromMimeType(mimeType, ".bin")
	if fileName != "" {
		if ext := filepath.Ext(fileName); ext != "" {
			extension = strings.TrimPrefix(ext, ".")
		}
	}

	// Fazer upload para MinIO usando o MediaDownloader existente
	if h.mediaDownloader == nil {
		return fmt.Errorf("MediaDownloader não disponível")
	}

	// Usar o método existente do MediaDownloader para upload
	reader := bytes.NewReader(mediaBytes)
	objectPath, err := h.mediaDownloader.minioClient.UploadMedia(ctx, reader, storage.MediaUploadOptions{
		SessionID:   msg.SessionID,
		ChatJID:     msg.ChatJID,
		Direction:   string(msg.Direction),
		MessageID:   msgIDStr,
		ContentType: mimeType,
		Extension:   extension,
		Size:        int64(len(mediaBytes)),
	})

	if err != nil {
		return fmt.Errorf("erro ao fazer upload para MinIO: %w", err)
	}

	// Atualizar mensagem com path da mídia
	if err := h.storage.UpdateMessageWithMediaPath(ctx, msgIDStr, objectPath); err != nil {
		h.logger.Error().Err(err).
			Str("message_id", msgIDStr).
			Str("object_path", objectPath).
			Msg("Erro ao atualizar mensagem com path da mídia")
		// Não retorna erro para não impedir o processamento
	}

	h.logger.Info().
		Str("message_id", msgIDStr).
		Str("object_path", objectPath).
		Int("size_bytes", len(mediaBytes)).
		Msg("✅ Mídia histórica baixada via DownloadMediaWithPath e armazenada com sucesso")

	return nil
}

// extractMediaMetadata extrai metadados necessários para download via whatsmeow
func (h *StorageHandler) extractMediaMetadata(mediaData map[string]any) (directPath string, mediaKey []byte, fileEncSHA256 []byte, fileSHA256 []byte, fileLength uint64, mimeType string, fileName string) {
	// Procurar em diferentes tipos de mensagem de mídia
	mediaTypes := []string{"audioMessage", "imageMessage", "videoMessage", "documentMessage", "stickerMessage"}

	for _, mediaType := range mediaTypes {
		if mediaObj, exists := mediaData[mediaType]; exists {
			if mediaMap, ok := mediaObj.(map[string]any); ok {
				// Extrair directPath
				if dp, exists := mediaMap["directPath"]; exists {
					if dpStr, ok := dp.(string); ok {
						directPath = dpStr
					}
				}

				// Extrair mediaKey (base64 encoded)
				if mk, exists := mediaMap["mediaKey"]; exists {
					if mkStr, ok := mk.(string); ok {
						if decoded, err := base64.StdEncoding.DecodeString(mkStr); err == nil {
							mediaKey = decoded
						}
					}
				}

				// Extrair fileEncSha256 (base64 encoded)
				if fes, exists := mediaMap["fileEncSha256"]; exists {
					if fesStr, ok := fes.(string); ok {
						if decoded, err := base64.StdEncoding.DecodeString(fesStr); err == nil {
							fileEncSHA256 = decoded
						}
					}
				}

				// Extrair fileSha256 (base64 encoded)
				if fs, exists := mediaMap["fileSha256"]; exists {
					if fsStr, ok := fs.(string); ok {
						if decoded, err := base64.StdEncoding.DecodeString(fsStr); err == nil {
							fileSHA256 = decoded
						}
					}
				}

				// Extrair fileLength
				if fl, exists := mediaMap["fileLength"]; exists {
					switch v := fl.(type) {
					case uint64:
						fileLength = v
					case int64:
						fileLength = uint64(v)
					case int:
						fileLength = uint64(v)
					case float64:
						fileLength = uint64(v)
					case string:
						if parsed, err := strconv.ParseUint(v, 10, 64); err == nil {
							fileLength = parsed
						}
					}
				}

				// Extrair mimeType
				if mt, exists := mediaMap["mimetype"]; exists {
					if mtStr, ok := mt.(string); ok {
						mimeType = mtStr
					}
				}

				// Extrair fileName
				if fn, exists := mediaMap["fileName"]; exists {
					if fnStr, ok := fn.(string); ok {
						fileName = fnStr
					}
				}

				// Se encontrou directPath e mediaKey, temos o suficiente
				if directPath != "" && len(mediaKey) > 0 {
					break
				}
			}
		}
	}

	return directPath, mediaKey, fileEncSHA256, fileSHA256, fileLength, mimeType, fileName
}

// getWhatsmeowClient obtém o cliente whatsmeow para uma sessão específica
func (h *StorageHandler) getWhatsmeowClient(_ uuid.UUID) (*whatsmeow.Client, error) {
	// Aqui precisamos acessar o cliente whatsmeow da sessão
	// Por enquanto, vamos retornar um erro indicando que não está implementado
	// TODO: Implementar acesso ao cliente whatsmeow da sessão
	return nil, fmt.Errorf("acesso ao cliente whatsmeow não implementado ainda")
}

// processHistoricalMediaFallback processa mídia histórica usando método de fallback (URL direta)
func (h *StorageHandler) processHistoricalMediaFallback(ctx context.Context, msg *message.Message, mediaData map[string]any) error {
	msgIDStr := msg.ID.String()

	// Extrair URL da mídia e metadados
	var mediaURL, mimeType, fileName string

	// Verificar URL nos dados de mídia (tanto maiúsculo quanto minúsculo)
	urlKeys := []string{"url", "URL"}
	for _, key := range urlKeys {
		if url, exists := mediaData[key]; exists {
			if urlStr, ok := url.(string); ok {
				mediaURL = urlStr
				break
			}
		}
	}

	// Se não encontrou, procurar em diferentes tipos de mensagem de mídia (fallback)
	if mediaURL == "" {
		mediaTypes := []string{"audioMessage", "imageMessage", "videoMessage", "documentMessage", "stickerMessage"}

		for _, mediaType := range mediaTypes {
			if mediaObj, exists := mediaData[mediaType]; exists {
				if mediaMap, ok := mediaObj.(map[string]any); ok {
					for _, key := range urlKeys {
						if url, exists := mediaMap[key]; exists {
							if urlStr, ok := url.(string); ok {
								mediaURL = urlStr
								break
							}
						}
					}
					// Extrair metadados do tipo específico
					if mt, exists := mediaMap["mimetype"]; exists {
						if mtStr, ok := mt.(string); ok {
							mimeType = mtStr
						}
					}
					if fn, exists := mediaMap["fileName"]; exists {
						if fnStr, ok := fn.(string); ok {
							fileName = fnStr
						}
					}
					if mediaURL != "" {
						break
					}
				}
			}
		}
	}

	if mediaURL == "" {
		return fmt.Errorf("URL de mídia não encontrada")
	}

	// Extrair metadados gerais se não foram encontrados nos tipos específicos
	if mimeType == "" {
		if mt, exists := mediaData["mimetype"]; exists {
			if mtStr, ok := mt.(string); ok {
				mimeType = mtStr
			}
		}
	}
	if fileName == "" {
		if fn, exists := mediaData["fileName"]; exists {
			if fnStr, ok := fn.(string); ok {
				fileName = fnStr
			}
		}
	}

	// Se não há nome de arquivo, usar o ID da mensagem
	if fileName == "" {
		fileName = msgIDStr
	}

	h.logger.Warn().
		Str("message_id", msgIDStr).
		Str("media_url", mediaURL).
		Str("mime_type", mimeType).
		Str("file_name", fileName).
		Msg("⚠️ Usando fallback: download direto de URL (mídia pode estar corrompida)")

	// Baixar mídia da URL
	mediaBytes, err := h.storage.DownloadMediaFromURL(ctx, mediaURL)
	if err != nil {
		return fmt.Errorf("erro ao baixar mídia: %w", err)
	}

	// Determinar extensão
	extension := h.storage.GetExtensionFromMimeType(mimeType, ".bin")
	if fileName != "" {
		if ext := filepath.Ext(fileName); ext != "" {
			extension = strings.TrimPrefix(ext, ".")
		}
	}

	// Fazer upload para MinIO usando o MediaDownloader existente
	if h.mediaDownloader == nil {
		return fmt.Errorf("MediaDownloader não disponível")
	}

	// Usar o método existente do MediaDownloader para upload
	reader := bytes.NewReader(mediaBytes)
	objectPath, err := h.mediaDownloader.minioClient.UploadMedia(ctx, reader, storage.MediaUploadOptions{
		SessionID:   msg.SessionID,
		ChatJID:     msg.ChatJID,
		Direction:   string(msg.Direction),
		MessageID:   msgIDStr,
		ContentType: mimeType,
		Extension:   extension,
		Size:        int64(len(mediaBytes)),
	})
	if err != nil {
		return fmt.Errorf("erro ao fazer upload da mídia histórica: %w", err)
	}

	// Atualizar mensagem com path da mídia
	if err := h.storage.UpdateMessageWithMediaPath(ctx, msgIDStr, objectPath); err != nil {
		h.logger.Error().Err(err).
			Str("message_id", msgIDStr).
			Str("object_path", objectPath).
			Msg("Erro ao atualizar mensagem com path da mídia")
		// Não retorna erro para não impedir o processamento
	}

	h.logger.Warn().
		Str("message_id", msgIDStr).
		Str("object_path", objectPath).
		Int("size_bytes", len(mediaBytes)).
		Msg("⚠️ Mídia histórica processada via fallback (pode estar corrompida)")

	return nil
}

// extractMessageContent extrai o conteúdo e tipo da mensagem baseado no payload
func (h *StorageHandler) extractMessageContent(messageField map[string]any) (message.MessageType, string, map[string]any) {
	msgContent, ok := messageField["message"].(map[string]any)
	if !ok {
		return message.MessageTypeText, "[Mensagem do histórico]", nil
	}

	mediaData := make(map[string]any)

	// Processar diferentes tipos de mensagem
	if conversation, ok := msgContent["conversation"].(string); ok {
		return message.MessageTypeText, conversation, nil
	}

	if extendedText, ok := msgContent["extendedTextMessage"].(map[string]any); ok {
		text, _ := extendedText["text"].(string)
		if matchedText, exists := extendedText["matchedText"].(string); exists {
			mediaData["matched_text"] = matchedText
		}
		if description, exists := extendedText["description"].(string); exists {
			mediaData["description"] = description
		}
		if title, exists := extendedText["title"].(string); exists {
			mediaData["title"] = title
		}
		return message.MessageTypeText, text, mediaData
	}

	if imageMsg, ok := msgContent["imageMessage"].(map[string]any); ok {
		caption, _ := imageMsg["caption"].(string)
		if caption == "" {
			caption = "[Imagem]"
		}

		// Extrair dados da imagem
		if url, exists := imageMsg["URL"].(string); exists {
			mediaData["url"] = url
		}
		if mimetype, exists := imageMsg["mimetype"].(string); exists {
			mediaData["mimetype"] = mimetype
		}
		if width, exists := imageMsg["width"].(float64); exists {
			mediaData["width"] = int(width)
		}
		if height, exists := imageMsg["height"].(float64); exists {
			mediaData["height"] = int(height)
		}
		if fileLength, exists := imageMsg["fileLength"].(float64); exists {
			mediaData["file_length"] = int(fileLength)
		}

		return message.MessageTypeImage, caption, mediaData
	}

	if videoMsg, ok := msgContent["videoMessage"].(map[string]any); ok {
		caption, _ := videoMsg["caption"].(string)
		if caption == "" {
			caption = "[Vídeo]"
		}

		// Extrair dados do vídeo
		if url, exists := videoMsg["URL"].(string); exists {
			mediaData["url"] = url
		}
		if mimetype, exists := videoMsg["mimetype"].(string); exists {
			mediaData["mimetype"] = mimetype
		}
		if seconds, exists := videoMsg["seconds"].(float64); exists {
			mediaData["duration"] = int(seconds)
		}
		if width, exists := videoMsg["width"].(float64); exists {
			mediaData["width"] = int(width)
		}
		if height, exists := videoMsg["height"].(float64); exists {
			mediaData["height"] = int(height)
		}

		return message.MessageTypeVideo, caption, mediaData
	}

	if audioMsg, ok := msgContent["audioMessage"].(map[string]any); ok {
		content := "[Áudio]"
		if ptt, exists := audioMsg["PTT"].(bool); exists && ptt {
			content = "[Áudio de voz]"
			mediaData["is_ptt"] = true
		}

		// Extrair dados do áudio
		if url, exists := audioMsg["URL"].(string); exists {
			mediaData["url"] = url
		}
		if mimetype, exists := audioMsg["mimetype"].(string); exists {
			mediaData["mimetype"] = mimetype
		}
		if seconds, exists := audioMsg["seconds"].(float64); exists {
			mediaData["duration"] = int(seconds)
		}

		return message.MessageTypeAudio, content, mediaData
	}

	if docMsg, ok := msgContent["documentMessage"].(map[string]any); ok {
		fileName, _ := docMsg["fileName"].(string)
		if fileName == "" {
			fileName = "Documento"
		}

		// Extrair dados do documento
		if url, exists := docMsg["URL"].(string); exists {
			mediaData["url"] = url
		}
		if mimetype, exists := docMsg["mimetype"].(string); exists {
			mediaData["mimetype"] = mimetype
		}
		if fileLength, exists := docMsg["fileLength"].(float64); exists {
			mediaData["file_length"] = int(fileLength)
		}
		if pageCount, exists := docMsg["pageCount"].(float64); exists {
			mediaData["page_count"] = int(pageCount)
		}
		if title, exists := docMsg["title"].(string); exists {
			mediaData["title"] = title
		}

		return message.MessageTypeDocument, fmt.Sprintf("[Documento: %s]", fileName), mediaData
	}

	if stickerMsg, ok := msgContent["stickerMessage"].(map[string]any); ok {
		// Extrair dados do sticker
		if url, exists := stickerMsg["URL"].(string); exists {
			mediaData["url"] = url
		}
		if mimetype, exists := stickerMsg["mimetype"].(string); exists {
			mediaData["mimetype"] = mimetype
		}
		if width, exists := stickerMsg["width"].(float64); exists {
			mediaData["width"] = int(width)
		}
		if height, exists := stickerMsg["height"].(float64); exists {
			mediaData["height"] = int(height)
		}

		return message.MessageTypeSticker, "[Sticker]", mediaData
	}

	// Continuar com outros tipos...
	return h.extractAdvancedMessageTypes(msgContent, mediaData)
}

// extractAdvancedMessageTypes processa tipos avançados de mensagens
func (h *StorageHandler) extractAdvancedMessageTypes(msgContent map[string]any, mediaData map[string]any) (message.MessageType, string, map[string]any) {

	if contactMsg, ok := msgContent["contactMessage"].(map[string]any); ok {
		displayName, _ := contactMsg["displayName"].(string)
		if displayName == "" {
			displayName = "Contato"
		}

		// Extrair dados do contato
		if vcard, exists := contactMsg["vcard"].(string); exists {
			mediaData["vcard"] = vcard
		}

		return message.MessageTypeContact, fmt.Sprintf("[Contato: %s]", displayName), mediaData
	}

	if locationMsg, ok := msgContent["locationMessage"].(map[string]any); ok {
		name, _ := locationMsg["name"].(string)
		if name == "" {
			name = "Localização"
		}

		// Extrair coordenadas
		if lat, exists := locationMsg["degreesLatitude"].(float64); exists {
			mediaData["latitude"] = lat
		}
		if lng, exists := locationMsg["degreesLongitude"].(float64); exists {
			mediaData["longitude"] = lng
		}
		if address, exists := locationMsg["address"].(string); exists {
			mediaData["address"] = address
		}

		return message.MessageTypeLocation, fmt.Sprintf("[Localização: %s]", name), mediaData
	}

	if listMsg, ok := msgContent["listMessage"].(map[string]any); ok {
		title, _ := listMsg["title"].(string)
		description, _ := listMsg["description"].(string)
		buttonText, _ := listMsg["buttonText"].(string)

		content := fmt.Sprintf("[Lista: %s]", title)
		if description != "" {
			content = fmt.Sprintf("[Lista: %s - %s]", title, description)
		}

		// Extrair dados da lista
		mediaData["title"] = title
		mediaData["description"] = description
		mediaData["button_text"] = buttonText
		if listType, exists := listMsg["listType"].(float64); exists {
			mediaData["list_type"] = int(listType)
		}
		if sections, exists := listMsg["sections"].([]any); exists {
			mediaData["sections"] = sections
		}

		return message.MessageTypeInteractive, content, mediaData
	}

	if buttonsMsg, ok := msgContent["buttonsMessage"].(map[string]any); ok {
		contentText, _ := buttonsMsg["contentText"].(string)
		if contentText == "" {
			contentText = "Mensagem com botões"
		}

		// Extrair dados dos botões
		if header, exists := buttonsMsg["Header"]; exists {
			mediaData["header"] = header
		}
		if footer, exists := buttonsMsg["footerText"].(string); exists {
			mediaData["footer"] = footer
		}
		if buttons, exists := buttonsMsg["buttons"].([]any); exists {
			mediaData["buttons"] = buttons
		}

		return message.MessageTypeInteractive, fmt.Sprintf("[Botões: %s]", contentText), mediaData
	}

	if templateMsg, ok := msgContent["templateMessage"].(map[string]any); ok {
		content := "[Template]"

		// Extrair dados do template
		if format, exists := templateMsg["Format"].(map[string]any); exists {
			mediaData["format"] = format

			if hydratedTemplate, exists := format["HydratedFourRowTemplate"].(map[string]any); exists {
				if hydratedContent, exists := hydratedTemplate["hydratedContentText"].(string); exists {
					content = fmt.Sprintf("[Template: %s]", hydratedContent)
				}
			}
		}

		return message.MessageTypeInteractive, content, mediaData
	}

	// Tipo não reconhecido - retornar como texto genérico
	return message.MessageTypeText, "[Mensagem do histórico - tipo não reconhecido]", mediaData
}
