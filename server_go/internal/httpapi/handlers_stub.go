package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"whoknows_variations/server_go/internal/auth"
	"whoknows_variations/server_go/internal/db"
)

type contextKey string

const userContextKey contextKey = "user"

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

type User struct {
	ID       int64
	Username string
	Email    string
}

type ViewData struct {
	User    *User
	Flashes []string
	Error   string
	Results []map[string]any
	Query   string
}

// UserFromSession is chi middleware that loads the logged-in user (if any)
// from the session cookie and stores it in the request context.
func (s *Server) UserFromSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, _ := s.Sessions.Get(r, SessionName)
		uid, ok := sess.Values["user_id"]
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		userID, ok := uid.(int64)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		row, err := db.GetUserByID(s.DB, userID)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		u := &User{ID: row.ID, Username: row.Username, Email: row.Email}
		ctx := context.WithValue(r.Context(), userContextKey, u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func currentUser(r *http.Request) *User {
	u, _ := r.Context().Value(userContextKey).(*User)
	return u
}

func (s *Server) addFlash(w http.ResponseWriter, r *http.Request, msg string) {
	sess, _ := s.Sessions.Get(r, SessionName)
	sess.AddFlash(msg)
	_ = sess.Save(r, w)
}

func (s *Server) getFlashes(w http.ResponseWriter, r *http.Request) []string {
	sess, _ := s.Sessions.Get(r, SessionName)
	raw := sess.Flashes()
	if len(raw) == 0 {
		return nil
	}
	_ = sess.Save(r, w)
	out := make([]string, len(raw))
	for i, v := range raw {
		out[i], _ = v.(string)
	}
	return out
}

var (
	// per-page template cache
	pageTemplates   = map[string]*template.Template{}
	pageTemplatesMu sync.RWMutex

	// resolved template directory (cached)
	foundTemplateDir    string
	foundTemplateDirMu  sync.Once
	foundTemplateDirErr error
)

func findTemplateDir() (string, error) {
	foundTemplateDirMu.Do(func() {
		candidates := []string{
			"./templates",
			"templates",
			"../templates",
			"./server_go/templates",
			"server_go/templates",
			"../server_go/templates",
		}
		for _, c := range candidates {
			if info, err := os.Stat(c); err == nil && info.IsDir() {
				foundTemplateDir = c
				break
			}
			_, _ = os.Stderr.WriteString("templates: checked candidate '" + c + "' -> not found\n")
		}
		if foundTemplateDir == "" {
			foundTemplateDirErr = os.ErrNotExist
			_, _ = os.Stderr.WriteString("templates: no templates directory found among candidates\n")
		}
	})
	return foundTemplateDir, foundTemplateDirErr
}

func loadTemplateFor(pageFilename string) (*template.Template, error) {
	// cache fast-path
	pageTemplatesMu.RLock()
	t := pageTemplates[pageFilename]
	pageTemplatesMu.RUnlock()
	if t != nil {
		return t, nil
	}

	// build and cache
	dir, err := findTemplateDir()
	if err != nil {
		return nil, err
	}

	layoutPath := filepath.Join(dir, "layout.html")
	pagePath := filepath.Join(dir, pageFilename)

	// parse layout first, then page so page definitions override blocks
	t, err = template.ParseFiles(layoutPath, pagePath)
	if err != nil {
		_, _ = os.Stderr.WriteString("templates: ParseFiles error for " + pageFilename + ": " + err.Error() + "\n")
		return nil, err
	}

	pageTemplatesMu.Lock()
	pageTemplates[pageFilename] = t
	pageTemplatesMu.Unlock()
	return t, nil
}

// render helper — parses layout + requested page, executes the page template
func renderTemplate(w http.ResponseWriter, name string, data any) {
	t, err := loadTemplateFor(name)
	if err != nil {
		_, _ = os.Stderr.WriteString("templates load error: " + err.Error() + "\n")
		// fallback if templates failed to load
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><head></head><body><h1>WhoKnows</h1></body></html>"))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// First try to execute the page's own template (e.g. "about.html")
	if execErr := t.ExecuteTemplate(w, name, data); execErr != nil {
		// fallback to executing layout directly
		_, _ = os.Stderr.WriteString("templates: ExecuteTemplate(" + name + ") error: " + execErr.Error() + "\n")
		if execErr2 := t.ExecuteTemplate(w, "layout", data); execErr2 != nil {
			_, _ = os.Stderr.WriteString("templates: ExecuteTemplate(layout) error: " + execErr2.Error() + "\n")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("<html><body><h1>Template execution error</h1></body></html>"))
		}
	}
}

func (s *Server) ServeRootPage(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	langParam := strings.TrimSpace(r.URL.Query().Get("language"))
	var lang *string
	if langParam != "" {
		lang = &langParam
	}

	var results []map[string]any
	if q != "" {
		var err error
		results, err = db.SearchPages(s.DB, q, lang)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	renderTemplate(w, "search.html", ViewData{
		User:    currentUser(r),
		Flashes: s.getFlashes(w, r),
		Results: results,
		Query:   q,
	})
}

func (s *Server) ServeSearchPage(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "search.html", ViewData{User: currentUser(r), Flashes: s.getFlashes(w, r)})
}

func (s *Server) ServeAboutPage(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "about.html", ViewData{User: currentUser(r), Flashes: s.getFlashes(w, r)})
}

func (s *Server) ServeRegisterPage(w http.ResponseWriter, r *http.Request) {
	if currentUser(r) != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	renderTemplate(w, "register.html", ViewData{Flashes: s.getFlashes(w, r)})
}

func (s *Server) ServeLoginPage(w http.ResponseWriter, r *http.Request) {
	if currentUser(r) != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	renderTemplate(w, "login.html", ViewData{Flashes: s.getFlashes(w, r)})
}

func (s *Server) Search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		// Match legacy behaviour: empty query simply returns empty result set with 200 OK
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(SearchResponse{Data: []map[string]any{}})
		return
	}

	langParam := r.URL.Query().Get("language")
	var lang *string
	if langParam != "" {
		lang = &langParam
	}

	results, err := db.SearchPages(s.DB, q, lang)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(SearchResponse{Data: results})
}

func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	if currentUser(r) != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	password2 := r.FormValue("password2")

	var formError string
	switch {
	case username == "":
		formError = "You have to enter a username"
	case email == "" || !strings.Contains(email, "@"):
		formError = "You have to enter a valid email address"
	case password == "":
		formError = "You have to enter a password"
	case password != password2:
		formError = "The two passwords do not match"
	default:
		_, err := db.GetUserByUsername(s.DB, username)
		if err == nil {
			formError = "The username is already taken"
		} else if !errors.Is(err, db.ErrUserNotFound) {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	if formError != "" {
		renderTemplate(w, "register.html", ViewData{Error: formError})
		return
	}

	hash := auth.HashPassword(password)
	if err := db.CreateUser(s.DB, username, email, hash); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	s.addFlash(w, r, "You were successfully registered and can login now")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
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
