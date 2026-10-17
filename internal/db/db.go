package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

type User struct {
	ID           int
	Username     string
	PasswordHash string
	Realm        string
	TOTPSecret   string
}

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

	createSessionsTable := `
	CREATE TABLE IF NOT EXISTS sessions (
		token TEXT PRIMARY KEY,
		user_id INTEGER,
		expires_at DATETIME
	);`

	if _, err := conn.Exec(createSessionsTable); err != nil {
		conn.Close()
		return nil, fmt.Errorf("erro ao criar tabela sessions: %w", err)
	}

	DB = conn
	return conn, nil
}

func CreateAdminUser(conn *sql.DB, username, passwordHash string) error {
	query := `
	INSERT INTO users (username, password_hash, realm, totp_secret)
	VALUES (?, ?, 'interno', '')
	ON CONFLICT(username) DO UPDATE SET
		password_hash = excluded.password_hash,
		realm = 'interno';`
	_, err := conn.Exec(query, username, passwordHash)
	if err != nil {
		return fmt.Errorf("erro ao criar usuário admin: %w", err)
	}
	return nil
}

func ListUsers() ([]User, error) {
	rows, err := DB.Query(`SELECT id, username, realm FROM users ORDER BY id ASC;`)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar utilizadores: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Realm); err != nil {
			return nil, fmt.Errorf("erro ao ler utilizador: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func CreateUser(username, passwordHash, realm string) error {
	if realm == "" {
		realm = "interno"
	}
	query := `INSERT INTO users (username, password_hash, realm, totp_secret) VALUES (?, ?, ?, '');`
	if _, err := DB.Exec(query, username, passwordHash, realm); err != nil {
		return fmt.Errorf("erro ao criar utilizador: %w", err)
	}
	return nil
}

func DeleteUser(id int) error {
	_, _ = DB.Exec(`DELETE FROM sessions WHERE user_id = ?;`, id)
	if _, err := DB.Exec(`DELETE FROM users WHERE id = ?;`, id); err != nil {
		return fmt.Errorf("erro ao remover utilizador: %w", err)
	}
	return nil
}

func CountUsers() (int, error) {
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM users;`).Scan(&count); err != nil {
		return 0, fmt.Errorf("erro ao contar utilizadores: %w", err)
	}
	return count, nil
}

func UpdateUserTOTPSecret(userID int, secret string) error {
	query := `UPDATE users SET totp_secret = ? WHERE id = ?;`
	if _, err := DB.Exec(query, secret, userID); err != nil {
		return fmt.Errorf("erro ao atualizar segredo TOTP: %w", err)
	}
	return nil
}

func GetUserByID(userID int) (*User, error) {
	var u User
	query := `SELECT id, username, password_hash, realm, COALESCE(totp_secret, '') FROM users WHERE id = ?;`
	err := DB.QueryRow(query, userID).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Realm, &u.TOTPSecret)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func GetUserByUsernameAndRealm(conn *sql.DB, username, realm string) (*User, error) {
	var u User
	query := `SELECT id, username, password_hash, realm, COALESCE(totp_secret, '') FROM users WHERE username = ? AND realm = ?;`
	err := conn.QueryRow(query, username, realm).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Realm, &u.TOTPSecret)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func CreateSession(conn *sql.DB, token string, userID int, expiresAt time.Time) error {
	query := `INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?);`
	_, err := conn.Exec(query, token, userID, expiresAt.UTC())
	if err != nil {
		return fmt.Errorf("erro ao salvar sessão: %w", err)
	}
	return nil
}

func ValidateSession(conn *sql.DB, token string) (int, bool) {
	if token == "" {
		return 0, false
	}
	var userID int
	var expiresAt time.Time
	query := `SELECT user_id, expires_at FROM sessions WHERE token = ?;`
	err := conn.QueryRow(query, token).Scan(&userID, &expiresAt)
	if err != nil {
		return 0, false
	}
	if time.Now().UTC().After(expiresAt.UTC()) {
		_, _ = conn.Exec(`DELETE FROM sessions WHERE token = ?;`, token)
		return 0, false
	}
	return userID, true
}
