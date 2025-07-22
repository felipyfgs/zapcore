package whatsapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"zapcore/internal/domain/chat"
	"zapcore/internal/domain/contact"
	"zapcore/internal/domain/message"
	"zapcore/internal/infra/storage"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// StorageOperations contém todas as operações de persistência e storage
type StorageOperations struct {
	storage *StorageHandler
}

// NewStorageOperations cria nova instância das operações de storage
func NewStorageOperations(storage *StorageHandler) *StorageOperations {
	return &StorageOperations{
		storage: storage,
	}
}

// ProcessMessageContent processa o conteúdo da mensagem baseado no tipo
func (so *StorageOperations) ProcessMessageContent(msg *message.Message, msgContent *waE2E.Message) error {
	switch {
	case msgContent.Conversation != nil:
		msg.MessageType = message.MessageTypeText
		msg.Content = *msgContent.Conversation

	case msgContent.ExtendedTextMessage != nil:
		msg.MessageType = message.MessageTypeText
		if msgContent.ExtendedTextMessage.Text != nil {
			msg.Content = *msgContent.ExtendedTextMessage.Text
		}

	case msgContent.ImageMessage != nil:
		msg.MessageType = message.MessageTypeImage
		if msgContent.ImageMessage.Caption != nil {
			msg.Content = *msgContent.ImageMessage.Caption
		}

	case msgContent.VideoMessage != nil:
		msg.MessageType = message.MessageTypeVideo
		if msgContent.VideoMessage.Caption != nil {
			msg.Content = *msgContent.VideoMessage.Caption
		}

	case msgContent.AudioMessage != nil:
		msg.MessageType = message.MessageTypeAudio
		msg.Content = "[Áudio]"

	case msgContent.DocumentMessage != nil:
		msg.MessageType = message.MessageTypeDocument
		if msgContent.DocumentMessage.FileName != nil {
			msg.Content = fmt.Sprintf("[Documento: %s]", *msgContent.DocumentMessage.FileName)
		} else {
			msg.Content = "[Documento]"
		}

	case msgContent.StickerMessage != nil:
		msg.MessageType = message.MessageTypeSticker
		msg.Content = "[Sticker]"

	case msgContent.LocationMessage != nil:
		msg.MessageType = message.MessageTypeLocation
		if msgContent.LocationMessage.Name != nil {
			msg.Content = fmt.Sprintf("[Localização: %s]", *msgContent.LocationMessage.Name)
		} else {
			msg.Content = "[Localização]"
		}

	case msgContent.ContactMessage != nil:
		msg.MessageType = message.MessageTypeContact
		if msgContent.ContactMessage.DisplayName != nil {
			msg.Content = fmt.Sprintf("[Contato: %s]", *msgContent.ContactMessage.DisplayName)
		} else {
			msg.Content = "[Contato]"
		}

	case msgContent.ContactsArrayMessage != nil:
		msg.MessageType = message.MessageTypeContact
		contactCount := len(msgContent.ContactsArrayMessage.Contacts)
		msg.Content = fmt.Sprintf("[%d Contatos]", contactCount)

	case msgContent.LiveLocationMessage != nil:
		msg.MessageType = message.MessageTypeLocation
		msg.Content = "[Localização ao Vivo]"

	case msgContent.GroupInviteMessage != nil:
		msg.MessageType = message.MessageTypeText
		if msgContent.GroupInviteMessage.GroupName != nil {
			msg.Content = fmt.Sprintf("[Convite para grupo: %s]", *msgContent.GroupInviteMessage.GroupName)
		} else {
			msg.Content = "[Convite para grupo]"
		}

	case msgContent.ButtonsMessage != nil:
		msg.MessageType = message.MessageTypeText
		if msgContent.ButtonsMessage.ContentText != nil {
			msg.Content = *msgContent.ButtonsMessage.ContentText
		} else {
			msg.Content = "[Mensagem com botões]"
		}

	case msgContent.ListMessage != nil:
		msg.MessageType = message.MessageTypeText
		if msgContent.ListMessage.Description != nil {
			msg.Content = *msgContent.ListMessage.Description
		} else {
			msg.Content = "[Lista]"
		}

	case msgContent.TemplateMessage != nil:
		msg.MessageType = message.MessageTypeText
		msg.Content = "[Mensagem de template]"

	case msgContent.HighlyStructuredMessage != nil:
		msg.MessageType = message.MessageTypeText
		msg.Content = "[Mensagem estruturada]"

	case msgContent.SendPaymentMessage != nil:
		msg.MessageType = message.MessageTypeText
		msg.Content = "[Pagamento]"

	case msgContent.RequestPaymentMessage != nil:
		msg.MessageType = message.MessageTypeText
		msg.Content = "[Solicitação de pagamento]"

	case msgContent.DeclinePaymentRequestMessage != nil:
		msg.MessageType = message.MessageTypeText
		msg.Content = "[Recusa de pagamento]"

	case msgContent.CancelPaymentRequestMessage != nil:
		msg.MessageType = message.MessageTypeText
		msg.Content = "[Cancelamento de pagamento]"

	case msgContent.Call != nil:
		msg.MessageType = message.MessageTypeText
		msg.Content = "[Chamada]"

	case msgContent.Chat != nil:
		msg.MessageType = message.MessageTypeText
		msg.Content = "[Chat]"

	case msgContent.ProtocolMessage != nil:
		msg.MessageType = message.MessageTypeText
		if msgContent.ProtocolMessage.Type != nil {
			switch *msgContent.ProtocolMessage.Type {
			case waE2E.ProtocolMessage_REVOKE:
				msg.Content = "[Mensagem apagada]"
			case waE2E.ProtocolMessage_EPHEMERAL_SETTING:
				msg.Content = "[Configuração de mensagem temporária]"
			default:
				msg.Content = "[Mensagem de protocolo]"
			}
		} else {
			msg.Content = "[Mensagem de protocolo]"
		}

	default:
		msg.MessageType = message.MessageTypeText
		msg.Content = "[Tipo de mensagem não suportado]"
	}

	return nil
}

