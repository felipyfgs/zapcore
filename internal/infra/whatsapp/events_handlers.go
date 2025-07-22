package whatsapp

import (
	"context"
	"fmt"
	"time"

	"zapcore/internal/domain/chat"
	"zapcore/internal/domain/contact"
	"zapcore/internal/domain/message"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/types/events"
)

// Constantes para padronização de logging
const (
	LogComponentHandlers = "handlers"
)

// EventHandlers contém todos os handlers específicos de eventos
type EventHandlers struct {
	storage *StorageHandler
}

// NewEventHandlers cria nova instância dos handlers de eventos
func NewEventHandlers(storage *StorageHandler) *EventHandlers {
	return &EventHandlers{
		storage: storage,
	}
}

// HandleMessage processa eventos de mensagem
func (eh *EventHandlers) HandleMessage(ctx context.Context, sessionID uuid.UUID, evt *events.Message) error {
	if evt.Message == nil {
		return nil
	}

	// Determinar direção da mensagem
	var direction message.MessageDirection
	if evt.Info.IsFromMe {
		direction = message.MessageDirectionOutbound
	} else {
		direction = message.MessageDirectionInbound
	}

	// Criar entidade de mensagem
	msg := message.NewMessage(sessionID, message.MessageTypeText, direction)
	msg.MsgID = evt.Info.ID
	msg.ChatJID = evt.Info.Chat.String()
	msg.SenderJID = evt.Info.Sender.String()
	msg.Timestamp = evt.Info.Timestamp

	// Processar conteúdo da mensagem
	if err := eh.storage.processMessageContent(msg, evt.Message); err != nil {
		eh.storage.logger.Error().Err(err).Msg("Erro ao processar conteúdo da mensagem")
		return err
	}

	// Armazenar payload bruto
	if err := eh.storage.storeRawPayload(msg, evt); err != nil {
		eh.storage.logger.Error().Err(err).Msg("Erro ao armazenar payload bruto")
	}

	// Salvar mensagem no banco
	if err := eh.storage.messageRepo.Create(ctx, msg); err != nil {
		eh.storage.logger.Error().Err(err).Msg("Erro ao salvar mensagem")
		return err
	}

	// Processar mídia se presente
	if msg.MessageType != message.MessageTypeText {
		if err := eh.storage.processMediaMessage(ctx, msg, evt); err != nil {
			eh.storage.logger.Error().Err(err).Msg("Erro ao processar mídia da mensagem")
		}
	}

	// Atualizar informações do chat
	if err := eh.storage.updateChatFromMessage(ctx, sessionID, evt); err != nil {
		eh.storage.logger.Error().Err(err).Msg("Erro ao atualizar chat")
	}

	// Atualizar informações do contato
	if err := eh.storage.updateContactFromMessage(ctx, sessionID, evt); err != nil {
		eh.storage.logger.Error().Err(err).Msg("Erro ao atualizar contato")
	}

	return nil
}

// HandleUndecryptableMessage processa eventos de mensagem não descriptografável
func (eh *EventHandlers) HandleUndecryptableMessage(ctx context.Context, sessionID uuid.UUID, evt *events.UndecryptableMessage) error {
	// Determinar direção da mensagem
	var direction message.MessageDirection
	if evt.Info.IsFromMe {
		direction = message.MessageDirectionOutbound
	} else {
		direction = message.MessageDirectionInbound
	}

	// Criar entidade de mensagem
	msg := message.NewMessage(sessionID, message.MessageTypeText, direction)
	msg.MsgID = evt.Info.ID
	msg.ChatJID = evt.Info.Chat.String()
	msg.SenderJID = evt.Info.Sender.String()
	msg.Timestamp = evt.Info.Timestamp
	msg.Content = "[Mensagem não descriptografável]"

	// Adicionar informações sobre o erro de descriptografia
	if evt.IsUnavailable {
		msg.Content = fmt.Sprintf("[Mensagem indisponível - Tipo: %s]", evt.UnavailableType)
	}

	// Armazenar payload bruto
	if err := eh.storage.storeUndecryptableRawPayload(msg, evt); err != nil {
		eh.storage.logger.Error().Err(err).Msg("Erro ao armazenar payload não descriptografável")
	}

	// Salvar mensagem no banco
	if err := eh.storage.messageRepo.Create(ctx, msg); err != nil {
		eh.storage.logger.Error().Err(err).Msg("Erro ao salvar mensagem não descriptografável")
		return err
	}

	eh.storage.logger.Info().
		Str("session_id", sessionID.String()).
		Str("message_id", evt.Info.ID).
		Str("chat_jid", evt.Info.Chat.String()).
		Str("sender_jid", evt.Info.Sender.String()).
		Bool("is_unavailable", evt.IsUnavailable).
		Str("unavailable_type", string(evt.UnavailableType)).
		Msg("Mensagem não descriptografável processada")

	return nil
}

