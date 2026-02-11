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

	// Serve static files from server_go/static
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// HTML routes (text/html)
	r.Get("/", s.ServeRootPage)
	r.Get("/search", s.ServeRootPage) // explicit search page route
	r.Get("/about", s.ServeAboutPage) // about page
	r.Get("/register", s.ServeRegisterPage)
	r.Get("/login", s.ServeLoginPage)

	// API routes (application/json)
	r.Get("/api/search", s.Search)
	r.Post("/api/register", s.Register)
	r.Post("/api/login", s.Login)
	r.Get("/api/logout", s.Logout)

	return r
}

