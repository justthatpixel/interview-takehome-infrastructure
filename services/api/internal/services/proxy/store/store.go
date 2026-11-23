package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/trybyteful/interview-takehome/services/api/internal/services/proxy"
)

// Querier is the slice of a connection pool the store needs.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Store struct {
	db Querier
}

func New(db Querier) *Store {
	return &Store{db: db}
}

// The table is small, so a random sort is cheap enough.
const randomProxySQL = `
SELECT residential_proxy_ip_address_v4, residential_proxy_port, country_id, city_id, residential_proxy_supported_protocol
FROM residential_proxy
ORDER BY random()
LIMIT 1`

// RandomProxy returns one proxy picked at random.
// It returns proxy.ErrNoProxy when the table is empty.
func (s *Store) RandomProxy(ctx context.Context) (proxy.Proxy, error) {
	var row proxyRow
	err := s.db.QueryRow(ctx, randomProxySQL).Scan(&row.IP, &row.Port, &row.Country, &row.City, &row.Protocol)
	if errors.Is(err, pgx.ErrNoRows) {
		return proxy.Proxy{}, proxy.ErrNoProxy
	}
	if err != nil {
		return proxy.Proxy{}, fmt.Errorf("select random proxy: %w", err)
	}
	return row.toEntity(), nil
}
