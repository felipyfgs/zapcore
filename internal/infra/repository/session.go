package repository

import (
	"context"
	"fmt"
	"time"

	"zapcore/internal/domain/session"
	"zapcore/pkg/logger"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// SessionRepository implementa o repositório de sessões usando Bun ORM
type SessionRepository struct {
	db     *bun.DB
	logger *logger.Logger
}

// NewSessionRepository cria uma nova instância do repositório
func NewSessionRepository(db *bun.DB) *SessionRepository {
	return &SessionRepository{
		db:     db,
		logger: logger.Get(),
	}
}

// Create cria uma nova sessão
func (r *SessionRepository) Create(ctx context.Context, sess *session.Session) error {
	// Garantir que timestamps estão definidos
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = time.Now()
	}
	if sess.UpdatedAt.IsZero() {
		sess.UpdatedAt = time.Now()
	}

	_, err := r.db.NewInsert().
		Model(sess).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("erro ao criar sessão: %w", err)
	}

	r.logger.Info().Str("session_id", sess.ID.String()).Msg("Sessão criada com sucesso")
	return nil
}

// GetByID busca uma sessão pelo ID
func (r *SessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	sess := new(session.Session)
	err := r.db.NewSelect().
		Model(sess).
		Where(`"id" = ?`, id).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, session.ErrSessionNotFound
		}
		return nil, fmt.Errorf("erro ao buscar sessão por ID: %w", err)
	}

	return sess, nil
}

// GetByName busca uma sessão pelo nome
func (r *SessionRepository) GetByName(ctx context.Context, name string) (*session.Session, error) {
	sess := new(session.Session)
	err := r.db.NewSelect().
		Model(sess).
		Where("? = ?", bun.Ident("name"), name).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, session.ErrSessionNotFound
		}
		return nil, fmt.Errorf("erro ao buscar sessão por nome: %w", err)
	}

	return sess, nil
}

// GetByJID busca uma sessão pelo JID
func (r *SessionRepository) GetByJID(ctx context.Context, jid string) (*session.Session, error) {
	sess := new(session.Session)
	err := r.db.NewSelect().
		Model(sess).
		Where(`"jid" = ?`, jid).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, session.ErrSessionNotFound
		}
		return nil, fmt.Errorf("erro ao buscar sessão por JID: %w", err)
	}

	return sess, nil
}

// List lista todas as sessões com filtros
func (r *SessionRepository) List(ctx context.Context, filters session.ListFilters) ([]*session.Session, error) {
	var sessions []*session.Session

	query := r.db.NewSelect().Model(&sessions)

	// Aplicar filtros
	if filters.Status != nil {
		query = query.Where(`"status" = ?`, *filters.Status)
	}
	if filters.IsActive != nil {
		query = query.Where(`"isActive" = ?`, *filters.IsActive)
	}

	// Ordenação - usar switch para garantir case sensitivity correto
	orderDir := "DESC"
	if filters.OrderDir != "" {
		orderDir = filters.OrderDir
	}

	// Mapear orderBy para garantir case sensitivity correto
	var orderColumn string
	switch filters.OrderBy {
	case "createdAt", "":
		orderColumn = `"createdAt"`
	case "updatedAt":
		orderColumn = `"updatedAt"`
	case "name":
		orderColumn = `"name"`
	case "status":
		orderColumn = `"status"`
	case "isActive":
		orderColumn = `"isActive"`
	default:
		orderColumn = `"createdAt"` // fallback seguro
	}

	query = query.OrderExpr(orderColumn + " " + orderDir)

	// Paginação
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Scan(ctx)
	if err != nil {
		r.logger.Error().Err(err).Msg("Erro ao listar sessões")
		return nil, fmt.Errorf("erro ao listar sessões: %w", err)
	}

	return sessions, nil
}

// ListLegacy lista todas as sessões com paginação (compatibilidade)
func (r *SessionRepository) ListLegacy(ctx context.Context, limit, offset int) ([]*session.Session, error) {
	filters := session.ListFilters{
		Limit:  limit,
		Offset: offset,
	}
	return r.List(ctx, filters)
}

// ListActive lista sessões ativas
func (r *SessionRepository) ListActive(ctx context.Context) ([]*session.Session, error) {
	var sessions []*session.Session

	err := r.db.NewSelect().
		Model(&sessions).
		Where(`"isActive" = ?`, true).
		OrderExpr(`"createdAt" DESC`).
		Scan(ctx)

	if err != nil {
		r.logger.Error().Err(err).Msg("Erro ao listar sessões ativas")
		return nil, fmt.Errorf("erro ao listar sessões ativas: %w", err)
	}

	return sessions, nil
}

