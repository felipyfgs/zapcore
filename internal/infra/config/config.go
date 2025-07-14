package configs

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config representa a configuração global da aplicação
type Config struct {
	Environment string         `json:"environment"`
	Server      ServerConfig   `json:"server"`
	Database    DatabaseConfig `json:"database"`
	WhatsApp    WhatsAppConfig `json:"whatsapp"`
	Logger      LoggerConfig   `json:"logger"`
}

// ServerConfig configurações do servidor HTTP
type ServerConfig struct {
	Port         string `json:"port"`
	Host         string `json:"host"`
	ReadTimeout  int    `json:"read_timeout"`
	WriteTimeout int    `json:"write_timeout"`
	IdleTimeout  int    `json:"idle_timeout"`
}

// DatabaseConfig configurações do banco de dados
type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"name"`
	SSLMode  string `json:"ssl_mode"`
}

// WhatsAppConfig configurações do WhatsApp
type WhatsAppConfig struct {
	Debug       bool   `json:"debug"`
	LogLevel    string `json:"log_level"`
	QRTimeout   int    `json:"qr_timeout"`
	MaxSessions int    `json:"max_sessions"`
}

// LoggerConfig configurações do logger
type LoggerConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
}

// Load carrega as configurações da aplicação
func Load() (*Config, error) {
	// Carrega arquivo .env se existir
	if err := godotenv.Load(); err != nil {
		// Não é um erro crítico se o arquivo .env não existir
		fmt.Println("Warning: .env file not found, using environment variables")
	}

	config := &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			Host:         getEnv("HOST", "0.0.0.0"),
			ReadTimeout:  getEnvAsInt("READ_TIMEOUT", 15),
			WriteTimeout: getEnvAsInt("WRITE_TIMEOUT", 15),
			IdleTimeout:  getEnvAsInt("IDLE_TIMEOUT", 60),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "wamex"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		WhatsApp: WhatsAppConfig{
			Debug:       getEnvAsBool("WA_DEBUG", false),
			LogLevel:    getEnv("WA_LOG_LEVEL", "INFO"),
			QRTimeout:   getEnvAsInt("WA_QR_TIMEOUT", 60),
			MaxSessions: getEnvAsInt("WA_MAX_SESSIONS", 100),
		},
		Logger: LoggerConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	return config, nil
}

// DSN retorna a string de conexão do banco de dados
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode)
}

// IsProduction verifica se está em ambiente de produção
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment verifica se está em ambiente de desenvolvimento
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// getEnv obtém variável de ambiente com valor padrão
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt obtém variável de ambiente como inteiro
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBool obtém variável de ambiente como boolean
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