// HandleReceipt processa eventos de confirmação de entrega/leitura
func (eh *EventHandlers) HandleReceipt(ctx context.Context, sessionID uuid.UUID, evt *events.Receipt) error {
	if len(evt.MessageIDs) == 0 {
		return nil
	}

	var status message.MessageStatus
	switch evt.Type {
	case "delivered":
		status = message.MessageStatusDelivered
	case "read":
		status = message.MessageStatusRead
	case "read-self":
		status = message.MessageStatusRead
	default:
		return nil // Tipo de recibo não reconhecido
	}

	// Atualizar status de todas as mensagens mencionadas no recibo
	for _, msgID := range evt.MessageIDs {
		if err := eh.storage.messageRepo.UpdateStatus(ctx, msgID, status); err != nil {
			eh.storage.logger.Error().
				Err(err).
				Str("session_id", sessionID.String()).
				Str("message_id", msgID).
				Str("status", string(status)).
				Msg("Erro ao atualizar status da mensagem")
			continue
		}

		eh.storage.logger.Debug().
			Str("session_id", sessionID.String()).
			Str("message_id", msgID).
			Str("status", string(status)).
			Msg("Status da mensagem atualizado")
	}

	return nil
}

// HandleContact processa eventos de informações de contato
func (eh *EventHandlers) HandleContact(ctx context.Context, sessionID uuid.UUID, evt *events.Contact) error {
	if evt.JID.IsEmpty() {
		return nil
	}

	contactJID := evt.JID.String()

	// Buscar contato existente
	existingContact, err := eh.storage.contactRepo.GetByJID(ctx, sessionID, contactJID)
	if err != nil && err != contact.ErrContactNotFound {
		return fmt.Errorf("erro ao buscar contato: %w", err)
	}

	if existingContact != nil {
		// Atualizar contato existente se houver mudanças
		updated := false
		// Eventos de Contact não têm PushName/BusinessName diretamente
		// Esses campos são atualizados via outros eventos

		if updated {
			existingContact.UpdatedAt = time.Now()
			return eh.storage.contactRepo.Update(ctx, existingContact)
		}
	} else {
		// Criar novo contato
		contactEntity := contact.NewContact(sessionID, contactJID)

		if err := eh.storage.contactRepo.Create(ctx, contactEntity); err != nil {
			return fmt.Errorf("erro ao criar contato: %w", err)
		}
	}

	return nil
}

// HandlePushName processa eventos de mudança de nome de exibição
func (eh *EventHandlers) HandlePushName(ctx context.Context, sessionID uuid.UUID, evt *events.PushName) error {
	if evt.JID.IsEmpty() || evt.NewPushName == "" {
		return nil
	}

	contactJID := evt.JID.String()

	// Buscar contato existente
	existingContact, err := eh.storage.contactRepo.GetByJID(ctx, sessionID, contactJID)
	if err != nil && err != contact.ErrContactNotFound {
		return fmt.Errorf("erro ao buscar contato: %w", err)
	}

	if existingContact != nil {
		// Atualizar nome do contato existente
		if existingContact.PushName != evt.NewPushName {
			existingContact.PushName = evt.NewPushName
			existingContact.UpdatedAt = time.Now()
			return eh.storage.contactRepo.Update(ctx, existingContact)
		}
	} else {
		// Criar novo contato
		contactEntity := contact.NewContact(sessionID, contactJID)
		contactEntity.PushName = evt.NewPushName

		if err := eh.storage.contactRepo.Create(ctx, contactEntity); err != nil {
			return fmt.Errorf("erro ao criar contato: %w", err)
		}
	}

	return nil
}