// ProcessMediaMessage processa mídia da mensagem fazendo download e upload para MinIO
func (so *StorageOperations) ProcessMediaMessage(ctx context.Context, msg *message.Message, evt *events.Message) error {
	// Fazer download e upload da mídia
	mediaInfo, err := so.storage.mediaDownloader.DownloadAndUploadMedia(ctx, evt, msg.SessionID)
	if err != nil {
		return fmt.Errorf("erro ao processar mídia: %w", err)
	}

	// Atualizar mensagem com informações da mídia
	if mediaInfo != nil {
		// Note: MediaInfo fields may vary - adjust according to actual struct
		msg.MediaMimeType = mediaInfo.MimeType
		msg.MediaSize = mediaInfo.Size
	}

	return nil
}

// UpdateChatFromMessage atualiza informações do chat baseado na mensagem
func (so *StorageOperations) UpdateChatFromMessage(ctx context.Context, sessionID uuid.UUID, evt *events.Message) error {
	chatJID := evt.Info.Chat.String()

	// Buscar chat existente
	existingChat, err := so.storage.chatRepo.GetByJID(ctx, sessionID, chatJID)
	if err != nil && err != chat.ErrChatNotFound {
		return fmt.Errorf("erro ao buscar chat: %w", err)
	}

	if existingChat != nil {
		// Atualizar última mensagem e contadores
		now := time.Now()
		existingChat.LastMessageTime = &now
		existingChat.MessageCount++
		if !evt.Info.IsFromMe {
			existingChat.UnreadCount++
		}
		existingChat.UpdatedAt = now

		return so.storage.chatRepo.Update(ctx, existingChat)
	} else {
		// Criar novo chat
		chatType := chat.ChatTypeIndividual
		if evt.Info.Chat.Server == "g.us" {
			chatType = chat.ChatTypeGroup
		}

		chatEntity := chat.NewChat(sessionID, chatJID, chatType)
		now := time.Now()
		chatEntity.LastMessageTime = &now
		chatEntity.MessageCount = 1
		if !evt.Info.IsFromMe {
			chatEntity.UnreadCount = 1
		}

		return so.storage.chatRepo.Create(ctx, chatEntity)
	}
}

