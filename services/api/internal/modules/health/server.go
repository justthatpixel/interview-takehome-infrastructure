package health

import (
	"context"
	"errors"
	"net/http"

	"go.uber.org/zap"
)

// Server is the side listener for the probes and the metrics scrape. It stays
// up while the main server drains, so the probes can still answer.
type Server struct {
	mux        *http.ServeMux
	httpServer *http.Server
	logger     *zap.Logger
}

func NewServer(addr string, logger *zap.Logger) *Server {
	mux := http.NewServeMux()
	return &Server{
		mux:    mux,
		logger: logger,
		httpServer: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
}

func (s *Server) Mux() *http.ServeMux {
	return s.mux
}

func (s *Server) Start() {
	go func() {
		s.logger.Info("Health check server starting", zap.String("address", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				s.logger.Info("Health check server stopped")
				return
			}
			s.logger.Fatal("Could not setup Health Check server", zap.Error(err))
		}
	}()
}

func (s *Server) Shutdown(ctx context.Context) {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("Health check server shutdown error", zap.Error(err))
	}
}
