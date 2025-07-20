package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"zapcore/internal/domain/session"
	"zapcore/pkg/logger"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// SessionData representa os dados básicos de uma sessão para reconexão
type SessionData struct {
	ID   uuid.UUID
	Name string
	JID  string
}

// SessionRepository implementa o repositório de sessões
type SessionRepository struct {
	db     *sql.DB
	logger *logger.Logger
}

// NewSessionRepository cria uma nova instância do repositório
func NewSessionRepository(db *sql.DB, zeroLogger zerolog.Logger) *SessionRepository {
	return &SessionRepository{
		db:     db,
		logger: logger.NewFromZerolog(zeroLogger),
	}
}

// Create cria uma nova sessão
func (r *SessionRepository) Create(ctx context.Context, sess *session.Session) error {
	query := `
		INSERT INTO zapcore_sessions (id, name, status, jid, qr_code, proxy_url, webhook, is_active, last_seen, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	metadataJSON, err := json.Marshal(sess.Metadata)
	if err != nil {
		return fmt.Errorf("erro ao serializar metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		sess.ID,
		sess.Name,
		sess.Status,
		sess.JID,
		sess.QRCode,
		sess.ProxyURL,
		sess.Webhook,
		sess.IsActive,
		sess.LastSeen,
		metadataJSON,
		sess.CreatedAt,
		sess.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("erro ao criar sessão: %w", err)
	}

	r.logger.Info().Str("session_id", sess.ID.String()).Msg("Sessão criada com sucesso")
	return nil
}

// GetByID busca uma sessão pelo ID
func (r *SessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	query := `
		SELECT id, name, status, jid, qr_code, proxy_url, webhook, is_active, last_seen, metadata, created_at, updated_at
		FROM zapcore_sessions
		WHERE id = $1
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanSession(row)
}

// GetByName busca uma sessão pelo nome
func (r *SessionRepository) GetByName(ctx context.Context, name string) (*session.Session, error) {
	query := `
		SELECT id, name, status, jid, qr_code, proxy_url, webhook, is_active, last_seen, metadata, created_at, updated_at
		FROM zapcore_sessions
		WHERE name = $1
	`

	row := r.db.QueryRowContext(ctx, query, name)
	return r.scanSession(row)
}

// List retorna todas as sessões com filtros opcionais
func (r *SessionRepository) List(ctx context.Context, filters session.ListFilters) ([]*session.Session, error) {
	query := "SELECT id, name, status, jid, qr_code, proxy_url, webhook, is_active, last_seen, metadata, created_at, updated_at FROM zapcore_sessions"
	args := []any{}
	conditions := []string{}
	argIndex := 1

	// Aplicar filtros
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filters.Status)
		argIndex++
	}

	if filters.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *filters.IsActive)
		argIndex++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Ordenação
	orderBy := "created_at"
	if filters.OrderBy != "" {
		orderBy = filters.OrderBy
	}

	orderDir := "DESC"
	if filters.OrderDir != "" {
		orderDir = filters.OrderDir
	}

	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDir)

	// Paginação
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar sessões: %w", err)
	}
	defer rows.Close()

	var sessions []*session.Session
	for rows.Next() {
		sess, err := r.scanSessionFromRows(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}

	return sessions, nil
}