// UpdateContactFromMessage atualiza informações do contato baseado na mensagem
func (so *StorageOperations) UpdateContactFromMessage(ctx context.Context, sessionID uuid.UUID, evt *events.Message) error {
	if evt.Info.IsFromMe {
		return nil // Não processar mensagens próprias
	}

	contactJID := evt.Info.Sender.String()

	// Buscar contato existente
	existingContact, err := so.storage.contactRepo.GetByJID(ctx, sessionID, contactJID)
	if err != nil && err != contact.ErrContactNotFound {
		return fmt.Errorf("erro ao buscar contato: %w", err)
	}

	if existingContact != nil {
		// Atualizar nome se disponível
		updated := false
		if evt.Info.PushName != "" && existingContact.PushName != evt.Info.PushName {
			existingContact.PushName = evt.Info.PushName
			updated = true
		}

		if updated {
			existingContact.UpdatedAt = time.Now()
			return so.storage.contactRepo.Update(ctx, existingContact)
		}
	} else {
		// Criar novo contato
		contactEntity := contact.NewContact(sessionID, contactJID)
		contactEntity.PushName = evt.Info.PushName

		return so.storage.contactRepo.Create(ctx, contactEntity)
	}

	return nil
}

// StoreRawPayload armazena o payload bruto do whatsmeow na mensagem
func (so *StorageOperations) StoreRawPayload(msg *message.Message, evt *events.Message) error {
	// Converter o evento completo para map[string]any
	rawPayload := map[string]any{
		"info": map[string]any{
			"id":         evt.Info.ID,
			"timestamp":  evt.Info.Timestamp,
			"chat":       evt.Info.Chat.String(),
			"sender":     evt.Info.Sender.String(),
			"is_from_me": evt.Info.IsFromMe,
			"push_name":  evt.Info.PushName,
		},
		"message": evt.Message,
	}

	// Armazenar no campo RawPayload da mensagem
	msg.RawPayload = rawPayload

	return nil
}

// CreateHistoricalMessage cria um registro básico de mensagem histórica
func (so *StorageOperations) CreateHistoricalMessage(ctx context.Context, msgID string, sessionID uuid.UUID, chatJID, senderJID string, status message.MessageStatus) error {
	// Criar entidade de mensagem básica
	msg := message.NewMessage(sessionID, message.MessageTypeText, message.MessageDirectionInbound)
	msg.MsgID = msgID
	msg.ChatJID = chatJID
	msg.SenderJID = senderJID
	msg.Content = "[Mensagem do histórico]"
	msg.Status = status

	return so.storage.messageRepo.Create(ctx, msg)
}

