package whatsapp

import (
	"fmt"

	entity "wamex/internal/domain/entity"
	domainRepo "wamex/internal/domain/repository"
	domainService "wamex/internal/domain/service"
)

// ManageSessionUseCase representa o caso de uso para gerenciamento de sessões
type ManageSessionUseCase struct {
	sessionRepo domainRepo.SessionRepository
	whatsappSvc domainService.SessionService
}

// NewManageSessionUseCase cria uma nova instância do use case
func NewManageSessionUseCase(
	sessionRepo domainRepo.SessionRepository,
	whatsappSvc domainService.SessionService,
) *ManageSessionUseCase {
	return &ManageSessionUseCase{
		sessionRepo: sessionRepo,
		whatsappSvc: whatsappSvc,
	}
}

// CreateSession cria uma nova sessão
func (uc *ManageSessionUseCase) CreateSession(req *entity.CreateSessionRequest) (*entity.Session, error) {
	// Validar se sessão já existe
	existingSession, err := uc.sessionRepo.GetBySession(req.Session)
	if err == nil && existingSession != nil {
		return nil, fmt.Errorf("session %s already exists", req.Session)
	}

	// Criar sessão através do service
	session, err := uc.whatsappSvc.CreateSession(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// ConnectSession conecta uma sessão existente
func (uc *ManageSessionUseCase) ConnectSession(sessionName string) error {
	// Validar se sessão existe
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("session %s not found: %w", sessionName, err)
	}

	if session.Status == entity.StatusConnected {
		return fmt.Errorf("session %s is already connected", sessionName)
	}

	// Conectar através do service
	return uc.whatsappSvc.ConnectSession(sessionName)
}

// DisconnectSession desconecta uma sessão
func (uc *ManageSessionUseCase) DisconnectSession(sessionName string) error {
	// Validar se sessão existe
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("session %s not found: %w", sessionName, err)
	}

	if session.Status != entity.StatusConnected {
		return fmt.Errorf("session %s is not connected", sessionName)
	}

	// Desconectar através do service
	return uc.whatsappSvc.DisconnectSession(sessionName)
}

// DeleteSession remove uma sessão
func (uc *ManageSessionUseCase) DeleteSession(sessionName string) error {
	// Validar se sessão existe
	_, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("session %s not found: %w", sessionName, err)
	}

	// Remover através do service
	return uc.whatsappSvc.DeleteSession(sessionName)
}

// GetSession obtém informações de uma sessão
func (uc *ManageSessionUseCase) GetSession(sessionName string) (*entity.Session, error) {
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return nil, fmt.Errorf("session %s not found: %w", sessionName, err)
	}

	return session, nil
}

// ListSessions lista todas as sessões
func (uc *ManageSessionUseCase) ListSessions() ([]*entity.Session, error) {
	sessions, err := uc.sessionRepo.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	return sessions, nil
}

// GenerateQRCode gera QR code para uma sessão
func (uc *ManageSessionUseCase) GenerateQRCode(sessionName string) (string, error) {
	// Validar se sessão existe
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return "", fmt.Errorf("session %s not found: %w", sessionName, err)
	}

	if session.Status == entity.StatusConnected {
		return "", fmt.Errorf("session %s is already connected", sessionName)
	}

	// Gerar QR code através do service
	return uc.whatsappSvc.GenerateQRCode(sessionName)
}

// PairPhone faz pareamento via código de telefone
func (uc *ManageSessionUseCase) PairPhone(sessionName, phone string) error {
	// Validar se sessão existe
	session, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return fmt.Errorf("session %s not found: %w", sessionName, err)
	}

	if session.Status == entity.StatusConnected {
		return fmt.Errorf("session %s is already connected", sessionName)
	}

	// Fazer pareamento através do service
	return uc.whatsappSvc.PairPhone(sessionName, phone)
}

// GetSessionStatus obtém o status de uma sessão
func (uc *ManageSessionUseCase) GetSessionStatus(sessionName string) (*entity.StatusResponse, error) {
	// Validar se sessão existe
	_, err := uc.sessionRepo.GetBySession(sessionName)
	if err != nil {
		return nil, fmt.Errorf("session %s not found: %w", sessionName, err)
	}

	// Obter status através do service
	return uc.whatsappSvc.GetSessionStatus(sessionName)
}

// GetConnectedSessionsCount obtém o número de sessões conectadas
func (uc *ManageSessionUseCase) GetConnectedSessionsCount() int {
	return uc.whatsappSvc.GetConnectedSessionsCount()
}
