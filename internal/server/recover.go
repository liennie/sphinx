package server

import (
	"net/http"
	"sphinx/internal/ctxlog"
)

type rec struct {
	next http.Handler
	err  http.Handler
}

func newRecover(next, err http.Handler) *rec {
	return &rec{
		next: next,
		err:  err,
	}
}

func (rec *rec) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log := ctxlog.Get(r.Context())
			log.Error("recovered panic", "error", err)

			clear(w.Header())
			rec.err.ServeHTTP(w, r)
		}
	}()

	rec.next.ServeHTTP(w, r)
}
