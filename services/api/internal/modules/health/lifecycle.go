package health

import (
	"net/http"
	"sync/atomic"
)

// Lifecycle is the state of one server process that the Kubernetes probes
// read. The zero value is a process that has not started.
type Lifecycle struct {
	started      atomic.Bool
	shuttingDown atomic.Bool
}

// MarkStarted records that the server accepts requests.
func (l *Lifecycle) MarkStarted() { l.started.Store(true) }

// MarkShuttingDown records that the server must get no new requests.
func (l *Lifecycle) MarkShuttingDown() { l.shuttingDown.Store(true) }

// Live answers 200 while the process can answer at all.
func (l *Lifecycle) Live(w http.ResponseWriter, r *http.Request) {
	writeStatus(w, true)
}

// Started answers 200 after MarkStarted, and 503 before it.
func (l *Lifecycle) Started(w http.ResponseWriter, r *http.Request) {
	writeStatus(w, l.started.Load())
}

// Ready answers 200 after MarkStarted and before MarkShuttingDown, and 503 otherwise.
func (l *Lifecycle) Ready(w http.ResponseWriter, r *http.Request) {
	writeStatus(w, l.started.Load() && !l.shuttingDown.Load())
}

func writeStatus(w http.ResponseWriter, ok bool) {
	if !ok {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Write([]byte("ok\n"))
}
