package httpapi

import (
	"encoding/json"
	"net/http"
)

type AuthResponse struct {
	StatusCode *int    `json:"statusCode"`
	Message    *string `json:"message"`
}

type SearchResponse struct {
	Data []map[string]any `json:"data"`
}

type RequestValidationError struct {
	StatusCode int     `json:"statusCode"`
	Message    *string `json:"message"`
}

func (s *Server) ServeRootPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("<html><body><h1>WhoKnows</h1></body></html>"))
}

func (s *Server) ServeRegisterPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("<html><body><h1>Register</h1></body></html>"))
}

func (s *Server) ServeLoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("<html><body><h1>Login</h1></body></html>"))
}

func (s *Server) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		msg := "Missing required query parameter: q"
		resp := RequestValidationError{StatusCode: 422, Message: &msg}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Stub: return empty list for now
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(SearchResponse{Data: []map[string]any{}})
}

func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	msg := "Not implemented yet"
	code := 200
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(AuthResponse{StatusCode: &code, Message: &msg})
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	msg := "Not implemented yet"
	code := 200
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(AuthResponse{StatusCode: &code, Message: &msg})
}

func (s *Server) Logout(w http.ResponseWriter, r *http.Request) {
	msg := "Logged out"
	code := 200
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(AuthResponse{StatusCode: &code, Message: &msg})
}

