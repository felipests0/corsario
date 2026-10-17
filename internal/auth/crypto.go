package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("erro ao gerar hash da senha: %w", err)
	}
	return string(bytes), nil
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("erro ao gerar token de sessão: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func GenerateTOTP(username string) (secret string, qrCodeBase64 string, err error) {
	account := strings.TrimSpace(username)
	if account == "" {
		account = "capitao"
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Corsario",
		AccountName: account,
	})
	if err != nil {
		return "", "", fmt.Errorf("erro ao gerar chave TOTP: %w", err)
	}

	pngBytes, err := qrcode.Encode(key.URL(), qrcode.Medium, 220)
	if err != nil {
		return "", "", fmt.Errorf("erro ao gerar QR Code TOTP: %w", err)
	}

	qrDataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	return key.Secret(), qrDataURI, nil
}

func ValidateTOTP(token, secret string) bool {
	t := strings.TrimSpace(token)
	s := strings.TrimSpace(secret)
	if t == "" || s == "" {
		return false
	}
	return totp.Validate(t, s)
}