// Update atualiza uma sessão existente
func (r *SessionRepository) Update(ctx context.Context, sess *session.Session) error {
	query := `
		UPDATE zapcore_sessions
		SET name = $2, status = $3, jid = $4, qr_code = $5, proxy_url = $6,
		    webhook = $7, is_active = $8, last_seen = $9, metadata = $10, updated_at = $11
		WHERE id = $1
	`

	metadataJSON, err := json.Marshal(sess.Metadata)
	if err != nil {
		return fmt.Errorf("erro ao serializar metadata: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query,
		sess.ID,
		sess.Name,
		sess.Status,
		sess.JID,
		sess.QRCode,
		sess.ProxyURL,
		sess.Webhook,
		sess.IsActive,
		sess.LastSeen,
		metadataJSON,
		time.Now(),
	)

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
	query := "DELETE FROM zapcore_sessions WHERE id = $1"

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
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

// GetActiveCount retorna o número de sessões ativas
func (r *SessionRepository) GetActiveCount(ctx context.Context) (int, error) {
	query := "SELECT COUNT(*) FROM sessions WHERE is_active = true"

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("erro ao contar sessões ativas: %w", err)
	}

	return count, nil
}

// UpdateStatus atualiza apenas o status de uma sessão
func (r *SessionRepository) UpdateStatus(ctx context.Context, sessionID uuid.UUID, status session.WhatsAppSessionStatus) error {
	query := `
		UPDATE zapcore_sessions
		SET status = $2, updated_at = $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, sessionID, status, time.Now())
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

// UpdateLastSeen atualiza o último acesso de uma sessão
func (r *SessionRepository) UpdateLastSeen(ctx context.Context, id uuid.UUID) error {
	query := "UPDATE sessions SET last_seen = $2, updated_at = $3 WHERE id = $1"

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, id, now, now)
	if err != nil {
		return fmt.Errorf("erro ao atualizar last_seen da sessão: %w", err)
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

// GetConnectedSessions retorna todas as sessões conectadas para reconexão
func (r *SessionRepository) GetConnectedSessions(ctx context.Context) ([]*session.Session, error) {
	query := `
		SELECT id, name, status, jid, qr_code, proxy_url, webhook, is_active, last_seen, metadata, created_at, updated_at
		FROM sessions
		WHERE status = 'connected' AND is_active = true
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar sessões conectadas: %w", err)
	}
	defer rows.Close()

	var sessions []*session.Session
	for rows.Next() {
		sess, err := r.scanSessionFromRows(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}

	return sessions, nil
}

// GetActiveSessions retorna todas as sessões ativas com JID para reconexão
func (r *SessionRepository) GetActiveSessions(ctx context.Context) ([]*SessionData, error) {
	query := `
		SELECT id, name, jid
		FROM zapcore_sessions
		WHERE is_active = true AND jid IS NOT NULL AND jid != ''
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar sessões ativas: %w", err)
	}
	defer rows.Close()

	var sessions []*SessionData
	for rows.Next() {
		var sess SessionData
		err := rows.Scan(&sess.ID, &sess.Name, &sess.JID)
		if err != nil {
			return nil, fmt.Errorf("erro ao escanear sessão ativa: %w", err)
		}
		sessions = append(sessions, &sess)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar sessões ativas: %w", err)
	}

	return sessions, nil
}

// UpdateJID atualiza o JID de uma sessão após pareamento bem-sucedido
func (r *SessionRepository) UpdateJID(ctx context.Context, sessionID uuid.UUID, jid string) error {
	query := `
		UPDATE zapcore_sessions
		SET jid = $2, updated_at = $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, sessionID, jid, time.Now())
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

// scanSession converte uma linha do banco em uma sessão
func (r *SessionRepository) scanSession(row *sql.Row) (*session.Session, error) {
	var sess session.Session
	var metadataJSON []byte

	err := row.Scan(
		&sess.ID,
		&sess.Name,
		&sess.Status,
		&sess.JID,
		&sess.QRCode,
		&sess.ProxyURL,
		&sess.Webhook,
		&sess.IsActive,
		&sess.LastSeen,
		&metadataJSON,
		&sess.CreatedAt,
		&sess.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, session.ErrSessionNotFound
		}
		return nil, fmt.Errorf("erro ao escanear sessão: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &sess.Metadata); err != nil {
			return nil, fmt.Errorf("erro ao deserializar metadata: %w", err)
		}
	}

	return &sess, nil
}

// scanSessionFromRows converte uma linha de rows em uma sessão
func (r *SessionRepository) scanSessionFromRows(rows *sql.Rows) (*session.Session, error) {
	var sess session.Session
	var metadataJSON []byte

	err := rows.Scan(
		&sess.ID,
		&sess.Name,
		&sess.Status,
		&sess.JID,
		&sess.QRCode,
		&sess.ProxyURL,
		&sess.Webhook,
		&sess.IsActive,
		&sess.LastSeen,
		&metadataJSON,
		&sess.CreatedAt,
		&sess.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("erro ao escanear sessão: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &sess.Metadata); err != nil {
			return nil, fmt.Errorf("erro ao deserializar metadata: %w", err)
		}
	}

	return &sess, nil
}
