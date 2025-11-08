package utils

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"log"
	"regexp"
	"strings"
	"time"

	pasetoware "github.com/gofiber/contrib/paseto"
	"golang.org/x/crypto/bcrypt"
)

func CamelToSnake(s string) string {
	var re = regexp.MustCompile("([a-z0-9])([A-Z])")
	snake := re.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(snake)
}

func CreateToken(privateKeySeed string, username string, expireTime time.Duration) (string, error) {
	privateKey := LoadPrivateKey(privateKeySeed)
	payload := map[string]interface{}{
		"username": username,
	}
	byts, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	encryptedToken, err := pasetoware.CreateToken(privateKey, string(byts), expireTime, pasetoware.PurposePublic)
	return encryptedToken, err
}

func LoadPrivateKey(privateKeySeed string) ed25519.PrivateKey {
	if privateKeySeed == "" {
		log.Fatal("private_key_seed is not set")
	}

	seed, _ := hex.DecodeString(privateKeySeed)
	privateKey := ed25519.NewKeyFromSeed(seed)
	return privateKey
}

// CheckPasswordHash checks if a plain text password matches its bcrypt hash.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
