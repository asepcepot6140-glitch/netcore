package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

func encryptionKey() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters for encryption")
	}

	key := sha256.Sum256([]byte(secret))
	return key[:], nil
}

func EncryptMikrotikPassword(password string) (string, error) {
	key, err := encryptionKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
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

	encrypted := gcm.Seal(nonce, nonce, []byte(password), nil)
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func DecryptMikrotikPassword(encryptedPassword string) (string, error) {
	key, err := encryptionKey()
	if err != nil {
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(encryptedPassword)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(data) < gcm.NonceSize()+gcm.Overhead() {
		return "", fmt.Errorf("invalid encrypted data")
	}

	nonce := data[:gcm.NonceSize()]
	ciphertext := data[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt password")
	}

	return string(plaintext), nil
}
