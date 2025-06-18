package server

import (
	"net/http"
	"strings"
)

func puzzlePathMiddleware(puzzles []string, next http.Handler) http.Handler {
	m := map[string]string{}
	for _, puzzle := range puzzles {
		key := puzzle
		key = strings.ReplaceAll(key, "-", "")
		key = strings.ToLower(key)
		m[key] = "/" + puzzle + "/"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path
		key = strings.TrimPrefix(key, "/")
		key = strings.TrimSuffix(key, "/")
		key = strings.ReplaceAll(key, "-", "")
		key = strings.ToLower(key)

		if path, ok := m[key]; ok && r.URL.Path != path {
			w.Header().Set("Location", path)
			w.WriteHeader(http.StatusMovedPermanently)
			return
		}

		next.ServeHTTP(w, r)
	})
}
