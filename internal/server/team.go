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

func validTeamName(team string) bool {
	if len(team) == 0 || len(team) > 30 {
		return false
	}

	if strings.ContainsFunc(team, func(c rune) bool {
		return c != ' ' && c != '_' && c != '-' && (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9')
	}) {
		return false
	}

	return true
}

func (t *teams) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cookie, _ := r.Cookie("Team-Bypass"); cookie != nil {
			next.ServeHTTP(w, r)
			return
		}

		if cookie, _ := r.Cookie("Team-Name"); cookie != nil && cookie.Value != "" {
			team := cookie.Value
			if newName, err := db.TeamRename(team); err != nil {
				log := ctxlog.Get(r.Context())
				log.Error("failed to get team rename", "error", err)
			} else if team != newName {
				team = newName
				http.SetCookie(w, &http.Cookie{
					Name:    "Team-Name",
					Value:   team,
					Path:    "/",
					Expires: time.Now().Add(365 * 24 * time.Hour),
				})
			}

			if validTeamName(team) {
				t.next(w, r, team, next)
				return
			}
		}

		if r.Method == http.MethodPost {
			team := r.FormValue("team_name")
			team = strings.TrimSpace(team)

			if newName, err := db.TeamRename(team); err != nil {
				log := ctxlog.Get(r.Context())
				log.Error("failed to get team rename", "error", err)
			} else {
				team = newName
			}

			if !validTeamName(team) {
				t.render(w, r, true)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:    "Team-Name",
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

	if len(teams) > 30 {
		log.Warn("too many teams, hiding team list", "len", len(teams))
		teams = nil
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
