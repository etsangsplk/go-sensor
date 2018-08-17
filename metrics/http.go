package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/splunk/ssc-observation/tracing"
)

var (
	httpRequestsActive             *prometheus.GaugeVec
	httpRequestsDurationsHistogram *prometheus.HistogramVec
)

// RegisterHTTPMetrics registers the http metrics for scraping at
// the service's prometheus metrics endpoint. The metrics registered
// are http_requests_active and http_requests_durations_histogram_seconds.
// If provided namespace is prefixed to the metric name for each metric.
func RegisterHTTPMetrics(namespace string) {
	httpRequestsActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "http_requests_active",
			Help:      "The count of current active http requests, partitioned by method and operation",
		},
		[]string{"method", "operation"})
	httpRequestsDurationsHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_requests_durations_histogram_seconds",
			Help:      "Http request latency distributions, partitioned by method, operation and statusCode",
			Buckets:   prometheus.DefBuckets},
		[]string{"method", "operation", "statusCode"})

	prometheus.MustRegister(
		httpRequestsActive,
		httpRequestsDurationsHistogram)
}

// httpAccessHandler provides http middleware to observe
// http metrics
type httpAccessHandler struct {
	next http.Handler
}

// NewHTTPAccessHandler constructs a new middleware instance for observing
// http metrics with the http_requests_durations_histogram_seconds metric.
// The handler uses tracing.OperationIDFrom() to extract the operation ID
// from the request context, if any.
func NewHTTPAccessHandler(next http.Handler) http.Handler {
	return &httpAccessHandler{next: next}
}

func (h *httpAccessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rw := tracing.NewHTTPResponseWriter(w)
	operation := tracing.OperationIDFrom(r.Context())
	httpRequestsActive.WithLabelValues(r.Method, operation).Inc()

	start := time.Now()
	h.next.ServeHTTP(rw, r)
	duration := time.Since(start)

	statusCodeString := strconv.FormatInt(int64(rw.StatusCode()), 10)
	httpRequestsDurationsHistogram.WithLabelValues(r.Method, operation, statusCodeString).
		Observe(duration.Seconds())
	httpRequestsActive.WithLabelValues(r.Method, operation).Dec()
}

// prometheusHandler provides http middleware for serving the metrics endpoint
type prometheusHandler struct {
	prom http.Handler
	next http.Handler
}

// NewPrometheusHandler constructs a new middleware instance for serving
// the Prometheus metrics endpoint at "/service/metrics".
func NewPrometheusHandler(next http.Handler) http.Handler {
	return &prometheusHandler{
		prom: promhttp.Handler(),
		next: next,
	}
}

// ServeHTTP implements the http.Handler interface.
func (p *prometheusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/service/metrics" {
		p.prom.ServeHTTP(w, r)
	} else {
		p.next.ServeHTTP(w, r)
	}
}
