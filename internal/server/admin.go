package server

import (
	"encoding/json"
	"maps"
	"net/http"
	"sphinx/internal/ctxlog"
	"sphinx/internal/db"
)

type admin struct {
	key             string
	notFoundHandler http.Handler
}

func newAdmin(key string, notFoundHandler http.Handler) *admin {
	return &admin{
		key:             key,
		notFoundHandler: notFoundHandler,
	}
}

func (a *admin) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cookie, _ := r.Cookie("X-Admin-Key"); cookie != nil && cookie.Value == a.key {
			next.ServeHTTP(w, r)
			return
		}

		a.notFoundHandler.ServeHTTP(w, r)
	})
}

func reloadHandler(rh *reloadingHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e := json.NewEncoder(w)
		w.Header().Set("Content-Type", "application/json")

		if err := rh.reload(); err != nil {
			log := ctxlog.Get(r.Context())
			log.Error("failed to reload handler", "error", err)

			w.WriteHeader(http.StatusInternalServerError)
			e.Encode(map[string]string{
				"error": err.Error(),
			})

			return
		}

		w.WriteHeader(http.StatusOK)
		e.Encode(map[string]string{})
	})
}

func progressHandler(order []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e := json.NewEncoder(w)
		w.Header().Set("Content-Type", "application/json")

		teams := maps.Collect(db.All())

		w.WriteHeader(http.StatusOK)
		e.Encode(map[string]any{
			"order": order,
			"teams": teams,
		})
	})
}

func hideTeamHandler(w http.ResponseWriter, r *http.Request) {
	e := json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Team string `json:"team"`
		Hide bool   `json:"hide"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		e.Encode(map[string]string{
			"error": "invalid JSON",
		})
		return
	}

	if req.Team == "" {
		w.WriteHeader(http.StatusBadRequest)
		e.Encode(map[string]string{
			"error": "missing team",
		})
		return
	}

	if err := db.SetTeamHidden(req.Team, req.Hide); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		e.Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	e.Encode(map[string]string{})
}
