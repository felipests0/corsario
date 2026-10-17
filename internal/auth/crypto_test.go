package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestHashAndCheckPassword(t *testing.T) {
	password := " VelaMestra#1700 "
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("esperava hash sem erro, obteve: %v", err)
	}
	if hash == "" || hash == password {
		t.Fatal("hash inválido gerado")
	}

	if !CheckPasswordHash(password, hash) {
		t.Error("esperava que CheckPasswordHash retornasse true para a senha correta")
	}

	if CheckPasswordHash("senha-errada", hash) {
		t.Error("esperava que CheckPasswordHash retornasse false para senha incorreta")
	}
}

func TestGenerateSessionToken(t *testing.T) {
	token1, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("erro ao gerar token: %v", err)
	}
	token2, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("erro ao gerar segundo token: %v", err)
	}
	if len(token1) != 64 || token1 == token2 {
		t.Errorf("tokens inválidos ou colidentes: %s / %s", token1, token2)
	}
}

func TestGenerateAndValidateTOTP(t *testing.T) {
	secret, qrBase64, err := GenerateTOTP("capitao")
	if err != nil {
		t.Fatalf("erro inesperado em GenerateTOTP: %v", err)
	}
	if secret == "" {
		t.Fatal("esperava secret TOTP não vazio")
	}
	if !strings.HasPrefix(qrBase64, "data:image/png;base64,") {
		t.Errorf("esperava QR Code em formato data:image/png;base64, obteve prefixo inválido")
	}

	code, err := totp.GenerateCode(secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("erro ao gerar código TOTP de teste: %v", err)
	}

	if !ValidateTOTP(code, secret) {
		t.Error("esperava que ValidateTOTP aceitasse o código recém-gerado")
	}

	if ValidateTOTP("000000", secret) {
		t.Error("esperava que ValidateTOTP recusasse código inválido")
	}
}
