package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func seedPages(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	rows := []struct {
		title, url, language, content string
	}{
		{"Go Programming", "/go", "en", "Learn Go"},
		{"Python Programming", "/python", "en", "Learn Python"},
		{"Dansk Søgning", "/dansk", "da", "Søg efter noget"},
	}
	for _, r := range rows {
		if _, err := pool.Exec(ctx,
			`INSERT INTO pages (title, url, language, content) VALUES ($1, $2, $3, $4)`,
			r.title, r.url, r.language, r.content,
		); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSearchPages_MatchesTitle(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)
	seedPages(t, pool)

	results, err := SearchPages(ctx, pool, "Go", nil)
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
	ctx := context.Background()
	pool := newTestPool(t)
	seedPages(t, pool)

	results, err := SearchPages(ctx, pool, "Rust", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestSearchPages_EmptyQuery(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)
	seedPages(t, pool)

	results, err := SearchPages(ctx, pool, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 english results for empty query, got %d", len(results))
	}
}

func TestSearchPages_FiltersByLanguage(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)
	seedPages(t, pool)

	da := "da"
	results, err := SearchPages(ctx, pool, "Søgning", &da)
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
	ctx := context.Background()
	pool := newTestPool(t)
	seedPages(t, pool)

	results, err := SearchPages(ctx, pool, "Programming", nil)
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
	ctx := context.Background()
	pool := newTestPool(t)
	seedPages(t, pool)

	results, err := SearchPages(ctx, pool, "Prog", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for partial match, got %d", len(results))
	}
}
