package server

import (
	"hash/fnv"
	"io"
	"net"
	"net/http"
	"time"
)

type antiCheatBucket struct {
	ticker  *time.Ticker
	tickets chan struct{}
}

type antiCheat struct {
	buckets         []antiCheatBucket
	tooManyRequests http.Handler
}

func newAntiCheat(buckets int, period time.Duration, maxConcurrent int, tooManyRequests http.Handler) *antiCheat {
	b := make([]antiCheatBucket, buckets)
	for i := 0; i < buckets; i++ {
		b[i] = antiCheatBucket{
			ticker:  time.NewTicker(period),
			tickets: make(chan struct{}, maxConcurrent),
		}
	}

	return &antiCheat{
		buckets:         b,
		tooManyRequests: tooManyRequests,
	}
}

func (a *antiCheat) middleware(next http.Handler) http.Handler {
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
