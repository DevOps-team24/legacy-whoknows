package httpapi

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sync"
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

type User struct {
	Username string
}

type ViewData struct {
	User    *User
	Flashes []string
	Results []map[string]any
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
	// pass an empty ViewData for now; populate User/Flashes/Results as you implement auth/search logic
	renderTemplate(w, "search.html", ViewData{
		User:    nil,
		Flashes: nil,
		Results: nil,
	})
}

func (s *Server) ServeSearchPage(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "search.html", ViewData{})
}

func (s *Server) ServeAboutPage(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "about.html", ViewData{})
}

func (s *Server) ServeRegisterPage(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "register.html", ViewData{
		User:    nil,
		Flashes: nil,
	})
}

func (s *Server) ServeLoginPage(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "login.html", ViewData{
		User:    nil,
		Flashes: nil,
	})
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
