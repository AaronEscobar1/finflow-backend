package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func extractPathID(r *http.Request, name string) (int64, error) {
	val := r.PathValue(name)
	if val == "" {
		// Fallback to parsing last segment
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) > 0 {
			val = parts[len(parts)-1]
		}
	}
	id, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("ID inválido: %w", err)
	}
	return id, nil
}

func parseQueryInt64(r *http.Request, key string) *int64 {
	val := r.URL.Query().Get(key)
	if val == "" {
		return nil
	}
	id, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return nil
	}
	return &id
}

func parseQueryBool(r *http.Request, key string) *bool {
	val := r.URL.Query().Get(key)
	if val == "" {
		return nil
	}
	b := val == "true" || val == "1"
	return &b
}

func parseQueryTime(r *http.Request, key string) *time.Time {
	val := r.URL.Query().Get(key)
	if val == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, val)
	if err != nil {
		t, err = time.Parse("2006-01-02", val)
		if err != nil {
			return nil
		}
	}
	return &t
}

func parseQueryString(r *http.Request, key string) *string {
	val := r.URL.Query().Get(key)
	if val == "" {
		return nil
	}
	trimmed := strings.TrimSpace(val)
	return &trimmed
}

func parseQueryInt(r *http.Request, key string, defaultVal int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}
