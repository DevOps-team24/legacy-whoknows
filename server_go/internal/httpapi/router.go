package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"

	"whoknows_variations/server_go/internal/queue"
)

const SessionName = "session"

type Server struct {
	DB         *pgxpool.Pool
	Sessions   *sessions.CookieStore
	Queue      *queue.Client
	ScraperKey string
}

func NewRouter(s *Server) http.Handler {
	r := chi.NewRouter()

	r.Use(s.UserFromSession)

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// HTML routes
	r.Get("/", s.ServeRootPage)
	r.Get("/about", s.ServeAboutPage)
	r.Get("/register", s.ServeRegisterPage)
	r.Get("/login", s.ServeLoginPage)

	// API routes
	r.Get("/api/search", s.Search)
	r.Post("/api/register", s.Register)
	r.Post("/api/login", s.Login)
	r.Get("/api/logout", s.Logout)
	r.Post("/api/pages", s.AddPage)

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	return r
}
