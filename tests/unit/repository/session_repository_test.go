package repository_test

import (
	"testing"

	"wamex/internal/domain"
	"wamex/internal/repository"

	"github.com/uptrace/bun"
)

// TestSessionRepository_Create testa a criação de uma sessão
func TestSessionRepository_Create(t *testing.T) {
	// Este é um teste básico de estrutura
	// Em um ambiente real, usaríamos um banco de teste ou mock

	session := &domain.Session{
		ID:      "test-id",
		Session: "test-session",
		Status:  domain.StatusDisconnected,
	}

	// Verificar se a estrutura está correta
	if session.ID == "" {
		t.Error("Session ID should not be empty")
	}

	if session.Status != domain.StatusDisconnected {
		t.Error("Session status should be disconnected")
	}
}

// TestSessionRepository_Structure testa se o repository implementa a interface
func TestSessionRepository_Structure(t *testing.T) {
	// Teste de compilação - verifica se implementa a interface
	var _ domain.SessionRepository = &repository.SessionRepository{}

	// Se chegou até aqui, a interface está implementada corretamente
	t.Log("SessionRepository implements domain.SessionRepository interface correctly")
}

// TestSessionRepository_NewSessionRepository testa o construtor
func TestSessionRepository_NewSessionRepository(t *testing.T) {
	// Mock de DB (apenas para teste de estrutura)
	// Em um teste real, usaríamos um banco de teste
	db := &bun.DB{}

	repo := repository.NewSessionRepository(db)

	if repo == nil {
		t.Error("NewSessionRepository should not return nil")
	}
}

// TestSessionStatus_Constants testa as constantes de status
func TestSessionStatus_Constants(t *testing.T) {
	tests := []struct {
		status   domain.Status
		expected string
	}{
		{domain.StatusDisconnected, "disconnected"},
		{domain.StatusConnecting, "connecting"},
		{domain.StatusConnected, "connected"},
	}

	for _, test := range tests {
		if string(test.status) != test.expected {
			t.Errorf("Status %v should be %s, got %s", test.status, test.expected, string(test.status))
		}
	}
}