// ListByStatus lista sessões por status
func (r *SessionRepository) ListByStatus(ctx context.Context, status session.WhatsAppSessionStatus) ([]*session.Session, error) {
	var sessions []*session.Session

	err := r.db.NewSelect().
		Model(&sessions).
		Where(`"status" = ?`, status).
		OrderExpr(`"createdAt" DESC`).
		Scan(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("status", string(status)).Msg("Erro ao listar sessões por status")
		return nil, fmt.Errorf("erro ao listar sessões por status: %w", err)
	}

	return sessions, nil
}

// Update atualiza uma sessão existente
func (r *SessionRepository) Update(ctx context.Context, sess *session.Session) error {
	sess.UpdatedAt = time.Now()

	result, err := r.db.NewUpdate().
		Model(sess).
		Where("? = ?", bun.Ident("id"), sess.ID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("erro ao atualizar sessão: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return session.ErrSessionNotFound
	}

	r.logger.Info().Str("session_id", sess.ID.String()).Msg("Sessão atualizada com sucesso")
	return nil
}

// Delete remove uma sessão
func (r *SessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.NewDelete().
		Model((*session.Session)(nil)).
		Where("? = ?", bun.Ident("id"), id).
		Exec(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("session_id", id.String()).Msg("Erro ao deletar sessão")
		return fmt.Errorf("erro ao deletar sessão: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return session.ErrSessionNotFound
	}

	r.logger.Info().Str("session_id", id.String()).Msg("Sessão deletada com sucesso")
	return nil
}

// UpdateStatus atualiza apenas o status de uma sessão
func (r *SessionRepository) UpdateStatus(ctx context.Context, sessionID uuid.UUID, status session.WhatsAppSessionStatus) error {
	result, err := r.db.NewUpdate().
		Model((*session.Session)(nil)).
		Set(`"status" = ?`, status).
		Set(`"updatedAt" = ?`, time.Now()).
		Where(`"id" = ?`, sessionID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("erro ao atualizar status da sessão: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return session.ErrSessionNotFound
	}

	r.logger.Info().
		Str("session_id", sessionID.String()).
		Str("status", string(status)).
		Msg("Status da sessão atualizado com sucesso")

	return nil
}

// UpdateJID atualiza o JID de uma sessão
func (r *SessionRepository) UpdateJID(ctx context.Context, sessionID uuid.UUID, jid string) error {
	result, err := r.db.NewUpdate().
		Model((*session.Session)(nil)).
		Set(`"jid" = ?`, jid).
		Set(`"updatedAt" = ?`, time.Now()).
		Where(`"id" = ?`, sessionID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("erro ao atualizar JID da sessão: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return session.ErrSessionNotFound
	}

	r.logger.Info().
		Str("session_id", sessionID.String()).
		Str("jid", jid).
		Msg("JID da sessão atualizado com sucesso")

	return nil
}

// UpdateLastSeen atualiza o último acesso de uma sessão
func (r *SessionRepository) UpdateLastSeen(ctx context.Context, sessionID uuid.UUID) error {
	now := time.Now()
	result, err := r.db.NewUpdate().
		Model((*session.Session)(nil)).
		Set("? = ?", bun.Ident("lastSeen"), now).
		Set("? = ?", bun.Ident("updatedAt"), now).
		Where("? = ?", bun.Ident("id"), sessionID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("erro ao atualizar último acesso da sessão: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return session.ErrSessionNotFound
	}

	return nil
}

// Count conta o total de sessões
func (r *SessionRepository) Count(ctx context.Context) (int64, error) {
	count, err := r.db.NewSelect().
		Model((*session.Session)(nil)).
		Count(ctx)

	if err != nil {
		return 0, fmt.Errorf("erro ao contar sessões: %w", err)
	}

	return int64(count), nil
}

// GetActiveCount retorna o número de sessões ativas
func (r *SessionRepository) GetActiveCount(ctx context.Context) (int, error) {
	count, err := r.db.NewSelect().
		Model((*session.Session)(nil)).
		Where("? = ?", bun.Ident("isActive"), true).
		Count(ctx)

	if err != nil {
		return 0, fmt.Errorf("erro ao contar sessões ativas: %w", err)
	}

	return count, nil
}

// ExistsByName verifica se uma sessão existe pelo nome
func (r *SessionRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	exists, err := r.db.NewSelect().
		Model((*session.Session)(nil)).
		Where("? = ?", bun.Ident("name"), name).
		Exists(ctx)

	if err != nil {
		return false, fmt.Errorf("erro ao verificar existência da sessão: %w", err)
	}

	return exists, nil
}

// GetActiveSessions retorna todas as sessões ativas (compatibilidade com WhatsApp client)
func (r *SessionRepository) GetActiveSessions(ctx context.Context) ([]*session.Session, error) {
	return r.ListActive(ctx)
}
