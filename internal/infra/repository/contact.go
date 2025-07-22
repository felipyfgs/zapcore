package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"zapcore/internal/domain/contact"
	"zapcore/pkg/logger"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ContactRepository implementa o repositório de contatos usando Bun ORM
type ContactRepository struct {
	db     *bun.DB
	logger *logger.Logger
}

// NewContactRepository cria uma nova instância do repositório
func NewContactRepository(db *bun.DB) *ContactRepository {
	return &ContactRepository{
		db:     db,
		logger: logger.Get(),
	}
}

// Create cria um novo contato
func (r *ContactRepository) Create(ctx context.Context, c *contact.Contact) error {
	// Garantir que timestamps estão definidos
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = time.Now()
	}

	_, err := r.db.NewInsert().
		Model(c).
		Exec(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("contact_jid", c.JID).Msg("Erro ao criar contato")
		return fmt.Errorf("erro ao criar contato: %w", err)
	}

	r.logger.Info().Str("contact_jid", c.JID).Str("session_id", c.SessionID.String()).Msg("Contato criado com sucesso")
	return nil
}

// GetByID busca um contato pelo ID
func (r *ContactRepository) GetByID(ctx context.Context, id uuid.UUID) (*contact.Contact, error) {
	c := new(contact.Contact)
	err := r.db.NewSelect().
		Model(c).
		Where(`"id" = ?`, id).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, contact.ErrContactNotFound
		}
		return nil, fmt.Errorf("erro ao buscar contato por ID: %w", err)
	}

	return c, nil
}

// GetByJID busca um contato pelo JID e session ID
func (r *ContactRepository) GetByJID(ctx context.Context, sessionID uuid.UUID, jid string) (*contact.Contact, error) {
	c := new(contact.Contact)
	err := r.db.NewSelect().
		Model(c).
		Where("? = ? AND ? = ?", bun.Ident("sessionId"), sessionID, bun.Ident("jid"), jid).
		Scan(ctx)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, contact.ErrContactNotFound
		}
		return nil, fmt.Errorf("erro ao buscar contato por JID: %w", err)
	}

	return c, nil
}

// Update atualiza um contato
func (r *ContactRepository) Update(ctx context.Context, c *contact.Contact) error {
	c.UpdatedAt = time.Now()

	result, err := r.db.NewUpdate().
		Model(c).
		Where("? = ?", bun.Ident("id"), c.ID).
		Exec(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("contact_jid", c.JID).Msg("Erro ao atualizar contato")
		return fmt.Errorf("erro ao atualizar contato: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return contact.ErrContactNotFound
	}

	r.logger.Info().Str("contact_jid", c.JID).Msg("Contato atualizado com sucesso")
	return nil
}

// Delete remove um contato
func (r *ContactRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.NewDelete().
		Model((*contact.Contact)(nil)).
		Where("? = ?", bun.Ident("id"), id).
		Exec(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("contact_id", id.String()).Msg("Erro ao deletar contato")
		return fmt.Errorf("erro ao deletar contato: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return contact.ErrContactNotFound
	}

	r.logger.Info().Str("contact_id", id.String()).Msg("Contato deletado com sucesso")
	return nil
}

// List retorna uma lista de contatos com filtros
func (r *ContactRepository) List(ctx context.Context, filters contact.ListFilters) ([]*contact.Contact, error) {
	query := r.db.NewSelect().Model(&[]*contact.Contact{})

	// Aplicar filtros se fornecidos
	if filters.SessionID != nil {
		query = query.Where(`"sessionId" = ?`, *filters.SessionID)
	}

	// Definir limite padrão
	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}

	var contacts []*contact.Contact
	err := query.
		OrderExpr(`"pushName" ASC NULLS LAST`).
		Limit(limit).
		Offset(filters.Offset).
		Scan(ctx, &contacts)

	if err != nil {
		r.logger.Error().Err(err).Msg("Erro ao listar contatos")
		return nil, fmt.Errorf("erro ao listar contatos: %w", err)
	}

	return contacts, nil
}

// ListBySessionID lista contatos por session ID com paginação
func (r *ContactRepository) ListBySessionID(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*contact.Contact, error) {
	var contacts []*contact.Contact

	err := r.db.NewSelect().
		Model(&contacts).
		Where("? = ?", bun.Ident("sessionId"), sessionID).
		OrderExpr("? ASC NULLS LAST", bun.Ident("pushName")).
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao listar contatos")
		return nil, fmt.Errorf("erro ao listar contatos: %w", err)
	}

	return contacts, nil
}

// ListGroups lista contatos que são grupos
func (r *ContactRepository) ListGroups(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*contact.Contact, error) {
	var contacts []*contact.Contact

	err := r.db.NewSelect().
		Model(&contacts).
		Where("? = ? AND ? = ?", bun.Ident("sessionId"), sessionID, bun.Ident("isGroup"), true).
		OrderExpr("? ASC NULLS LAST", bun.Ident("pushName")).
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao listar grupos")
		return nil, fmt.Errorf("erro ao listar grupos: %w", err)
	}

	return contacts, nil
}

