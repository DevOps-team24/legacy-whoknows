package db

import (
	"database/sql"
	"errors"
)

type UserRow struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
}

var ErrUserNotFound = errors.New("user not found")

func GetUserByUsername(conn *sql.DB, username string) (*UserRow, error) {
	row := conn.QueryRow(
		"SELECT id, username, email, password FROM users WHERE username = ?",
		username,
	)

	u := &UserRow{}
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return u, nil
}

func GetUserByID(conn *sql.DB, id int64) (*UserRow, error) {
	row := conn.QueryRow(
		"SELECT id, username, email, password FROM users WHERE id = ?",
		id,
	)

	u := &UserRow{}
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return u, nil
}

func CreateUser(conn *sql.DB, username, email, passwordHash string) error {
	_, err := conn.Exec(
		"INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
		username, email, passwordHash,
	)
	return err
}
