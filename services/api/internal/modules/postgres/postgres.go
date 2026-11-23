package postgres

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnConfig is what a pool needs to reach one database.
type ConnConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// NewPool returns a pool for cfg. It does not connect: the first query or
// ping opens the first connection.
// It returns an error when cfg does not form a valid connection config.
func NewPool(cfg ConnConfig) (*pgxpool.Pool, error) {
	connURL := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Path:     cfg.Database,
		RawQuery: url.Values{"sslmode": {cfg.SSLMode}}.Encode(),
	}

	poolCfg, err := pgxpool.ParseConfig(connURL.String())
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	return pgxpool.NewWithConfig(context.Background(), poolCfg)
}
