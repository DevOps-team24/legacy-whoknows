package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/sessions"

	_ "whoknows_variations/server_go/docs"
	"whoknows_variations/server_go/internal/db"
	"whoknows_variations/server_go/internal/httpapi"
)

// @title WhoKnows API
// @version 1.0
// @description API for the WhoKnows search application
// @host huw.dk
// @BasePath /
func main() {
	workingDir, _ := os.Getwd()
	dbPath := os.Getenv("WHOKNOWS_DB_PATH")
	if dbPath == "" {
		absPath, _ := filepath.Abs(filepath.Join(workingDir, "..", "whoknows.db"))
		dbPath = absPath
	}

	conn, err := db.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("close db connection failed: %v", err)
		}
	}()

	secretKey := os.Getenv("WHOKNOWS_SECRET_KEY")
	if secretKey == "" {
		secretKey = "default-secret-change-me"
	}
	store := sessions.NewCookieStore([]byte(secretKey))

	s := &httpapi.Server{DB: conn, Sessions: store}
	router := httpapi.NewRouter(s)

	port := os.Getenv("WHOKNOWS_PORT")
	if port == "" {
		port = "8080"
	}
	addr := os.Getenv("WHOKNOWS_ADDR")
	if addr == "" {
		addr = "0.0.0.0"
	}
	log.Printf("Connecting to DB at: %s", sanitizeLogValue(dbPath))                  // #nosec G706 -- Value is newline-sanitized before logging; source is deployment configuration.
	log.Printf("listening on %s:%s", sanitizeLogValue(addr), sanitizeLogValue(port)) // #nosec G706 -- Values are newline-sanitized before logging; sources are deployment configuration.

	srv := &http.Server{
		Addr:              addr + ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func sanitizeLogValue(value string) string {
	value = strings.ReplaceAll(value, "\r", "")
	return strings.ReplaceAll(value, "\n", "")
}
