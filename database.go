package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

// SessionDB representa uma sessão no banco de dados
type SessionDB struct {
	bun.BaseModel `bun:"table:sessions,alias:s"`

	ID          string     `bun:"id,pk" json:"id"`
	Name        string     `bun:"name,notnull" json:"name"`
	Status      string     `bun:"status,notnull,default:'disconnected'" json:"status"`
	DeviceJID   *string    `bun:"device_jid,nullzero" json:"device_jid,omitempty"` // FK para whatsmeow_device
	CreatedAt   time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	ConnectedAt *time.Time `bun:"connected_at,nullzero" json:"connected_at,omitempty"`
	LastSeen    *time.Time `bun:"last_seen,nullzero" json:"last_seen,omitempty"`

	// Relação com device
	Device *WhatsmeowDevice `bun:"rel:belongs-to,join:device_jid=jid" json:"device,omitempty"`
}

// WhatsmeowDevice representa um device do WhatsApp
type WhatsmeowDevice struct {
	bun.BaseModel `bun:"table:whatsmeow_device,alias:d"`

	JID            string `bun:"jid,pk" json:"jid"`
	LID            string `bun:"lid" json:"lid,omitempty"`
	FacebookUUID   string `bun:"facebook_uuid" json:"facebook_uuid,omitempty"`
	RegistrationID int64  `bun:"registration_id" json:"registration_id"`
	Platform       string `bun:"platform" json:"platform"`
	BusinessName   string `bun:"business_name" json:"business_name"`
	PushName       string `bun:"push_name" json:"push_name"`
	LIDMigrationTS int64  `bun:"lid_migration_ts" json:"lid_migration_ts"`
}

// DatabaseManager gerencia as operações do banco de dados
type DatabaseManager struct {
	db *bun.DB
}

// NewDatabaseManager cria um novo gerenciador de banco de dados
func NewDatabaseManager(dbPath string) (*DatabaseManager, error) {
	// Abrir conexão SQLite usando sqliteshim (driver puro Go)
	sqldb, err := sql.Open(sqliteshim.ShimName, dbPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir banco de dados: %v", err)
	}

	// Criar instância Bun
	db := bun.NewDB(sqldb, sqlitedialect.New())

	// Testar conexão
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("erro ao conectar com banco de dados: %v", err)
	}

	dm := &DatabaseManager{db: db}

	// Criar tabelas se não existirem
	if err := dm.createTables(context.Background()); err != nil {
		return nil, fmt.Errorf("erro ao criar tabelas: %v", err)
	}

	return dm, nil
}

// createTables cria as tabelas necessárias
func (dm *DatabaseManager) createTables(ctx context.Context) error {
	// Primeiro, criar tabela sessions se não existir
	_, err := dm.db.NewCreateTable().
		Model((*SessionDB)(nil)).
		IfNotExists().
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("erro ao criar tabela sessions: %v", err)
	}

	// Depois verificar se a coluna device_jid já existe
	var columnExists bool
	err = dm.db.NewRaw("SELECT COUNT(*) > 0 FROM pragma_table_info('sessions') WHERE name = 'device_jid'").
		Scan(ctx, &columnExists)

	if err == nil && !columnExists {
		// Adicionar coluna device_jid se não existir
		_, err = dm.db.NewRaw("ALTER TABLE sessions ADD COLUMN device_jid TEXT REFERENCES whatsmeow_device(jid) ON DELETE CASCADE").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("erro ao adicionar coluna device_jid: %v", err)
		}
	}

	// Criar índices
	_, err = dm.db.NewCreateIndex().
		Table("sessions").
		Index("idx_sessions_status").
		Column("status").
		IfNotExists().
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("erro ao criar índice status: %v", err)
	}

	_, err = dm.db.NewCreateIndex().
		Table("sessions").
		Index("idx_sessions_created_at").
		Column("created_at").
		IfNotExists().
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("erro ao criar índice created_at: %v", err)
	}

	// Criar índice para device_jid
	_, err = dm.db.NewCreateIndex().
		Table("sessions").
		Index("idx_sessions_device_jid").
		Column("device_jid").
		IfNotExists().
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("erro ao criar índice device_jid: %v", err)
	}

	return nil
}