// HandlePresence processa eventos de presença (online/offline)
func (eh *EventHandlers) HandlePresence(ctx context.Context, sessionID uuid.UUID, evt *events.Presence) error {
	if evt.From.IsEmpty() {
		return nil
	}

	contactJID := evt.From.String()

	// Buscar contato existente
	existingContact, err := eh.storage.contactRepo.GetByJID(ctx, sessionID, contactJID)
	if err != nil && err != contact.ErrContactNotFound {
		return fmt.Errorf("erro ao buscar contato: %w", err)
	}

	if existingContact != nil {
		// Atualizar última vez visto
		if !evt.LastSeen.IsZero() {
			existingContact.LastSeen = &evt.LastSeen
			existingContact.UpdatedAt = time.Now()
			return eh.storage.contactRepo.Update(ctx, existingContact)
		}
	}

	return nil
}

// HandleChatPresence processa eventos de presença em chat (digitando, gravando, etc.)
func (eh *EventHandlers) HandleChatPresence(_ context.Context, sessionID uuid.UUID, evt *events.ChatPresence) error {
	// Por enquanto apenas logamos o evento, pode ser expandido para salvar estados de digitação
	eh.storage.logger.Debug().
		Str("session_id", sessionID.String()).
		Str("chat_jid", evt.MessageSource.Chat.String()).
		Str("sender_jid", evt.MessageSource.Sender.String()).
		Msg("Presença em chat detectada")

	return nil
}

// HandleMute processa eventos de silenciar chat
func (eh *EventHandlers) HandleMute(ctx context.Context, sessionID uuid.UUID, evt *events.Mute) error {
	chatJID := evt.JID.String()

	// Buscar chat existente
	existingChat, err := eh.storage.chatRepo.GetByJID(ctx, sessionID, chatJID)
	if err != nil {
		if err == chat.ErrChatNotFound {
			// Criar novo chat se não existir
			chatEntity := chat.NewChat(sessionID, chatJID, chat.ChatTypeIndividual)
			chatEntity.IsMuted = true
			// Note: MutedUntil field may not exist in chat entity

			if err := eh.storage.chatRepo.Create(ctx, chatEntity); err != nil {
				return fmt.Errorf("erro ao criar chat: %w", err)
			}
		} else {
			return fmt.Errorf("erro ao buscar chat: %w", err)
		}
	} else {
		// Atualizar chat existente
		existingChat.IsMuted = evt.Action.GetMuted()
		// Note: MutedUntil field may not exist in chat entity
		existingChat.UpdatedAt = time.Now()

		if err := eh.storage.chatRepo.Update(ctx, existingChat); err != nil {
			return fmt.Errorf("erro ao atualizar chat: %w", err)
		}
	}

	return nil
}

// HandleArchive processa eventos de arquivar chat
func (eh *EventHandlers) HandleArchive(ctx context.Context, sessionID uuid.UUID, evt *events.Archive) error {
	chatJID := evt.JID.String()

	// Buscar chat existente
	existingChat, err := eh.storage.chatRepo.GetByJID(ctx, sessionID, chatJID)
	if err != nil {
		if err == chat.ErrChatNotFound {
			// Criar novo chat se não existir
			chatEntity := chat.NewChat(sessionID, chatJID, chat.ChatTypeIndividual)
			chatEntity.IsArchived = evt.Action.GetArchived()

			if err := eh.storage.chatRepo.Create(ctx, chatEntity); err != nil {
				return fmt.Errorf("erro ao criar chat: %w", err)
			}
		} else {
			return fmt.Errorf("erro ao buscar chat: %w", err)
		}
	} else {
		// Atualizar chat existente
		existingChat.IsArchived = evt.Action.GetArchived()
		existingChat.UpdatedAt = time.Now()

		if err := eh.storage.chatRepo.Update(ctx, existingChat); err != nil {
			return fmt.Errorf("erro ao atualizar chat: %w", err)
		}
	}

	return nil
}

