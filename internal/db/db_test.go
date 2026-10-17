package db

import (
	"testing"
	"time"
)

func TestInitDB(t *testing.T) {
	conn, err := InitDB("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("esperava inicialização sem erros, mas obteve: %v", err)
	}
	defer conn.Close()

	for _, expectedTable := range []string{"users", "sessions"} {
		var tableName string
		err = conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", expectedTable).Scan(&tableName)
		if err != nil {
			t.Fatalf("erro ao verificar existência da tabela %s: %v", expectedTable, err)
		}
		if tableName != expectedTable {
			t.Errorf("esperava tabela '%s', mas obteve '%s'", expectedTable, tableName)
		}
	}

	if err := CreateAdminUser(conn, "capitao", "hash123"); err != nil {
		t.Fatalf("erro ao criar usuário admin: %v", err)
	}

	u, err := GetUserByUsernameAndRealm(conn, "capitao", "interno")
	if err != nil || u.Username != "capitao" {
		t.Fatalf("erro ao buscar usuário admin: %v", err)
	}

	expires := time.Now().UTC().Add(1 * time.Hour)
	if err := CreateSession(conn, "tok-valido", u.ID, expires); err != nil {
		t.Fatalf("erro ao criar sessão: %v", err)
	}

	if uid, ok := ValidateSession(conn, "tok-valido"); !ok || uid != u.ID {
		t.Errorf("esperava sessão válida para user %d, obteve uid=%d ok=%v", u.ID, uid, ok)
	}

	expired := time.Now().UTC().Add(-1 * time.Hour)
	_ = CreateSession(conn, "tok-expirado", u.ID, expired)
	if _, ok := ValidateSession(conn, "tok-expirado"); ok {
		t.Error("esperava sessão expirada inválida")
	}
}
