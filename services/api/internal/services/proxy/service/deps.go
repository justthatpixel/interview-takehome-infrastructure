package service

import (
	"context"

	"github.com/trybyteful/interview-takehome/services/api/internal/services/proxy"
)

// proxyStore is the storage the proxy service reads.
type proxyStore interface {
	RandomProxy(ctx context.Context) (proxy.Proxy, error)
}