// SaveSession salva uma sessão no banco de dados
func (dm *DatabaseManager) SaveSession(ctx context.Context, session *WhatsAppSession) error {
	sessionDB := &SessionDB{
		ID:          session.ID,
		Name:        session.Name,
		Status:      string(session.Status),
		DeviceJID:   session.DeviceJID,
		CreatedAt:   session.CreatedAt,
		ConnectedAt: session.ConnectedAt,
		LastSeen:    session.LastSeen,
	}

	_, err := dm.db.NewInsert().
		Model(sessionDB).
		On("CONFLICT (id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("status = EXCLUDED.status").
		Set("device_jid = EXCLUDED.device_jid").
		Set("connected_at = EXCLUDED.connected_at").
		Set("last_seen = EXCLUDED.last_seen").
		Exec(ctx)

	return err
}

// GetSession obtém uma sessão do banco de dados
func (dm *DatabaseManager) GetSession(ctx context.Context, id string) (*SessionDB, error) {
	session := &SessionDB{}

	err := dm.db.NewSelect().
		Model(session).
		Where("id = ?", id).
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	return session, nil
}

// GetAllSessions obtém todas as sessões do banco de dados
func (dm *DatabaseManager) GetAllSessions(ctx context.Context) ([]*SessionDB, error) {
	var sessions []*SessionDB

	err := dm.db.NewSelect().
		Model(&sessions).
		Order("created_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	return sessions, nil
}

// GetConnectedSessions obtém sessões que estavam conectadas
func (dm *DatabaseManager) GetConnectedSessions(ctx context.Context) ([]*SessionDB, error) {
	var sessions []*SessionDB

	err := dm.db.NewSelect().
		Model(&sessions).
		Where("status = ?", StatusConnected).
		Order("created_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	return sessions, nil
}

// DeleteSession remove uma sessão do banco de dados
func (dm *DatabaseManager) DeleteSession(ctx context.Context, id string) error {
	_, err := dm.db.NewDelete().
		Model((*SessionDB)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// UpdateSessionStatus atualiza o status de uma sessão
func (dm *DatabaseManager) UpdateSessionStatus(ctx context.Context, id string, status SessionStatus, connectedAt, lastSeen *time.Time) error {
	_, err := dm.db.NewUpdate().
		Model((*SessionDB)(nil)).
		Set("status = ?", string(status)).
		Set("connected_at = ?", connectedAt).
		Set("last_seen = ?", lastSeen).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// Close fecha a conexão com o banco de dados
func (dm *DatabaseManager) Close() error {
	return dm.db.Close()
}

// GetSessionCount retorna o número total de sessões
func (dm *DatabaseManager) GetSessionCount(ctx context.Context) (int, error) {
	count, err := dm.db.NewSelect().
		Model((*SessionDB)(nil)).
		Count(ctx)

	return count, err
}

// GetSessionsByStatus retorna sessões filtradas por status
func (dm *DatabaseManager) GetSessionsByStatus(ctx context.Context, status SessionStatus) ([]*SessionDB, error) {
	var sessions []*SessionDB

	err := dm.db.NewSelect().
		Model(&sessions).
		Where("status = ?", string(status)).
		Order("created_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	return sessions, nil
}

// SessionExists verifica se uma sessão existe
func (dm *DatabaseManager) SessionExists(ctx context.Context, id string) (bool, error) {
	exists, err := dm.db.NewSelect().
		Model((*SessionDB)(nil)).
		Where("id = ?", id).
		Exists(ctx)

	return exists, err
}

// ToWhatsAppSession converte SessionDB para WhatsAppSession
func (s *SessionDB) ToWhatsAppSession() *WhatsAppSession {
	return &WhatsAppSession{
		ID:          s.ID,
		Name:        s.Name,
		Status:      SessionStatus(s.Status),
		DeviceJID:   s.DeviceJID,
		CreatedAt:   s.CreatedAt,
		ConnectedAt: s.ConnectedAt,
		LastSeen:    s.LastSeen,
	}
}

// LinkSessionToDevice vincula uma sessão a um device
func (dm *DatabaseManager) LinkSessionToDevice(ctx context.Context, sessionID, deviceJID string) error {
	_, err := dm.db.NewUpdate().
		Model((*SessionDB)(nil)).
		Set("device_jid = ?", deviceJID).
		Where("id = ?", sessionID).
		Exec(ctx)

	return err
}

// UnlinkSessionFromDevice remove a vinculação entre sessão e device
func (dm *DatabaseManager) UnlinkSessionFromDevice(ctx context.Context, sessionID string) error {
	_, err := dm.db.NewUpdate().
		Model((*SessionDB)(nil)).
		Set("device_jid = NULL").
		Where("id = ?", sessionID).
		Exec(ctx)

	return err
}

// GetSessionWithDevice obtém uma sessão com seus dados de device
func (dm *DatabaseManager) GetSessionWithDevice(ctx context.Context, id string) (*SessionDB, error) {
	session := &SessionDB{}

	err := dm.db.NewSelect().
		Model(session).
		Relation("Device").
		Where("s.id = ?", id).
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	return session, nil
}

// DeleteSessionAndDevice remove uma sessão e seu device associado
func (dm *DatabaseManager) DeleteSessionAndDevice(ctx context.Context, sessionID string) error {
	// Buscar o device_jid da sessão
	session := &SessionDB{}
	err := dm.db.NewSelect().
		Model(session).
		Column("device_jid").
		Where("id = ?", sessionID).
		Scan(ctx)

	if err != nil {
		return fmt.Errorf("erro ao buscar sessão: %v", err)
	}

	// Iniciar transação
	tx, err := dm.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("erro ao iniciar transação: %v", err)
	}
	defer tx.Rollback()

	// Remover sessão (isso deve remover o device automaticamente devido ao CASCADE)
	_, err = tx.NewDelete().
		Model((*SessionDB)(nil)).
		Where("id = ?", sessionID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("erro ao remover sessão: %v", err)
	}

	// Se houver device_jid, remover o device explicitamente
	if session.DeviceJID != nil && *session.DeviceJID != "" {
		_, err = tx.NewDelete().
			Model((*WhatsmeowDevice)(nil)).
			Where("jid = ?", *session.DeviceJID).
			Exec(ctx)

		if err != nil {
			return fmt.Errorf("erro ao remover device: %v", err)
		}
	}

	// Commit da transação
	return tx.Commit()
}

// GetSessionByName obtém uma sessão pelo nome
func (dm *DatabaseManager) GetSessionByName(ctx context.Context, name string) (*SessionDB, error) {
	session := &SessionDB{}

	err := dm.db.NewSelect().
		Model(session).
		Where("name = ?", name).
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	return session, nil
}

// GetSessionByNameWithDevice obtém uma sessão pelo nome com dados do device
func (dm *DatabaseManager) GetSessionByNameWithDevice(ctx context.Context, name string) (*SessionDB, error) {
	session := &SessionDB{}

	err := dm.db.NewSelect().
		Model(session).
		Relation("Device").
		Where("s.name = ?", name).
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	return session, nil
}

// SessionExistsByName verifica se uma sessão existe pelo nome
func (dm *DatabaseManager) SessionExistsByName(ctx context.Context, name string) (bool, error) {
	exists, err := dm.db.NewSelect().
		Model((*SessionDB)(nil)).
		Where("name = ?", name).
		Exists(ctx)

	return exists, err
}

// DeleteSessionByName remove uma sessão pelo nome
func (dm *DatabaseManager) DeleteSessionByName(ctx context.Context, name string) error {
	_, err := dm.db.NewDelete().
		Model((*SessionDB)(nil)).
		Where("name = ?", name).
		Exec(ctx)

	return err
}

// DeleteSessionAndDeviceByName remove uma sessão e seu device pelo nome
func (dm *DatabaseManager) DeleteSessionAndDeviceByName(ctx context.Context, name string) error {
	// Buscar a sessão pelo nome
	session := &SessionDB{}
	err := dm.db.NewSelect().
		Model(session).
		Column("id", "device_jid").
		Where("name = ?", name).
		Scan(ctx)

	if err != nil {
		return fmt.Errorf("erro ao buscar sessão: %v", err)
	}

	// Usar o método existente com o ID
	return dm.DeleteSessionAndDevice(ctx, session.ID)
}
