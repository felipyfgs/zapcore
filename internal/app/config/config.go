package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config representa todas as configurações da aplicação
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Log       LogConfig
	Auth      AuthConfig
	WhatsApp  WhatsAppConfig
	CORS      CORSConfig
	RateLimit RateLimitConfig
	Timeout   TimeoutConfig
	MinIO     MinIOConfig
}

// ServerConfig configurações do servidor HTTP
type ServerConfig struct {
	Port string
	Host string
	Env  string
}

// DatabaseConfig configurações do banco de dados PostgreSQL
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// LogConfig configurações de logging
type LogConfig struct {
	Level         string
	Format        string
	DualOutput    bool
	ConsoleFormat string
	FileFormat    string
	FilePath      string
}

// AuthConfig configurações de autenticação
type AuthConfig struct {
	APIKey string
}

// WhatsAppConfig configurações do WhatsApp
type WhatsAppConfig struct {
	WebhookURL  string
	MediaPath   string
	SessionPath string
}

// CORSConfig configurações de CORS
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// RateLimitConfig configurações de rate limiting
type RateLimitConfig struct {
	Requests int
	Window   time.Duration
}

// TimeoutConfig configurações de timeout
type TimeoutConfig struct {
	Request  time.Duration
	Shutdown time.Duration
}

// MinIOConfig configurações do MinIO
type MinIOConfig struct {
	Enabled         bool
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	DefaultBucket   string
}

// Load carrega as configurações usando Viper
func Load() (*Config, error) {
	// Configurar Viper para ler arquivo .env
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")

	// Permitir variáveis de ambiente (com prioridade sobre arquivo)
	viper.AutomaticEnv()

	// Definir valores padrão primeiro (mas não para API_KEY)
	setDefaults()

	// Tentar ler arquivo de configuração
	if err := viper.ReadInConfig(); err != nil {
		// Arquivo .env não encontrado - usar apenas variáveis de ambiente
	}

	config := &Config{}

	// Configurações do servidor
	config.Server = ServerConfig{
		Port: viper.GetString("PORT"),
		Host: viper.GetString("HOST"),
		Env:  viper.GetString("ENV"),
	}

	// Configurações do banco de dados
	config.Database = DatabaseConfig{
		Host:            viper.GetString("DB_HOST"),
		Port:            viper.GetString("DB_PORT"),
		User:            viper.GetString("DB_USER"),
		Password:        viper.GetString("DB_PASSWORD"),
		Name:            viper.GetString("DB_NAME"),
		SSLMode:         viper.GetString("DB_SSLMODE"),
		MaxOpenConns:    viper.GetInt("DB_MAX_OPEN_CONNS"),
		MaxIdleConns:    viper.GetInt("DB_MAX_IDLE_CONNS"),
		ConnMaxLifetime: viper.GetDuration("DB_CONN_MAX_LIFETIME"),
	}

	// Configurações de log
	config.Log = LogConfig{
		Level:         viper.GetString("LOG_LEVEL"),
		Format:        viper.GetString("LOG_FORMAT"),
		DualOutput:    viper.GetBool("LOG_DUAL_OUTPUT"),
		ConsoleFormat: viper.GetString("LOG_CONSOLE_FORMAT"),
		FileFormat:    viper.GetString("LOG_FILE_FORMAT"),
		FilePath:      viper.GetString("LOG_FILE_PATH"),
	}

	// Configurações de autenticação
	config.Auth = AuthConfig{
		APIKey: viper.GetString("API_KEY"),
	}

	// Configurações do WhatsApp
	config.WhatsApp = WhatsAppConfig{
		WebhookURL:  viper.GetString("WHATSAPP_WEBHOOK_URL"),
		MediaPath:   viper.GetString("WHATSAPP_MEDIA_PATH"),
		SessionPath: viper.GetString("WHATSAPP_SESSION_PATH"),
	}

	// Configurações de CORS
	config.CORS = CORSConfig{
		AllowedOrigins: strings.Split(viper.GetString("CORS_ALLOWED_ORIGINS"), ","),
		AllowedMethods: strings.Split(viper.GetString("CORS_ALLOWED_METHODS"), ","),
		AllowedHeaders: strings.Split(viper.GetString("CORS_ALLOWED_HEADERS"), ","),
	}

	// Configurações de rate limiting
	config.RateLimit = RateLimitConfig{
		Requests: viper.GetInt("RATE_LIMIT_REQUESTS"),
		Window:   viper.GetDuration("RATE_LIMIT_WINDOW"),
	}

	// Configurações de timeout
	config.Timeout = TimeoutConfig{
		Request:  viper.GetDuration("REQUEST_TIMEOUT"),
		Shutdown: viper.GetDuration("SHUTDOWN_TIMEOUT"),
	}

	// Configurações do MinIO
	config.MinIO = MinIOConfig{
		Enabled:         viper.GetBool("MINIO_ENABLED"),
		Endpoint:        viper.GetString("MINIO_ENDPOINT"),
		AccessKeyID:     viper.GetString("MINIO_ACCESS_KEY_ID"),
		SecretAccessKey: viper.GetString("MINIO_SECRET_ACCESS_KEY"),
		UseSSL:          viper.GetBool("MINIO_USE_SSL"),
		DefaultBucket:   viper.GetString("MINIO_DEFAULT_BUCKET"),
	}

	return config, nil
}

