package services

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

type DecryptService struct {
	secretKey string
}

func NewDecryptService(secretKey string) *DecryptService {
	return &DecryptService{
		secretKey: secretKey,
	}
}

func (d *DecryptService) Decode(token string) (string, error) {
	secret, err := hex.DecodeString(d.secretKey)
	if err != nil {
		fmt.Printf("Error decoding secret: %v\n", err)
		return "", err
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", errors.New("invalid token")
	}

	b64u := func(s string) ([]byte, error) {
		pad := len(s) % 4
		if pad > 0 {
			s += strings.Repeat("=", 4-pad)
		}
		return base64.URLEncoding.DecodeString(s)
	}

	iv, err := b64u(parts[0])
	if err != nil {
		return "", fmt.Errorf("failed to decode iv: %w", err)
	}

	ciphertext, err := b64u(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	tag, err := b64u(parts[2])
	if err != nil {
		return "", fmt.Errorf("failed to decode tag: %w", err)
	}

	block, err := aes.NewCipher(secret)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	plain, err := gcm.Open(nil, iv, append(ciphertext, tag...), nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plain), nil
}
