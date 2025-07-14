package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// Encrypt criptografa dados usando AES-GCM
func Encrypt(data []byte, key []byte) (string, error) {
	// Criar cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Criar GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Gerar nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Criptografar
	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	// Retornar como base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt descriptografa dados usando AES-GCM
func Decrypt(encryptedData string, key []byte) ([]byte, error) {
	// Decodificar base64
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	// Criar cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Criar GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Verificar tamanho mínimo
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extrair nonce e ciphertext
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	// Descriptografar
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptString criptografa uma string
func EncryptString(data string, key []byte) (string, error) {
	return Encrypt([]byte(data), key)
}

// DecryptString descriptografa para string
func DecryptString(encryptedData string, key []byte) (string, error) {
	data, err := Decrypt(encryptedData, key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GenerateKey gera uma chave AES de 32 bytes (256 bits)
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// KeyFromPassword deriva uma chave de 32 bytes a partir de uma senha
func KeyFromPassword(password string) []byte {
	// Usar SHA256 para derivar chave de tamanho fixo
	return []byte(SHA256Hash([]byte(password)))[:32]
}