// ProcessHistoricalMedia processa mídia de mensagens históricas usando DownloadMediaWithPath do whatsmeow
func (so *StorageOperations) ProcessHistoricalMedia(ctx context.Context, msg *message.Message, mediaData map[string]any) error {
	msgIDStr := msg.ID.String()

	// Verificar se a mídia já foi processada para esta mensagem
	if so.IsMediaAlreadyProcessed(msgIDStr) {
		so.storage.logger.Debug().
			Str("message_id", msgIDStr).
			Msg("🔄 Mídia já processada, pulando")
		return nil
	}

	so.storage.logger.Info().
		Str("message_id", msgIDStr).
		Interface("media_data_keys", so.getMapKeys(mediaData)).
		Msg("🚀 INICIANDO: processHistoricalMedia")

	// Verificar se há URL de mídia válida
	if !so.HasMediaURL(mediaData) {
		so.storage.logger.Warn().
			Str("message_id", msgIDStr).
			Msg("⚠️ Nenhuma URL de mídia válida encontrada")
		return nil
	}

	// Extrair metadados necessários para download via whatsmeow
	directPath, mediaKey, fileEncSHA256, fileSHA256, fileLength, mimeType, _ := so.ExtractMediaMetadata(mediaData)

	if directPath == "" || len(mediaKey) == 0 {
		so.storage.logger.Warn().
			Str("message_id", msgIDStr).
			Str("direct_path", directPath).
			Int("media_key_len", len(mediaKey)).
			Msg("⚠️ Metadados de mídia insuficientes para download via whatsmeow")

		// Tentar método de fallback
		return so.ProcessHistoricalMediaFallback(ctx, msg, mediaData)
	}

	so.storage.logger.Info().
		Str("message_id", msgIDStr).
		Str("direct_path", directPath).
		Str("mime_type", mimeType).
		Uint64("file_length", fileLength).
		Msg("📋 Metadados extraídos com sucesso")

	// Obter cliente whatsmeow para download
	client, err := so.GetWhatsmeowClient(msg.SessionID)
	if err != nil {
		so.storage.logger.Error().
			Err(err).
			Str("message_id", msgIDStr).
			Msg("❌ Erro ao obter cliente whatsmeow")
		return so.ProcessHistoricalMediaFallback(ctx, msg, mediaData)
	}

	// Fazer download usando whatsmeow
	so.storage.logger.Info().
		Str("message_id", msgIDStr).
		Msg("⬇️ Iniciando download via whatsmeow")

	mediaBytes, err := client.DownloadMediaWithPath(ctx, directPath, fileEncSHA256, fileSHA256, mediaKey, int(fileLength), whatsmeow.MediaImage, "")
	if err != nil {
		so.storage.logger.Error().
			Err(err).
			Str("message_id", msgIDStr).
			Msg("❌ Erro no download via whatsmeow, tentando fallback")
		return so.ProcessHistoricalMediaFallback(ctx, msg, mediaData)
	}

	so.storage.logger.Info().
		Str("message_id", msgIDStr).
		Int("bytes_downloaded", len(mediaBytes)).
		Msg("✅ Download via whatsmeow concluído")

	// Fazer upload para MinIO
	uploadOpts := storage.MediaUploadOptions{
		SessionID:   msg.SessionID,
		ChatJID:     msg.ChatJID,
		Direction:   string(msg.Direction),
		MessageID:   msg.MsgID,
		ContentType: mimeType,
	}

	objectPath, err := so.storage.mediaDownloader.uploadToMinIO(ctx, mediaBytes, uploadOpts)
	if err != nil {
		so.storage.logger.Error().
			Err(err).
			Str("message_id", msgIDStr).
			Str("object_path", objectPath).
			Msg("❌ Erro ao fazer upload para MinIO")
		return fmt.Errorf("erro ao fazer upload para MinIO: %w", err)
	}

	so.storage.logger.Info().
		Str("message_id", msgIDStr).
		Str("object_path", objectPath).
		Msg("✅ Upload para MinIO concluído")

	// Atualizar mensagem com path da mídia
	if err := so.UpdateMessageWithMediaPath(ctx, msgIDStr, objectPath); err != nil {
		so.storage.logger.Error().
			Err(err).
			Str("message_id", msgIDStr).
			Msg("❌ Erro ao atualizar mensagem com path da mídia")
	}

	return nil
}

// ProcessHistoricalMediaFallback processa mídia histórica usando método de fallback (URL direta)
func (so *StorageOperations) ProcessHistoricalMediaFallback(ctx context.Context, msg *message.Message, mediaData map[string]any) error {
	msgIDStr := msg.ID.String()

	// Extrair URL da mídia e metadados
	var mediaURL, mimeType string

	// Procurar em diferentes tipos de mensagem de mídia
	mediaTypes := []string{"audioMessage", "imageMessage", "videoMessage", "documentMessage", "stickerMessage"}

	for _, mediaType := range mediaTypes {
		if mediaObj, exists := mediaData[mediaType]; exists {
			if mediaMap, ok := mediaObj.(map[string]any); ok {
				if url, exists := mediaMap["url"]; exists {
					if urlStr, ok := url.(string); ok && urlStr != "" {
						mediaURL = urlStr
					}
				}
				if mime, exists := mediaMap["mimetype"]; exists {
					if mimeStr, ok := mime.(string); ok {
						mimeType = mimeStr
					}
				}

				break
			}
		}
	}

	if mediaURL == "" {
		so.storage.logger.Warn().
			Str("message_id", msgIDStr).
			Msg("⚠️ URL de mídia não encontrada no fallback")
		return nil
	}

	so.storage.logger.Info().
		Str("message_id", msgIDStr).
		Str("media_url", mediaURL).
		Str("mime_type", mimeType).
		Msg("🔄 Processando mídia via fallback (URL direta)")

	// Fazer download da mídia
	mediaBytes, err := so.DownloadMediaFromURL(ctx, mediaURL)
	if err != nil {
		so.storage.logger.Error().
			Err(err).
			Str("message_id", msgIDStr).
			Str("media_url", mediaURL).
			Msg("❌ Erro no download via URL direta")
		return fmt.Errorf("erro ao fazer download da mídia: %w", err)
	}

	so.storage.logger.Info().
		Str("message_id", msgIDStr).
		Int("bytes_downloaded", len(mediaBytes)).
		Msg("✅ Download via URL direta concluído")

	// Fazer upload para MinIO
	uploadOpts := storage.MediaUploadOptions{
		SessionID:   msg.SessionID,
		ChatJID:     msg.ChatJID,
		Direction:   string(msg.Direction),
		MessageID:   msg.MsgID,
		ContentType: mimeType,
	}

	objectPath, err := so.storage.mediaDownloader.uploadToMinIO(ctx, mediaBytes, uploadOpts)
	if err != nil {
		so.storage.logger.Error().
			Err(err).
			Str("message_id", msgIDStr).
			Str("object_path", objectPath).
			Msg("❌ Erro ao fazer upload para MinIO")
		return fmt.Errorf("erro ao fazer upload para MinIO: %w", err)
	}

	so.storage.logger.Info().
		Str("message_id", msgIDStr).
		Str("object_path", objectPath).
		Msg("✅ Upload para MinIO concluído via fallback")

	// Atualizar mensagem com path da mídia
	if err := so.UpdateMessageWithMediaPath(ctx, msgIDStr, objectPath); err != nil {
		so.storage.logger.Error().
			Err(err).
			Str("message_id", msgIDStr).
			Msg("❌ Erro ao atualizar mensagem com path da mídia")
	}

	return nil
}

