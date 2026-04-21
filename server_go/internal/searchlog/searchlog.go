package searchlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const defaultLogPath = "searches.log"

var mu sync.Mutex

type entry struct {
	Timestamp   string `json:"timestamp"`
	Query       string `json:"query"`
	Language    string `json:"language,omitempty"`
	ResultCount int    `json:"result_count"`
}

func LogSearch(query string, language *string, resultCount int) error {
	path := os.Getenv("WHOKNOWS_SEARCH_LOG_PATH")
	if path == "" {
		path = defaultLogPath
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	record := entry{
		Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
		Query:       query,
		ResultCount: resultCount,
	}
	if language != nil {
		record.Language = strings.TrimSpace(*language)
	}

	mu.Lock()
	defer mu.Unlock()

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	return json.NewEncoder(f).Encode(record)
}
