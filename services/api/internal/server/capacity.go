package server

import "net/http"

// maxInFlight is the number of metered requests that one process answers at
// once.
const maxInFlight = 20

// capacity is the set of free request slots, shared by every metered route.
type capacity chan struct{}

func newCapacity(n int) capacity {
	return make(capacity, n)
}

// limit returns h, wrapped to answer 503 when no slot is free.
func (c capacity) limit(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case c <- struct{}{}:
			defer func() { <-c }()
			h.ServeHTTP(w, r)
		default:
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		}
	})
}
