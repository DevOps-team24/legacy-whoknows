package httpapi

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	DB *sql.DB
}

func NewRouter(s *Server) http.Handler {
	r := chi.NewRouter()

	// HTML routes (OpenAPI: text/html)
	r.Get("/", s.ServeRootPage)
	r.Get("/register", s.ServeRegisterPage)
	r.Get("/login", s.ServeLoginPage)

	// API routes (OpenAPI: application/json)
	r.Get("/api/search", s.Search)
	r.Post("/api/register", s.Register)
	r.Post("/api/login", s.Login)
	r.Get("/api/logout", s.Logout)

	return r
}

