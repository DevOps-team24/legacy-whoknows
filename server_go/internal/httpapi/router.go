package httpapi

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	httpSwagger "github.com/swaggo/http-swagger"
)

const SessionName = "session"

type Server struct {
	DB       *sql.DB
	Sessions *sessions.CookieStore
}

func NewRouter(s *Server) http.Handler {
	r := chi.NewRouter()

	r.Use(s.UserFromSession)

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// HTML routes
	r.Get("/", s.ServeRootPage)
	r.Get("/register", s.ServeRegisterPage)
	r.Get("/login", s.ServeLoginPage)

	// API routes
	r.Get("/api/search", s.Search)
	r.Post("/api/register", s.Register)
	r.Post("/api/login", s.Login)
	r.Get("/api/logout", s.Logout)

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	return r
}
