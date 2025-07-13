package service

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"wamex/pkg/logger"
)

// MediaSecurityService implementa validações de segurança para mídia
type MediaSecurityService struct {
	rateLimiter *RateLimiter
	domainWhitelist map[string]bool
	mu sync.RWMutex
}

// RateLimiter implementa rate limiting simples
type RateLimiter struct {
	requests map[string][]time.Time
	mu sync.RWMutex
}

// NewMediaSecurityService cria uma nova instância do serviço de segurança
func NewMediaSecurityService() *MediaSecurityService {
	return &MediaSecurityService{
		rateLimiter: &RateLimiter{
			requests: make(map[string][]time.Time),
		},
		domainWhitelist: getDefaultDomainWhitelist(),
	}
}

// ValidateRateLimit verifica se o IP/sessão não excedeu o limite de requests
func (s *MediaSecurityService) ValidateRateLimit(identifier string, maxRequests int, window time.Duration) error {
	s.rateLimiter.mu.Lock()
	defer s.rateLimiter.mu.Unlock()
	
	now := time.Now()
	windowStart := now.Add(-window)
	
	// Obter requests existentes
	requests := s.rateLimiter.requests[identifier]
	
	// Filtrar requests dentro da janela de tempo
	var validRequests []time.Time
	for _, reqTime := range requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}
	
	// Verificar limite
	if len(validRequests) >= maxRequests {
		logger.WithComponent("media-security").Warn().
			Str("identifier", identifier).
			Int("requests", len(validRequests)).
			Int("max_requests", maxRequests).
			Dur("window", window).
			Msg("Rate limit excedido")
		return fmt.Errorf("rate limit excedido: %d requests em %v", len(validRequests), window)
	}
	
	// Adicionar request atual
	validRequests = append(validRequests, now)
	s.rateLimiter.requests[identifier] = validRequests
	
	return nil
}

// ValidateDomain verifica se o domínio está na whitelist
func (s *MediaSecurityService) ValidateDomain(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("URL inválida: %w", err)
	}
	
	domain := strings.ToLower(parsedURL.Hostname())
	
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Verificar domínio exato
	if s.domainWhitelist[domain] {
		return nil
	}
	
	// Verificar subdomínios
	for allowedDomain := range s.domainWhitelist {
		if strings.HasSuffix(domain, "."+allowedDomain) {
			return nil
		}
	}
	
	logger.WithComponent("media-security").Warn().
		Str("domain", domain).
		Str("url", urlStr).
		Msg("Domínio não está na whitelist")
	
	return fmt.Errorf("domínio %s não está na whitelist", domain)
}

// ValidatePrivateIP verifica se o IP não é privado/local
func (s *MediaSecurityService) ValidatePrivateIP(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("URL inválida: %w", err)
	}
	
	host := parsedURL.Hostname()
	
	// Resolver IP se for hostname
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("erro ao resolver hostname %s: %w", host, err)
	}
	
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("acesso a IP privado não permitido: %s", ip.String())
		}
	}
	
	return nil
}

// ValidateFileSize verifica se o tamanho do arquivo está dentro dos limites
func (s *MediaSecurityService) ValidateFileSize(size int64, maxSize int64) error {
	if size > maxSize {
		return fmt.Errorf("arquivo muito grande: %d bytes (máximo: %d bytes)", size, maxSize)
	}
	return nil
}

// ValidateFileExtension verifica se a extensão é permitida
func (s *MediaSecurityService) ValidateFileExtension(filename string) error {
	ext := strings.ToLower(filename[strings.LastIndex(filename, "."):])
	
	allowedExtensions := []string{
		".jpg", ".jpeg", ".png", ".gif", ".webp",
		".mp3", ".ogg", ".aac", ".amr", ".wav",
		".mp4", ".3gp",
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt",
	}
	
	for _, allowed := range allowedExtensions {
		if ext == allowed {
			return nil
		}
	}
	
	return fmt.Errorf("extensão %s não permitida", ext)
}

// AddDomainToWhitelist adiciona um domínio à whitelist
func (s *MediaSecurityService) AddDomainToWhitelist(domain string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.domainWhitelist[strings.ToLower(domain)] = true
	
	logger.WithComponent("media-security").Info().
		Str("domain", domain).
		Msg("Domínio adicionado à whitelist")
}

// RemoveDomainFromWhitelist remove um domínio da whitelist
func (s *MediaSecurityService) RemoveDomainFromWhitelist(domain string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.domainWhitelist, strings.ToLower(domain))
	
	logger.WithComponent("media-security").Info().
		Str("domain", domain).
		Msg("Domínio removido da whitelist")
}

// GetWhitelistedDomains retorna a lista de domínios na whitelist
func (s *MediaSecurityService) GetWhitelistedDomains() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	domains := make([]string, 0, len(s.domainWhitelist))
	for domain := range s.domainWhitelist {
		domains = append(domains, domain)
	}
	
	return domains
}

// CleanupRateLimit remove entradas antigas do rate limiter
func (s *MediaSecurityService) CleanupRateLimit(maxAge time.Duration) {
	s.rateLimiter.mu.Lock()
	defer s.rateLimiter.mu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	
	for identifier, requests := range s.rateLimiter.requests {
		var validRequests []time.Time
		for _, reqTime := range requests {
			if reqTime.After(cutoff) {
				validRequests = append(validRequests, reqTime)
			}
		}
		
		if len(validRequests) == 0 {
			delete(s.rateLimiter.requests, identifier)
		} else {
			s.rateLimiter.requests[identifier] = validRequests
		}
	}
}

// getDefaultDomainWhitelist retorna a whitelist padrão de domínios
func getDefaultDomainWhitelist() map[string]bool {
	return map[string]bool{
		// Serviços de imagem
		"imgur.com": true,
		"i.imgur.com": true,
		
		// GitHub
		"github.com": true,
		"raw.githubusercontent.com": true,
		
		// Serviços de armazenamento
		"dropbox.com": true,
		"dl.dropboxusercontent.com": true,
		"drive.google.com": true,
		"docs.google.com": true,
		"onedrive.live.com": true,
		
		// AWS/CloudFront
		"amazonaws.com": true,
		"s3.amazonaws.com": true,
		"cloudfront.net": true,
		
		// Discord
		"cdn.discordapp.com": true,
		"media.discordapp.net": true,
		
		// Telegram
		"telegram.org": true,
		"t.me": true,
		
		// MinIO (servidor do projeto)
		"minio.resolvecert.com": true,
	}
}

// isPrivateIP verifica se um IP é privado/local
func isPrivateIP(ip net.IP) bool {
	// IPv4 private ranges
	private4 := []net.IPNet{
		{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
		{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},
		{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)},
		{IP: net.IPv4(127, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
	}
	
	// IPv6 private ranges
	private6 := []net.IPNet{
		{IP: net.ParseIP("::1"), Mask: net.CIDRMask(128, 128)}, // localhost
		{IP: net.ParseIP("fc00::"), Mask: net.CIDRMask(7, 128)}, // unique local
		{IP: net.ParseIP("fe80::"), Mask: net.CIDRMask(10, 128)}, // link local
	}
	
	if ip.To4() != nil {
		for _, private := range private4 {
			if private.Contains(ip) {
				return true
			}
		}
	} else {
		for _, private := range private6 {
			if private.Contains(ip) {
				return true
			}
		}
	}
	
	return false
}
