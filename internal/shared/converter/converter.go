package converter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// StringToInt converte string para int com valor padrão
func StringToInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	
	value, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	
	return value
}

// StringToInt64 converte string para int64 com valor padrão
func StringToInt64(s string, defaultValue int64) int64 {
	if s == "" {
		return defaultValue
	}
	
	value, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return defaultValue
	}
	
	return value
}

// StringToBool converte string para bool com valor padrão
func StringToBool(s string, defaultValue bool) bool {
	if s == "" {
		return defaultValue
	}
	
	value, err := strconv.ParseBool(s)
	if err != nil {
		return defaultValue
	}
	
	return value
}

// StringToFloat64 converte string para float64 com valor padrão
func StringToFloat64(s string, defaultValue float64) float64 {
	if s == "" {
		return defaultValue
	}
	
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defaultValue
	}
	
	return value
}

// StringToDuration converte string para time.Duration com valor padrão
func StringToDuration(s string, defaultValue time.Duration) time.Duration {
	if s == "" {
		return defaultValue
	}
	
	value, err := time.ParseDuration(s)
	if err != nil {
		return defaultValue
	}
	
	return value
}

// ToJSON converte qualquer valor para JSON string
func ToJSON(v interface{}) (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal to JSON: %w", err)
	}
	return string(bytes), nil
}

// FromJSON converte JSON string para o tipo especificado
func FromJSON(jsonStr string, v interface{}) error {
	if err := json.Unmarshal([]byte(jsonStr), v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return nil
}

// SanitizeString remove espaços e caracteres especiais
func SanitizeString(s string) string {
	return strings.TrimSpace(s)
}

// TruncateString trunca string para o tamanho máximo especificado
func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength]
}

// FormatPhoneNumber formata número de telefone para padrão WhatsApp
func FormatPhoneNumber(phone string) string {
	// Remove todos os caracteres não numéricos
	cleaned := strings.ReplaceAll(phone, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.ReplaceAll(cleaned, "+", "")
	
	// Se não começar com código do país, assume Brasil (55)
	if !strings.HasPrefix(cleaned, "55") && len(cleaned) >= 10 {
		cleaned = "55" + cleaned
	}
	
	return cleaned
}

// BytesToHumanReadable converte bytes para formato legível
func BytesToHumanReadable(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}
