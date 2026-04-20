package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
)

// newTestPool connects to TEST_DATABASE_URL, applies migrations, and truncates
// data so each test starts with a clean slate.
func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}

	ctx := context.Background()

	// Apply migrations (idempotent thanks to goose_db_version tracking).
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open sql db: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("goose dialect: %v", err)
	}
	if err := goose.Up(sqlDB, migrationsDir()); err != nil {
		t.Fatalf("goose up: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, "TRUNCATE users, pages RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	return pool
}

// migrationsDir resolves the path to the migrations directory relative to the
// repo root, regardless of which package runs the tests.
func migrationsDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations")
}
