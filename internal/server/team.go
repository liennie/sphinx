package server

import (
	"html/template"
	"net/http"
	"sphinx/internal/ctxlog"
	"sphinx/internal/db"
	"strings"
	"time"
)

type teams struct {
	tmpl *template.Template
	ct   string
}

func newTeams(content []byte, ct string) *teams {
	return &teams{
		tmpl: template.Must(template.New("teams").Parse(string(content))),
		ct:   ct,
	}
}

func (t *teams) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cookie, _ := r.Cookie("X-Team-Name"); cookie != nil {
			t.next(w, r, cookie.Value, next)
			return
		}

		if r.Method == http.MethodPost {
			team := r.FormValue("team_name")
			if len(team) == 0 || len(team) > 30 {
				t.render(w, r, true)
				return
			}
			if strings.ContainsFunc(team, func(c rune) bool {
				return c != ' ' && c != '_' && c != '-' && (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9')
			}) {
				t.render(w, r, true)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:    "X-Team-Name",
				Value:   team,
				Path:    "/",
				Expires: time.Now().Add(365 * 24 * time.Hour),
			})

			t.next(w, r, team, next)
			return
		}

		if r.Method == http.MethodGet {
			t.render(w, r, false)
			return
		}
	})
}

func (t *teams) render(w http.ResponseWriter, r *http.Request, terr bool) {
	log := ctxlog.Get(r.Context())

	teams, err := db.Teams()
	if err != nil {
		log.Error("failed to load teams from db", "error", err)
	}

	type data struct {
		Teams []string
		Error bool
	}

	w.Header().Set("Content-Type", t.ct)
	w.WriteHeader(http.StatusOK)
	if err := t.tmpl.Execute(w, data{Teams: teams, Error: terr}); err != nil {
		log.Error("failed to write response", "error", err)
	}
}

func (t *teams) next(w http.ResponseWriter, r *http.Request, team string, next http.Handler) {
	puzzle := r.URL.Path
	puzzle = strings.TrimPrefix(puzzle, "/")
	puzzle = strings.TrimSuffix(puzzle, "/")
	db.AddTeamProgress(team, puzzle, time.Now())

	r = r.WithContext(ctxlog.With(r.Context(), "team", team))
	l := ctxlog.Get(r.Context())
	l.Info("team")

	next.ServeHTTP(w, r)
}