// setDefaults define os valores padrão para as configurações
func setDefaults() {
	// Servidor
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("HOST", "localhost")
	viper.SetDefault("ENV", "development")

	// Banco de dados
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "zapcore_user")
	viper.SetDefault("DB_PASSWORD", "zapcore_password")
	viper.SetDefault("DB_NAME", "zapcore_db")
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("DB_MAX_OPEN_CONNS", 25)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 5)
	viper.SetDefault("DB_CONN_MAX_LIFETIME", "300s")

	// Log
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_FORMAT", "json")
	viper.SetDefault("LOG_DUAL_OUTPUT", false)
	viper.SetDefault("LOG_CONSOLE_FORMAT", "console")
	viper.SetDefault("LOG_FILE_FORMAT", "json")
	viper.SetDefault("LOG_FILE_PATH", "./logs/zapcore.log")

	// Autenticação (sem valor padrão para forçar configuração)
	// viper.SetDefault("API_KEY", "your-api-key-for-authentication")

	// WhatsApp
	viper.SetDefault("WHATSAPP_WEBHOOK_URL", "http://localhost:8080/webhook")
	viper.SetDefault("WHATSAPP_MEDIA_PATH", "./media")
	viper.SetDefault("WHATSAPP_SESSION_PATH", "./sessions")

	// CORS
	viper.SetDefault("CORS_ALLOWED_ORIGINS", "*")
	viper.SetDefault("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS")
	viper.SetDefault("CORS_ALLOWED_HEADERS", "Content-Type,Authorization")

	// Rate Limiting
	viper.SetDefault("RATE_LIMIT_REQUESTS", 100)
	viper.SetDefault("RATE_LIMIT_WINDOW", "60s")

	// Timeouts
	viper.SetDefault("REQUEST_TIMEOUT", "30s")
	viper.SetDefault("SHUTDOWN_TIMEOUT", "10s")

	// MinIO
	viper.SetDefault("MINIO_ENABLED", true)
	viper.SetDefault("MINIO_ENDPOINT", "localhost:9000")
	viper.SetDefault("MINIO_ACCESS_KEY_ID", "admin")
	viper.SetDefault("MINIO_SECRET_ACCESS_KEY", "4xN4PEDyxijbN4gM")
	viper.SetDefault("MINIO_USE_SSL", false)
	viper.SetDefault("MINIO_DEFAULT_BUCKET", "zapcore-media")
}

// GetDatabaseDSN retorna a string de conexão do banco de dados
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// GetServerAddress retorna o endereço completo do servidor
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%s", c.Server.Host, c.Server.Port)
}

// IsProduction verifica se está em ambiente de produção
func (c *Config) IsProduction() bool {
	return c.Server.Env == "production"
}

// IsDevelopment verifica se está em ambiente de desenvolvimento
func (c *Config) IsDevelopment() bool {
	return c.Server.Env == "development"
}

// Validate valida se as configurações obrigatórias estão presentes
func (c *Config) Validate() error {
	if c.Auth.APIKey == "" {
		return fmt.Errorf("API_KEY deve ser configurada")
	}

	if c.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD deve ser configurada")
	}

	return nil
}
