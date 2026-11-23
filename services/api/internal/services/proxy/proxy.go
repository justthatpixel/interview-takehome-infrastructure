package proxy

import (
	"context"
	"errors"
	"net/netip"
)

// ErrNoProxy means the pool holds no proxy to hand out.
var ErrNoProxy = errors.New("no proxy available")

// Picker is the proxy service contract. The transport and the server wiring
// depend on it.
type Picker interface {
	Random(ctx context.Context) (Proxy, error)
}

// Proxy is one residential proxy that a customer request can go out through.
type Proxy struct {
	Addr     netip.AddrPort
	Country  string
	City     string
	Protocol string
}
