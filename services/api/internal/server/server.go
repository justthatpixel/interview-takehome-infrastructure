package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/trybyteful/interview-takehome/services/api/internal/apiconfig"
	"github.com/trybyteful/interview-takehome/services/api/internal/common"
	"github.com/trybyteful/interview-takehome/services/api/internal/modules/health"
	"github.com/trybyteful/interview-takehome/services/api/internal/modules/postgres"
	proxyservice "github.com/trybyteful/interview-takehome/services/api/internal/services/proxy/service"
	proxytransport "github.com/trybyteful/interview-takehome/services/api/internal/services/proxy/transport"
	"go.uber.org/zap"
)

const dbPingInterval = time.Second

// APIServer is the process face: bind listeners, serve the mux, shut down.
type APIServer struct {
	logger     *zap.Logger
	httpServer *http.Server
	health     *health.Server
	lifecycle  *health.Lifecycle
	db         *pgxpool.Pool
}

func NewAPI(logger *zap.Logger, cfg *apiconfig.Config) (*APIServer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}

	db, err := postgres.NewPool(postgres.ConnConfig{
		Host:     cfg.Postgres.Host,
		Port:     cfg.Postgres.Port,
		User:     cfg.Postgres.User,
		Password: cfg.Postgres.Password,
		Database: cfg.Postgres.Database,
		SSLMode:  cfg.Postgres.SSLMode,
	})
	if err != nil {
		return nil, err
	}

	picker := proxyservice.New(proxyservice.Config{DB: db})

	lifecycle := &health.Lifecycle{}
	healthServer := health.NewServer(fmt.Sprintf("%s:%d", cfg.API.Addr, cfg.API.HealthCheckPort), logger)
	proxyAPI := proxytransport.NewHandler(picker, logger)

	mux := http.NewServeMux()
	mount(mux, proxyAPI)
	wireInfra(healthServer, lifecycle)

	return &APIServer{
		logger:    logger,
		health:    healthServer,
		lifecycle: lifecycle,
		db:        db,
		httpServer: &http.Server{
			Handler: mux,
			Addr:    fmt.Sprintf("%s:%d", cfg.API.Addr, cfg.API.Port),
		},
	}, nil
}

// Run serves until Shutdown. The startup probe passes only after Postgres
// answers a ping, and Run gives up waiting when ctx ends.
func (s *APIServer) Run(ctx context.Context) {
	s.logger.Info("API is starting")

	s.health.Start()

	if err := s.waitForDB(ctx); err != nil {
		s.logger.Info("Stopped waiting for Postgres", zap.Error(err))
		return
	}

	l, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		s.logger.Fatal("Could not start listener", zap.Error(err))
		return
	}
	s.logger.Info("Listening on TCP", zap.String("address", s.httpServer.Addr))
	s.lifecycle.MarkStarted()

	if err := s.httpServer.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.Fatal("Could not setup server", zap.Error(err))
	}
	s.logger.Info("Server stopped accepting requests")
}

// waitForDB returns nil after the first successful ping, and the context
// error when ctx ends first.
func (s *APIServer) waitForDB(ctx context.Context) error {
	ticker := time.NewTicker(dbPingInterval)
	defer ticker.Stop()

	for {
		err := s.db.Ping(ctx)
		if err == nil {
			s.logger.Info("Postgres answered a ping")
			return nil
		}
		s.logger.Warn("Postgres did not answer a ping", zap.Error(err))

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Shutdown fails readiness, drains the requests in flight, closes the pool,
// then stops the health server last so the probes answer for the whole drain.
func (s *APIServer) Shutdown(ctx context.Context) {
	s.lifecycle.MarkShuttingDown()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP server shutdown error", zap.Error(err))
	}
	s.db.Close()
	s.health.Shutdown(ctx)
}

// mount registers every route a service's transport declares. The server
// never names a handler method. Metered routes share one request limit.
func mount(mux *http.ServeMux, providers ...common.Endpoints) {
	slots := newCapacity(maxInFlight)
	for _, p := range providers {
		for _, rt := range p.Routes() {
			if rt.Metered {
				mux.Handle(rt.Path, instrument(rt.RouteLabel(), slots.limit(rt.Handler)))
				continue
			}
			mux.HandleFunc(rt.Path, rt.Handler)
		}
	}
}

// wireInfra registers the endpoints that are not a service: the probes and
// the metrics scrape, all on the health server.
func wireInfra(healthServer *health.Server, lifecycle *health.Lifecycle) {
	m := healthServer.Mux()
	m.HandleFunc("GET /ping", lifecycle.Live)
	m.HandleFunc("GET /db", lifecycle.Started)
	m.HandleFunc("GET /status", lifecycle.Ready)
	m.Handle("GET /metrics", promhttp.Handler())
}
