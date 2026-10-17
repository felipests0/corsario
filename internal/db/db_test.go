package db

import "testing"

func TestInitDB(t *testing.T) {
	conn, err := InitDB("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("esperava inicialização sem erros, mas obteve: %v", err)
	}
	defer conn.Close()

	var tableName string
	err = conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users';").Scan(&tableName)
	if err != nil {
		t.Fatalf("erro ao verificar existência da tabela users: %v", err)
	}
	if tableName != "users" {
		t.Errorf("esperava tabela 'users', mas obteve '%s'", tableName)
	}
}
