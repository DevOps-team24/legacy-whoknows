package db

import "database/sql"

// InsertPage inserts a scraped page into the database.
// If a page with the same title already exists it is silently skipped.
// ON CONFLICT(title) DO NOTHING works in both SQLite (3.24+) and PostgreSQL.
func InsertPage(conn *sql.DB, title, url, language, content string) error {
	_, err := conn.Exec(`
		INSERT INTO pages (title, url, language, content, last_updated)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT DO NOTHING
	`, title, url, language, content)
	return err
}
