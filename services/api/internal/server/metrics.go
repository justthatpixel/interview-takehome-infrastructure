package server

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Request metrics that wiring owns. Names and labels are the dashboard and
// autoscaling contract — they change deliberately, not incidentally.

var requestsInFlight = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "http_requests_in_flight",
	Help: "Number of HTTP requests that the server is answering now",
})

var totalRequests = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "http_requests_total",
	Help: "Total number of HTTP requests by route and status code",
}, []string{"route", "code"})

var requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "http_request_duration_seconds",
	Help:    "Duration of HTTP responses",
	Buckets: []float64{0.1, 0.25, 0.5, 1, 1.5, 2, 2.5, 3, 5},
}, []string{"route"})

// instrument wraps h with the request metrics, labelled with route.
func instrument(route string, h http.Handler) http.Handler {
	labels := prometheus.Labels{"route": route}
	return promhttp.InstrumentHandlerInFlight(requestsInFlight,
		promhttp.InstrumentHandlerDuration(requestDuration.MustCurryWith(labels),
			promhttp.InstrumentHandlerCounter(totalRequests.MustCurryWith(labels), h),
		),
	)
}
