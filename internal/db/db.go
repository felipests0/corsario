package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB(filepath string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite3", filepath)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir conexão com SQLite: %w", err)
	}

	conn.SetMaxOpenConns(1)

	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA busy_timeout=5000;",
	}

	for _, pragma := range pragmas {
		if _, err := conn.Exec(pragma); err != nil {
			conn.Close()
			return nil, fmt.Errorf("erro ao executar %s: %w", pragma, err)
		}
	}

	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY,
		username TEXT UNIQUE,
		password_hash TEXT,
		realm TEXT,
		totp_secret TEXT
	);`

	if _, err := conn.Exec(createUsersTable); err != nil {
		conn.Close()
		return nil, fmt.Errorf("erro ao criar tabela users: %w", err)
	}

	DB = conn
	return conn, nil
}
