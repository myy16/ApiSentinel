package envelope

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// Encrypt protects a value with AES-256-GCM. The caller must provide a high-entropy
// application secret through WEBHOOK_SECRET_ENCRYPTION_KEY.
func Encrypt(keyMaterial, plaintext string) (string, error) {
	if keyMaterial == "" {
		return "", fmt.Errorf("WEBHOOK_SECRET_ENCRYPTION_KEY is required")
	}
	block, err := aes.NewCipher(deriveKey(keyMaterial))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return base64.RawStdEncoding.EncodeToString(append(nonce, ciphertext...)), nil
}

func Decrypt(keyMaterial, encoded string) (string, error) {
	if keyMaterial == "" {
		return "", fmt.Errorf("WEBHOOK_SECRET_ENCRYPTION_KEY is required")
	}
	data, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("invalid encrypted secret: %w", err)
	}
	block, err := aes.NewCipher(deriveKey(keyMaterial))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", fmt.Errorf("invalid encrypted secret")
	}
	plaintext, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("could not decrypt secret: %w", err)
	}
	return string(plaintext), nil
}

func deriveKey(keyMaterial string) []byte {
	sum := sha256.Sum256([]byte(keyMaterial))
	return sum[:]
}
