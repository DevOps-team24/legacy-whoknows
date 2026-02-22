package db

import (
	"database/sql"
	"strings"
)

func SearchPages(conn *sql.DB, q string, language *string) ([]map[string]any, error) {
	q = strings.TrimSpace(q)
	like := "%" + q + "%"

	// Legacy default: "en"
	lang := "en"
	if language != nil && strings.TrimSpace(*language) != "" {
		lang = strings.TrimSpace(*language)
	}

	rows, err := conn.Query(`
		SELECT title, url, language, last_updated, content
		FROM pages
		WHERE language = ? AND title LIKE ?
		LIMIT 30
	`, lang, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]map[string]any, 0)
	for rows.Next() {
		var title, url, language string
		var lastUpdated sql.NullString
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
		if lastUpdated.Valid {
			row["last_updated"] = lastUpdated.String
		} else {
			row["last_updated"] = nil
		}

		out = append(out, row)
	}

	return out, rows.Err()
}
