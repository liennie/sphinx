package server

import (
	"hash/fnv"
	"io"
	"net"
	"net/http"
	"time"
)

type antidos struct {
	buckets []*time.Ticker
}

func newAntidos(buckets int, period time.Duration) *antidos {
	b := make([]*time.Ticker, buckets)
	for i := 0; i < buckets; i++ {
		b[i] = time.NewTicker(period)
	}

	return &antidos{
		buckets: b,
	}
}

func (a *antidos) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bucket int
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			h := fnv.New64()
			io.WriteString(h, host)
			bucket = int(h.Sum64() % uint64(len(a.buckets)))
		}

		<-a.buckets[bucket].C

		next.ServeHTTP(w, r)
	})
}
