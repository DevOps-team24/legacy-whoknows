package db

import (
	"database/sql"
	"os"
)

func ApplyMigrations(db *sql.DB, path string) error {
	// #nosec G304 -- The migration file path is provided by trusted application code, not user input.
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	_, err = db.Exec(string(sqlBytes))
	return err
}
