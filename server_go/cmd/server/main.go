package main

import (
	"log"
	"net/http"
	"os"

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

	s := &httpapi.Server{DB: conn}
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
