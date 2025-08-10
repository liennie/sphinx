package server

import (
	"net/http"
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
