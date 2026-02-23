package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"

	"whoknows_variations/server_go/internal/db"
	"whoknows_variations/server_go/internal/httpapi"
)

func main() {
	dbPath := os.Getenv("WHOKNOWS_DB_PATH")
	if dbPath == "" {
		dbPath = "./whoknows.db"
	}

	conn, err := db.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

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

	log.Printf("listening on %s:%s", addr, port)
	log.Fatal(http.ListenAndServe(addr+":"+port, router))
}