// ListBusiness lista contatos que são contas business
func (r *ContactRepository) ListBusiness(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*contact.Contact, error) {
	var contacts []*contact.Contact

	// Buscar todos os contatos da sessão e filtrar no Go
	err := r.db.NewSelect().
		Model(&contacts).
		Where("? = ?", bun.Ident("sessionId"), sessionID).
		OrderExpr("? ASC", bun.Ident("businessName")).
		Limit(limit * 2). // Buscar mais para compensar filtro
		Offset(offset).
		Scan(ctx)

	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Erro ao listar contatos business")
		return nil, fmt.Errorf("erro ao listar contatos business: %w", err)
	}

	// Filtrar contatos business no Go
	var businessContacts []*contact.Contact
	for _, contact := range contacts {
		if contact.BusinessName != "" {
			businessContacts = append(businessContacts, contact)
			if len(businessContacts) >= limit {
				break
			}
		}
	}

	return businessContacts, nil
}

// SearchByQuery busca contatos por nome ou JID (método interno)
func (r *ContactRepository) SearchByQuery(ctx context.Context, sessionID uuid.UUID, query string, limit, offset int) ([]*contact.Contact, error) {
	var allContacts []*contact.Contact

	// Buscar todos os contatos da sessão
	err := r.db.NewSelect().
		Model(&allContacts).
		Where(`"sessionId" = ?`, sessionID).
		OrderExpr(`"pushName" ASC NULLS LAST`).
		Scan(ctx)

	if err != nil {
		r.logger.Error().Err(err).
			Str("session_id", sessionID.String()).
			Str("query", query).
			Msg("Erro ao buscar contatos")
		return nil, fmt.Errorf("erro ao buscar contatos: %w", err)
	}

	// Filtrar contatos no Go usando strings.Contains (case insensitive)
	var filteredContacts []*contact.Contact
	queryLower := strings.ToLower(query)

	for _, contact := range allContacts {
		if strings.Contains(strings.ToLower(contact.PushName), queryLower) ||
			strings.Contains(strings.ToLower(contact.BusinessName), queryLower) ||
			strings.Contains(strings.ToLower(contact.JID), queryLower) {
			filteredContacts = append(filteredContacts, contact)
		}
	}

	// Aplicar paginação no Go
	start := offset
	end := offset + limit
	if start > len(filteredContacts) {
		return []*contact.Contact{}, nil
	}
	if end > len(filteredContacts) {
		end = len(filteredContacts)
	}

	return filteredContacts[start:end], nil
}

// UpdateLastSeen atualiza o último acesso do contato
func (r *ContactRepository) UpdateLastSeen(ctx context.Context, sessionID uuid.UUID, jid string, lastSeen time.Time) error {
	result, err := r.db.NewUpdate().
		Model((*contact.Contact)(nil)).
		Set("? = ?", bun.Ident("lastSeen"), lastSeen).
		Set("? = ?", bun.Ident("updatedAt"), time.Now()).
		Where("? = ? AND ? = ?", bun.Ident("sessionId"), sessionID, bun.Ident("jid"), jid).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("erro ao atualizar último acesso do contato: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return contact.ErrContactNotFound
	}

	return nil
}

// Count conta o total de contatos
func (r *ContactRepository) Count(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	count, err := r.db.NewSelect().
		Model((*contact.Contact)(nil)).
		Where("? = ?", bun.Ident("sessionId"), sessionID).
		Count(ctx)

	if err != nil {
		return 0, fmt.Errorf("erro ao contar contatos: %w", err)
	}

	return int64(count), nil
}

// CountGroups conta o total de grupos
func (r *ContactRepository) CountGroups(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	count, err := r.db.NewSelect().
		Model((*contact.Contact)(nil)).
		Where("? = ? AND ? = ?", bun.Ident("sessionId"), sessionID, bun.Ident("isGroup"), true).
		Count(ctx)

	if err != nil {
		return 0, fmt.Errorf("erro ao contar grupos: %w", err)
	}

	return int64(count), nil
}

// ExistsByJID verifica se um contato existe pelo JID
func (r *ContactRepository) ExistsByJID(ctx context.Context, sessionID uuid.UUID, jid string) (bool, error) {
	exists, err := r.db.NewSelect().
		Model((*contact.Contact)(nil)).
		Where(`"sessionId" = ? AND "jid" = ?`, sessionID, jid).
		Exists(ctx)

	if err != nil {
		return false, fmt.Errorf("erro ao verificar existência do contato: %w", err)
	}

	return exists, nil
}

// GetBySessionID implementa a interface contact.Repository
func (r *ContactRepository) GetBySessionID(ctx context.Context, sessionID uuid.UUID, filters contact.ListFilters) ([]*contact.Contact, error) {
	// Usar o filtro SessionID
	filters.SessionID = &sessionID
	return r.List(ctx, filters)
}

// GetBusinessContacts implementa a interface contact.Repository
func (r *ContactRepository) GetBusinessContacts(ctx context.Context, sessionID uuid.UUID, filters contact.ListFilters) ([]*contact.Contact, error) {
	// Usar o método ListBusiness existente
	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	return r.ListBusiness(ctx, sessionID, limit, filters.Offset)
}

// GetGroupContacts implementa a interface contact.Repository
func (r *ContactRepository) GetGroupContacts(ctx context.Context, sessionID uuid.UUID, filters contact.ListFilters) ([]*contact.Contact, error) {
	// Usar o método ListGroups existente
	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	return r.ListGroups(ctx, sessionID, limit, filters.Offset)
}

// Search implementa a interface contact.Repository
func (r *ContactRepository) Search(ctx context.Context, sessionID uuid.UUID, query string, filters contact.ListFilters) ([]*contact.Contact, error) {
	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	return r.SearchByQuery(ctx, sessionID, query, limit, filters.Offset)
}
