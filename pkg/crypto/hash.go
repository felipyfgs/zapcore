package crypto

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
)

// HashAlgorithm representa os algoritmos de hash suportados
type HashAlgorithm string

const (
	MD5    HashAlgorithm = "md5"
	SHA1   HashAlgorithm = "sha1"
	SHA256 HashAlgorithm = "sha256"
	SHA512 HashAlgorithm = "sha512"
)

// Hash calcula o hash de dados usando o algoritmo especificado
func Hash(data []byte, algorithm HashAlgorithm) (string, error) {
	var hasher hash.Hash

	switch algorithm {
	case MD5:
		hasher = md5.New()
	case SHA1:
		hasher = sha1.New()
	case SHA256:
		hasher = sha256.New()
	case SHA512:
		hasher = sha512.New()
	default:
		return "", fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}

	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// HashString calcula o hash de uma string
func HashString(data string, algorithm HashAlgorithm) (string, error) {
	return Hash([]byte(data), algorithm)
}

// MD5Hash calcula o hash MD5 de dados
func MD5Hash(data []byte) string {
	hash, _ := Hash(data, MD5)
	return hash
}

// SHA256Hash calcula o hash SHA256 de dados
func SHA256Hash(data []byte) string {
	hash, _ := Hash(data, SHA256)
	return hash
}

// SHA512Hash calcula o hash SHA512 de dados
func SHA512Hash(data []byte) string {
	hash, _ := Hash(data, SHA512)
	return hash
}

// VerifyHash verifica se os dados correspondem ao hash fornecido
func VerifyHash(data []byte, expectedHash string, algorithm HashAlgorithm) bool {
	actualHash, err := Hash(data, algorithm)
	if err != nil {
		return false
	}
	return actualHash == expectedHash
}
