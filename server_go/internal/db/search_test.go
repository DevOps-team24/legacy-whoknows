package db

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.Exec(`
		CREATE TABLE pages (
			title TEXT PRIMARY KEY,
			url TEXT NOT NULL UNIQUE,
			language TEXT NOT NULL DEFAULT 'en',
			last_updated TIMESTAMP,
			content TEXT NOT NULL
		);
		INSERT INTO pages (title, url, language, content) VALUES
			('Go Programming', '/go', 'en', 'Learn Go'),
			('Python Programming', '/python', 'en', 'Learn Python'),
			('Dansk Søgning', '/dansk', 'da', 'Søg efter noget');
	`)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func TestSearchPages_MatchesTitle(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	results, err := SearchPages(conn, "Go", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0]["title"] != "Go Programming" {
		t.Errorf("expected title 'Go Programming', got %q", results[0]["title"])
	}
}

func TestSearchPages_NoMatch(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	results, err := SearchPages(conn, "Rust", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestSearchPages_EmptyQuery(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	results, err := SearchPages(conn, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 english results for empty query, got %d", len(results))
	}
}

func TestSearchPages_FiltersByLanguage(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	da := "da"
	results, err := SearchPages(conn, "Søgning", &da)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 danish result, got %d", len(results))
	}
	if results[0]["language"] != "da" {
		t.Errorf("expected language 'da', got %q", results[0]["language"])
	}
}

func TestSearchPages_DefaultsToEnglish(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	results, err := SearchPages(conn, "Programming", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 english results, got %d", len(results))
	}
	for _, r := range results {
		if r["language"] != "en" {
			t.Errorf("expected language 'en', got %q", r["language"])
		}
	}
}

func TestSearchPages_PartialMatch(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	results, err := SearchPages(conn, "Prog", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for partial match, got %d", len(results))
	}
}
