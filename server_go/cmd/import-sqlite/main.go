// Command import-sqlite copies data from a legacy SQLite whoknows.db into
// the configured Postgres database. It preserves primary keys and bumps
// sequences afterwards. Safe to run multiple times (uses ON CONFLICT DO NOTHING).
package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "modernc.org/sqlite"
)

func main() {
	sqlitePath := flag.String("sqlite", "whoknows.db", "path to the legacy SQLite file")
	flag.Parse()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()

	src, err := sql.Open("sqlite", *sqlitePath+"?mode=ro")
	if err != nil {
		log.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = src.Close() }()

	dst, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer dst.Close()

	if err := importUsers(ctx, src, dst); err != nil {
		log.Fatalf("import users: %v", err)
	}
	if err := importPages(ctx, src, dst); err != nil {
		log.Fatalf("import pages: %v", err)
	}
	if err := resetUserSequence(ctx, dst); err != nil {
		log.Fatalf("reset users sequence: %v", err)
	}

	log.Println("import complete")
}

func importUsers(ctx context.Context, src *sql.DB, dst *pgxpool.Pool) error {
	rows, err := src.QueryContext(ctx, "SELECT id, username, email, password FROM users")
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	batch := &pgx.Batch{}
	count := 0
	for rows.Next() {
		var id int64
		var username, email, password string
		if err := rows.Scan(&id, &username, &email, &password); err != nil {
			return err
		}
		batch.Queue(
			"INSERT INTO users (id, username, email, password) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO NOTHING",
			id, username, email, password,
		)
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if count == 0 {
		log.Println("users: nothing to import")
		return nil
	}

	br := dst.SendBatch(ctx, batch)
	defer func() { _ = br.Close() }()
	for i := 0; i < count; i++ {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	log.Printf("users: imported %d rows", count)
	return nil
}

func importPages(ctx context.Context, src *sql.DB, dst *pgxpool.Pool) error {
	rows, err := src.QueryContext(ctx, "SELECT title, url, language, last_updated, content FROM pages")
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	batch := &pgx.Batch{}
	count := 0
	for rows.Next() {
		var title, url, language, content string
		var lastUpdatedStr sql.NullString
		if err := rows.Scan(&title, &url, &language, &lastUpdatedStr, &content); err != nil {
			return err
		}

		var lastUpdated *time.Time
		if lastUpdatedStr.Valid && lastUpdatedStr.String != "" {
			if t, err := parseSQLiteTimestamp(lastUpdatedStr.String); err == nil {
				lastUpdated = &t
			}
		}

		batch.Queue(
			`INSERT INTO pages (title, url, language, last_updated, content)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (title) DO NOTHING`,
			title, url, language, lastUpdated, content,
		)
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if count == 0 {
		log.Println("pages: nothing to import")
		return nil
	}

	br := dst.SendBatch(ctx, batch)
	defer func() { _ = br.Close() }()
	for i := 0; i < count; i++ {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	log.Printf("pages: imported %d rows", count)
	return nil
}

// resetUserSequence bumps the users_id_seq so future inserts don't collide
// with the ids we copied over.
func resetUserSequence(ctx context.Context, dst *pgxpool.Pool) error {
	_, err := dst.Exec(ctx,
		`SELECT setval('users_id_seq', COALESCE((SELECT MAX(id) FROM users), 1))`,
	)
	return err
}

func parseSQLiteTimestamp(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	var firstErr error
	for _, layout := range layouts {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return time.Time{}, firstErr
}
