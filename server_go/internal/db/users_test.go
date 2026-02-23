package db

import (
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"
)

func setupUsersDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL
		);
	`)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func TestCreateUser_And_GetByUsername(t *testing.T) {
	conn := setupUsersDB(t)
	defer conn.Close()

	err := CreateUser(conn, "alice", "alice@example.com", "abc123hash")
	if err != nil {
		t.Fatal(err)
	}

	u, err := GetUserByUsername(conn, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if u.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", u.Username)
	}
	if u.Email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', got %q", u.Email)
	}
	if u.PasswordHash != "abc123hash" {
		t.Errorf("expected password hash 'abc123hash', got %q", u.PasswordHash)
	}
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	conn := setupUsersDB(t)
	defer conn.Close()

	_, err := GetUserByUsername(conn, "nobody")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGetUserByID(t *testing.T) {
	conn := setupUsersDB(t)
	defer conn.Close()

	_ = CreateUser(conn, "bob", "bob@example.com", "somehash")

	u, err := GetUserByUsername(conn, "bob")
	if err != nil {
		t.Fatal(err)
	}

	u2, err := GetUserByID(conn, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if u2.Username != "bob" {
		t.Errorf("expected username 'bob', got %q", u2.Username)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	conn := setupUsersDB(t)
	defer conn.Close()

	_, err := GetUserByID(conn, 9999)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	conn := setupUsersDB(t)
	defer conn.Close()

	_ = CreateUser(conn, "charlie", "charlie@example.com", "hash1")
	err := CreateUser(conn, "charlie", "other@example.com", "hash2")
	if err == nil {
		t.Error("expected error for duplicate username, got nil")
	}
}
