package auth

import "testing"

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