// IsMediaAlreadyProcessed verifica se a mídia já foi processada para esta mensagem
func (so *StorageOperations) IsMediaAlreadyProcessed(_ string) bool {
	// Verificar se a mensagem já tem um path de mídia no banco
	// Por simplicidade, vamos assumir que se chegou até aqui, ainda não foi processada
	// Em uma implementação mais robusta, poderia consultar o banco para verificar se já existe um path de mídia
	return false
}

// getMapKeys obtém as chaves de um map para logging (otimizado para performance)
func (so *StorageOperations) getMapKeys(data map[string]any) []string {
	if len(data) == 0 {
		return nil
	}
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	return keys
}

// HasMediaURL verifica se os dados de mídia contêm uma URL válida
func (so *StorageOperations) HasMediaURL(mediaData map[string]any) bool {
	so.storage.logger.Debug().
		Interface("media_data_keys", so.getMapKeys(mediaData)).
		Msg("🔍 DEBUG: Iniciando verificação de URL de mídia")

	if mediaData == nil {
		so.storage.logger.Debug().Msg("🔍 DEBUG: mediaData é nil")
		return false
	}

	// Procurar em diferentes tipos de mensagem de mídia
	mediaTypes := []string{"audioMessage", "imageMessage", "videoMessage", "documentMessage", "stickerMessage"}

	for _, mediaType := range mediaTypes {
		so.storage.logger.Debug().
			Str("media_type", mediaType).
			Msg("🔍 DEBUG: Verificando tipo de mídia")

		if mediaObj, exists := mediaData[mediaType]; exists {
			so.storage.logger.Debug().
				Str("media_type", mediaType).
				Msg("🔍 DEBUG: Tipo de mídia encontrado")

			if mediaMap, ok := mediaObj.(map[string]any); ok {
				so.storage.logger.Debug().
					Str("media_type", mediaType).
					Interface("media_keys", so.getMapKeys(mediaMap)).
					Msg("🔍 DEBUG: Chaves do objeto de mídia")

				if url, exists := mediaMap["url"]; exists {
					if urlStr, ok := url.(string); ok && urlStr != "" {
						so.storage.logger.Debug().
							Str("media_type", mediaType).
							Str("url", urlStr).
							Msg("🔍 DEBUG: URL válida encontrada")
						return true
					}
				}
			}
		}
	}

	so.storage.logger.Debug().Msg("🔍 DEBUG: Nenhuma URL válida encontrada")
	return false
}

