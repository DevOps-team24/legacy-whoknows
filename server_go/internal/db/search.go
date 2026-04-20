package db

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func SearchPages(ctx context.Context, conn *pgxpool.Pool, q string, language *string) ([]map[string]any, error) {
	q = strings.TrimSpace(q)
	like := "%" + q + "%"

	// Legacy default: "en"
	lang := "en"
	if language != nil && strings.TrimSpace(*language) != "" {
		lang = strings.TrimSpace(*language)
	}

	rows, err := conn.Query(ctx, `
		SELECT title, url, language, last_updated, content
		FROM pages
		WHERE language = $1 AND title ILIKE $2
		LIMIT 30
	`, lang, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]map[string]any, 0)
	for rows.Next() {
		var title, url, language string
		var lastUpdated *time.Time
		var content string

		if err := rows.Scan(&title, &url, &language, &lastUpdated, &content); err != nil {
			return nil, err
		}

		row := map[string]any{
			"title":    title,
			"url":      url,
			"language": language,
			"content":  content,
		}
		if lastUpdated != nil {
			row["last_updated"] = lastUpdated.Format(time.RFC3339)
		} else {
			row["last_updated"] = nil
		}

		out = append(out, row)
	}

	return out, rows.Err()
}
