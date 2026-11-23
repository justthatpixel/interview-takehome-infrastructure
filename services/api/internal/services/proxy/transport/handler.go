package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/trybyteful/interview-takehome/services/api/internal/common"
	"github.com/trybyteful/interview-takehome/services/api/internal/services/proxy"
	"go.uber.org/zap"
)

// Handler is the proxy service transport: call the service, encode the result.
type Handler struct {
	service proxy.Picker
	logger  *zap.Logger
}

func NewHandler(service proxy.Picker, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

type proxyResponse struct {
	IP       string `json:"ip"`
	Port     uint16 `json:"port"`
	Country  string `json:"country"`
	City     string `json:"city"`
	Protocol string `json:"protocol"`
}

func (h *Handler) Random(w http.ResponseWriter, r *http.Request) {
	p, err := h.service.Random(r.Context())
	switch {
	case errors.Is(err, context.Canceled):
		// The client left, so nobody reads the response.
		return
	case errors.Is(err, proxy.ErrNoProxy):
		http.Error(w, "No proxy available", http.StatusNotFound)
		return
	case err != nil:
		h.logger.Error("Random proxy failed", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(proxyResponse{
		IP:       p.Addr.Addr().String(),
		Port:     p.Addr.Port(),
		Country:  p.Country,
		City:     p.City,
		Protocol: p.Protocol,
	})
}

// Routes are the proxy service's HTTP endpoints.
func (h *Handler) Routes() []common.Route {
	return []common.Route{
		{Path: "GET /proxy", Label: "/proxy", Handler: h.Random, Metered: true},
	}
}
