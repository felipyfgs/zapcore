package repository

import (
	"context"
	"database/sql"
	"time"

	"wamex/internal/domain"

	"github.com/uptrace/bun"
)

// SessionRepository implementa a interface domain.SessionRepository usando bun ORM
type SessionRepository struct {
	db *bun.DB
}

// NewSessionRepository cria uma nova instância do repositório de sessões
func NewSessionRepository(db *bun.DB) *SessionRepository {
	return &SessionRepository{
		db: db,
	}
}

// Create cria uma nova sessão no banco de dados
func (r *SessionRepository) Create(session *domain.Session) error {
	ctx := context.Background()

	// Define timestamps
	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now

	_, err := r.db.NewInsert().
		Model(session).
		Exec(ctx)

	return err
}

// GetByID busca uma sessão por ID
func (r *SessionRepository) GetByID(id string) (*domain.Session, error) {
	ctx := context.Background()

	session := &domain.Session{}
	err := r.db.NewSelect().
		Model(session).
		Where("id = ?", id).
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return session, nil
}

// GetBySession busca uma sessão por nome
func (r *SessionRepository) GetBySession(sessionName string) (*domain.Session, error) {
	ctx := context.Background()

	session := &domain.Session{}
	err := r.db.NewSelect().
		Model(session).
		Where("session = ?", sessionName).
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return session, nil
}

// GetByToken busca uma sessão por token (não implementado ainda)
func (r *SessionRepository) GetByToken(token string) (*domain.Session, error) {
	// TODO: Implementar quando adicionar campo token
	return nil, nil
}

// Update atualiza uma sessão existente
func (r *SessionRepository) Update(session *domain.Session) error {
	ctx := context.Background()

	// Atualiza timestamp
	session.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().
		Model(session).
		Where("id = ?", session.ID).
		Exec(ctx)

	return err
}

// Delete remove uma sessão do banco de dados
func (r *SessionRepository) Delete(id string) error {
	ctx := context.Background()

	_, err := r.db.NewDelete().
		Model((*domain.Session)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// DeleteBySession remove uma sessão por nome
func (r *SessionRepository) DeleteBySession(sessionName string) error {
	ctx := context.Background()

	_, err := r.db.NewDelete().
		Model((*domain.Session)(nil)).
		Where("session = ?", sessionName).
		Exec(ctx)

	return err
}

// List retorna todas as sessões
func (r *SessionRepository) List() ([]*domain.Session, error) {
	ctx := context.Background()

	var sessions []*domain.Session
	err := r.db.NewSelect().
		Model(&sessions).
		Order("created_at DESC").
		Scan(ctx)

	return sessions, err
}

// GetActive retorna apenas as sessões ativas (conectadas)
func (r *SessionRepository) GetActive() ([]*domain.Session, error) {
	ctx := context.Background()

	var sessions []*domain.Session
	err := r.db.NewSelect().
		Model(&sessions).
		Where("status = ?", domain.StatusConnected).
		Order("created_at DESC").
		Scan(ctx)

	return sessions, err
}

// GetConnectedSessions retorna sessões que devem ser reconectadas (com DeviceJID)
func (r *SessionRepository) GetConnectedSessions() ([]*domain.Session, error) {
	ctx := context.Background()

	var sessions []*domain.Session
	err := r.db.NewSelect().
		Model(&sessions).
		Where("device_jid != '' AND device_jid IS NOT NULL").
		Order("created_at DESC").
		Scan(ctx)

	return sessions, err
}

// CreateTable cria a tabela de sessões se não existir
func (r *SessionRepository) CreateTable(ctx context.Context) error {
	_, err := r.db.NewCreateTable().
		Model((*domain.Session)(nil)).
		IfNotExists().
		Exec(ctx)

	return err
}
