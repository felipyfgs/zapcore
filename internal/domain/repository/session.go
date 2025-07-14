package domain

import (
	entity "wamex/internal/domain/entity"
)

// SessionRepository define a interface para operações de sessão no banco de dados
type SessionRepository interface {
	Create(session *entity.Session) error
	GetByID(id string) (*entity.Session, error)
	GetBySession(sessionName string) (*entity.Session, error)
	GetByToken(token string) (*entity.Session, error)
	Update(session *entity.Session) error
	Delete(id string) error
	DeleteBySession(sessionName string) error
	List() ([]*entity.Session, error)
	GetActive() ([]*entity.Session, error)
	GetConnectedSessions() ([]*entity.Session, error)
}
