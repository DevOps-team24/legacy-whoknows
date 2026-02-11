package main

import (
	"log"
	"net/http"

	"whoknows_variations/server_go/internal/db"
	"whoknows_variations/server_go/internal/httpapi"
)

func main() {
	conn, err := db.Open("../whoknows.db")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Hvis du har migrations-funktionen:
	// _ = db.ApplyMigrations(conn, "./migrations/001_init.sql")

	s := &httpapi.Server{DB: conn}
	router := httpapi.NewRouter(s)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
