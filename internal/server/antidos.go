package server

import (
	"hash/fnv"
	"io"
	"net"
	"net/http"
	"time"
)

type antidosBucket struct {
	ticker  *time.Ticker
	tickets chan struct{}
}

type antidos struct {
	buckets         []antidosBucket
	tooManyRequests http.Handler
}

func newAntidos(buckets int, period time.Duration, maxConcurrent int, tooManyRequests http.Handler) *antidos {
	b := make([]antidosBucket, buckets)
	for i := 0; i < buckets; i++ {
		b[i] = antidosBucket{
			ticker:  time.NewTicker(period),
			tickets: make(chan struct{}, maxConcurrent),
		}
	}

	return &antidos{
		buckets:         b,
		tooManyRequests: tooManyRequests,
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

		select {
		case a.buckets[bucket].tickets <- struct{}{}:
			<-a.buckets[bucket].ticker.C
			next.ServeHTTP(w, r)
			<-a.buckets[bucket].tickets

		default:
			a.tooManyRequests.ServeHTTP(w, r)
		}
	})
}
