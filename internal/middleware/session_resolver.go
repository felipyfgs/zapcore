package middleware

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"wamex/internal/domain"
	"wamex/pkg/logger"
)

// SessionService interface para resolução de sessões
type SessionService interface {
	GetSession(sessionName string) (*domain.Session, error)
	GetSessionByID(sessionID string) (*domain.Session, error)
}

// CacheEntry representa uma entrada no cache de resolução
type CacheEntry struct {
	SessionID string
	ExpiresAt time.Time
}

// SessionResolver middleware para resolução automática de identificadores
type SessionResolver struct {
	sessionService SessionService
	cache          map[string]*CacheEntry
	cacheMutex     sync.RWMutex
	cacheTimeout   time.Duration
}

// NewSessionResolver cria uma nova instância do resolver
func NewSessionResolver(sessionService SessionService) *SessionResolver {
	return &SessionResolver{
		sessionService: sessionService,
		cache:          make(map[string]*CacheEntry),
		cacheTimeout:   5 * time.Minute, // Cache por 5 minutos
	}
}

// uuidPattern regex para detectar UUIDs
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// isUUID verifica se a string é um UUID válido
func (sr *SessionResolver) isUUID(identifier string) bool {
	return uuidPattern.MatchString(strings.ToLower(identifier))
}

// isValidSessionName verifica se é um nome de sessão válido
func (sr *SessionResolver) isValidSessionName(identifier string) bool {
	// Validação básica: não vazio, sem caracteres especiais perigosos
	if len(identifier) == 0 || len(identifier) > 100 {
		return false
	}
	
	// Permite letras, números, hífens e underscores
	validNamePattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	return validNamePattern.MatchString(identifier)
}

// getCachedSessionID busca no cache
func (sr *SessionResolver) getCachedSessionID(sessionName string) (string, bool) {
	sr.cacheMutex.RLock()
	defer sr.cacheMutex.RUnlock()
	
	entry, exists := sr.cache[sessionName]
	if !exists {
		return "", false
	}
	
	// Verifica se o cache expirou
	if time.Now().After(entry.ExpiresAt) {
		// Remove entrada expirada (será limpa depois)
		return "", false
	}
	
	return entry.SessionID, true
}

// setCachedSessionID adiciona ao cache
func (sr *SessionResolver) setCachedSessionID(sessionName, sessionID string) {
	sr.cacheMutex.Lock()
	defer sr.cacheMutex.Unlock()
	
	sr.cache[sessionName] = &CacheEntry{
		SessionID: sessionID,
		ExpiresAt: time.Now().Add(sr.cacheTimeout),
	}
}

// cleanExpiredCache remove entradas expiradas do cache
func (sr *SessionResolver) cleanExpiredCache() {
	sr.cacheMutex.Lock()
	defer sr.cacheMutex.Unlock()
	
	now := time.Now()
	for key, entry := range sr.cache {
		if now.After(entry.ExpiresAt) {
			delete(sr.cache, key)
		}
	}
}

// resolveSessionIdentifier resolve o identificador para sessionID
func (sr *SessionResolver) resolveSessionIdentifier(identifier string) (string, error) {
	// Limpa cache expirado periodicamente
	go sr.cleanExpiredCache()
	
	// Chain-of-thought para resolução:
	
	// 1. Se é UUID, usar diretamente como sessionID
	if sr.isUUID(identifier) {
		logger.WithComponent("middleware").Debug().
			Str("identifier", identifier).
			Str("type", "uuid").
			Msg("Identifier detected as UUID")
		return identifier, nil
	}
	
	// 2. Se é nome válido, buscar sessionID correspondente
	if sr.isValidSessionName(identifier) {
		logger.WithComponent("middleware").Debug().
			Str("identifier", identifier).
			Str("type", "name").
			Msg("Identifier detected as session name")
		
		// Verifica cache primeiro
		if cachedID, found := sr.getCachedSessionID(identifier); found {
			logger.WithComponent("middleware").Debug().
				Str("session_name", identifier).
				Str("cached_session_id", cachedID).
				Msg("Session ID resolved from cache")
			return cachedID, nil
		}
		
		// Busca no banco de dados
		session, err := sr.sessionService.GetSession(identifier)
		if err != nil {
			logger.WithComponent("middleware").Warn().
				Err(err).
				Str("session_name", identifier).
				Msg("Failed to resolve session name to ID")
			return "", err
		}
		
		// Adiciona ao cache
		sr.setCachedSessionID(identifier, session.ID)
		
		logger.WithComponent("middleware").Info().
			Str("session_name", identifier).
			Str("resolved_session_id", session.ID).
			Msg("Session name resolved to ID")
		
		return session.ID, nil
	}
	
	// 3. Identificador inválido
	logger.WithComponent("middleware").Warn().
		Str("identifier", identifier).
		Msg("Invalid session identifier format")
	
	return "", &InvalidIdentifierError{Identifier: identifier}
}

// InvalidIdentifierError erro para identificadores inválidos
type InvalidIdentifierError struct {
	Identifier string
}

func (e *InvalidIdentifierError) Error() string {
	return "invalid session identifier: " + e.Identifier
}

// contextKey tipo para chaves do contexto
type contextKey string

const (
	// ResolvedSessionIDKey chave para sessionID resolvido no contexto
	ResolvedSessionIDKey contextKey = "resolved_session_id"
	// OriginalIdentifierKey chave para identificador original no contexto
	OriginalIdentifierKey contextKey = "original_identifier"
)

// Middleware retorna o middleware HTTP para resolução de sessões
func (sr *SessionResolver) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extrai o identificador da URL
			identifier := chi.URLParam(r, "sessionID")
			if identifier == "" {
				// Se não há sessionID na rota, continua normalmente
				next.ServeHTTP(w, r)
				return
			}
			
			// Resolve o identificador
			sessionID, err := sr.resolveSessionIdentifier(identifier)
			if err != nil {
				logger.WithComponent("middleware").Error().
					Err(err).
					Str("identifier", identifier).
					Msg("Failed to resolve session identifier")
				
				http.Error(w, "Invalid session identifier", http.StatusBadRequest)
				return
			}
			
			// Adiciona informações ao contexto
			ctx := context.WithValue(r.Context(), ResolvedSessionIDKey, sessionID)
			ctx = context.WithValue(ctx, OriginalIdentifierKey, identifier)
			
			// Atualiza o parâmetro da URL com o sessionID resolvido
			rctx := chi.RouteContext(ctx)
			if rctx != nil {
				rctx.URLParams.Add("sessionID", sessionID)
			}
			
			logger.WithComponent("middleware").Debug().
				Str("original_identifier", identifier).
				Str("resolved_session_id", sessionID).
				Msg("Session identifier resolved successfully")
			
			// Continua para o próximo handler
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetResolvedSessionID obtém o sessionID resolvido do contexto
func GetResolvedSessionID(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value(ResolvedSessionIDKey).(string)
	return sessionID, ok
}

// GetOriginalIdentifier obtém o identificador original do contexto
func GetOriginalIdentifier(ctx context.Context) (string, bool) {
	identifier, ok := ctx.Value(OriginalIdentifierKey).(string)
	return identifier, ok
}
