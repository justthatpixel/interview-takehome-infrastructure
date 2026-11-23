package service

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/trybyteful/interview-takehome/services/api/internal/services/proxy"
	"github.com/trybyteful/interview-takehome/services/api/internal/services/proxy/store"
)

// The range of the delay before each pick.
const (
	delayMin = 200 * time.Millisecond
	delayMax = 3 * time.Second
)

type Config struct {
	DB store.Querier
}

// Picker hands out a random proxy after a random delay.
type Picker struct {
	store proxyStore
}

func New(cfg Config) *Picker {
	return &Picker{store: store.New(cfg.DB)}
}

// Random returns one proxy from the store after the delay.
// It returns the context error when ctx ends first, and the store error when the read fails.
func (p *Picker) Random(ctx context.Context) (proxy.Proxy, error) {
	// The delay stands in for the routing work a real pick does, so that load
	// shows up in the metrics.
	timer := time.NewTimer(randomDelay())
	defer timer.Stop()

	select {
	case <-timer.C:
	case <-ctx.Done():
		return proxy.Proxy{}, ctx.Err()
	}

	return p.store.RandomProxy(ctx)
}

func randomDelay() time.Duration {
	return delayMin + rand.N(delayMax-delayMin+1)
}