// ExtractMediaMetadata extrai metadados necessários para download via whatsmeow
func (so *StorageOperations) ExtractMediaMetadata(mediaData map[string]any) (directPath string, mediaKey []byte, fileEncSHA256 []byte, fileSHA256 []byte, fileLength uint64, mimeType string, fileName string) {
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

				// Extrair mediaKey (base64)
				if mk, exists := mediaMap["mediaKey"]; exists {
					if mkStr, ok := mk.(string); ok {
						// Decodificar base64
						if decoded, err := base64.StdEncoding.DecodeString(mkStr); err == nil {
							mediaKey = decoded
						}
					}
				}

				// Extrair fileEncSha256 (base64)
				if fes, exists := mediaMap["fileEncSha256"]; exists {
					if fesStr, ok := fes.(string); ok {
						if decoded, err := base64.StdEncoding.DecodeString(fesStr); err == nil {
							fileEncSHA256 = decoded
						}
					}
				}

				// Extrair fileSha256 (base64)
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
					case string:
						if parsed, err := strconv.ParseUint(v, 10, 64); err == nil {
							fileLength = parsed
						}
					case float64:
						fileLength = uint64(v)
					case int:
						fileLength = uint64(v)
					case int64:
						fileLength = uint64(v)
					}
				}

				// Extrair mimeType
				if mt, exists := mediaMap["mimetype"]; exists {
					if mtStr, ok := mt.(string); ok {
						mimeType = mtStr
					}
				}

				// Extrair fileName (apenas para documentos)
				if fn, exists := mediaMap["fileName"]; exists {
					if fnStr, ok := fn.(string); ok {
						fileName = fnStr
					}
				}

				break
			}
		}
	}

	return directPath, mediaKey, fileEncSHA256, fileSHA256, fileLength, mimeType, fileName
}

// GetWhatsmeowClient obtém o cliente whatsmeow para uma sessão específica
func (so *StorageOperations) GetWhatsmeowClient(_ uuid.UUID) (*whatsmeow.Client, error) {
	// Aqui precisamos acessar o cliente whatsmeow da sessão
	// Por enquanto, vamos retornar um erro indicando que não está implementado
	// TODO: Implementar acesso ao cliente whatsmeow da sessão
	return nil, fmt.Errorf("acesso ao cliente whatsmeow não implementado ainda")
}

// GetExtensionFromMimeType obtém a extensão do arquivo baseada no MIME type
func (so *StorageOperations) GetExtensionFromMimeType(mimeType, defaultExt string) string {
	if mimeType == "" {
		return strings.TrimPrefix(defaultExt, ".")
	}

	exts, err := mime.ExtensionsByType(mimeType)
	if err != nil || len(exts) == 0 {
		return strings.TrimPrefix(defaultExt, ".")
	}

	// Retornar a primeira extensão sem o ponto
	return strings.TrimPrefix(exts[0], ".")
}

// UpdateMessageWithMediaPath atualiza a mensagem com o path da mídia no MinIO
func (so *StorageOperations) UpdateMessageWithMediaPath(_ context.Context, messageID, objectPath string) error {
	// Por enquanto, apenas log da operação
	// Em uma implementação completa, atualizaria o registro no banco de dados
	so.storage.logger.Debug().
		Str("message_id", messageID).
		Str("object_path", objectPath).
		Msg("Atualizando mensagem com path da mídia")
	return nil
}

// DownloadMediaFromURL baixa mídia de uma URL do WhatsApp
func (so *StorageOperations) DownloadMediaFromURL(ctx context.Context, mediaURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", mediaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %w", err)
	}

	return data, nil
}

// StoreUndecryptableRawPayload armazena o payload bruto do evento UndecryptableMessage
func (so *StorageOperations) StoreUndecryptableRawPayload(msg *message.Message, evt *events.UndecryptableMessage) error {
	// Converter o evento UndecryptableMessage para map[string]any
	rawPayload := map[string]any{
		"info": map[string]any{
			"id":         evt.Info.ID,
			"timestamp":  evt.Info.Timestamp,
			"chat":       evt.Info.Chat.String(),
			"sender":     evt.Info.Sender.String(),
			"is_from_me": evt.Info.IsFromMe,
			"push_name":  evt.Info.PushName,
		},
		"undecryptable_info": map[string]any{
			"is_unavailable":   evt.IsUnavailable,
			"unavailable_type": string(evt.UnavailableType),
		},
	}

	// Armazenar no campo RawPayload da mensagem
	msg.RawPayload = rawPayload

	return nil
}
