package db

import (
	"database/sql"
	"os"
)

func ApplyMigrations(db *sql.DB, path string) error {
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	_, err = db.Exec(string(sqlBytes))
	return err
}