// HandlePin processa eventos de fixar chat
func (eh *EventHandlers) HandlePin(ctx context.Context, sessionID uuid.UUID, evt *events.Pin) error {
	chatJID := evt.JID.String()

	// Buscar chat existente
	existingChat, err := eh.storage.chatRepo.GetByJID(ctx, sessionID, chatJID)
	if err != nil {
		if err == chat.ErrChatNotFound {
			// Criar novo chat se não existir
			chatEntity := chat.NewChat(sessionID, chatJID, chat.ChatTypeIndividual)
			chatEntity.IsPinned = evt.Action.GetPinned()

			if err := eh.storage.chatRepo.Create(ctx, chatEntity); err != nil {
				return fmt.Errorf("erro ao criar chat: %w", err)
			}
		} else {
			return fmt.Errorf("erro ao buscar chat: %w", err)
		}
	} else {
		// Atualizar chat existente
		existingChat.IsPinned = evt.Action.GetPinned()
		existingChat.UpdatedAt = time.Now()

		if err := eh.storage.chatRepo.Update(ctx, existingChat); err != nil {
			return fmt.Errorf("erro ao atualizar chat: %w", err)
		}
	}

	return nil
}

// HandleGroupInfo processa eventos de informações de grupo
func (eh *EventHandlers) HandleGroupInfo(ctx context.Context, sessionID uuid.UUID, evt *events.GroupInfo) error {
	if evt.JID.IsEmpty() {
		return nil
	}

	chatJID := evt.JID.String()

	// Buscar chat existente
	existingChat, err := eh.storage.chatRepo.GetByJID(ctx, sessionID, chatJID)
	if err != nil && err != chat.ErrChatNotFound {
		return fmt.Errorf("erro ao buscar chat: %w", err)
	}

	if existingChat != nil {
		// Atualizar informações do grupo
		updated := false
		if evt.Name != nil && existingChat.Name != evt.Name.Name {
			existingChat.Name = evt.Name.Name
			updated = true
		}
		// Note: Description field may not exist in chat entity

		if updated {
			existingChat.UpdatedAt = time.Now()
			return eh.storage.chatRepo.Update(ctx, existingChat)
		}
	} else {
		// Criar novo chat de grupo
		chatEntity := chat.NewChat(sessionID, chatJID, chat.ChatTypeGroup)
		if evt.Name != nil {
			chatEntity.Name = evt.Name.Name
		}
		// Note: Description field may not exist in chat entity

		if err := eh.storage.chatRepo.Create(ctx, chatEntity); err != nil {
			return fmt.Errorf("erro ao criar chat de grupo: %w", err)
		}
	}

	return nil
}

// HandlePicture processa eventos de mudança de foto de perfil
func (eh *EventHandlers) HandlePicture(ctx context.Context, sessionID uuid.UUID, evt *events.Picture) error {
	if evt.JID.IsEmpty() {
		return nil
	}

	contactJID := evt.JID.String()

	// Buscar contato existente
	existingContact, err := eh.storage.contactRepo.GetByJID(ctx, sessionID, contactJID)
	if err != nil && err != contact.ErrContactNotFound {
		return fmt.Errorf("erro ao buscar contato: %w", err)
	}

	if existingContact != nil {
		// Atualizar foto do contato existente
		// Note: ProfilePictureURL field may not exist in contact entity
		// Picture events contain image data, not URL
		existingContact.UpdatedAt = time.Now()
		return eh.storage.contactRepo.Update(ctx, existingContact)
	} else {
		// Criar novo contato
		contactEntity := contact.NewContact(sessionID, contactJID)
		// Note: ProfilePictureURL field may not exist in contact entity

		if err := eh.storage.contactRepo.Create(ctx, contactEntity); err != nil {
			return fmt.Errorf("erro ao criar contato: %w", err)
		}
	}

	return nil
}
