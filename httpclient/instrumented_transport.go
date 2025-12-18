package httpclient

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// InstrumentedTransport wraps http.RoundTripper to measure request duration
type InstrumentedTransport struct {
	base      http.RoundTripper
	histogram *prometheus.HistogramVec
}

// NewInstrumentedTransport creates a transport that records metrics
func NewInstrumentedTransport(base http.RoundTripper, histogram *prometheus.HistogramVec) *InstrumentedTransport {
	return &InstrumentedTransport{
		base:      base,
		histogram: histogram,
	}
}

func (t *InstrumentedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	// Use full URL path as endpoint (preserves API version info)
	endpoint := req.URL.Path
	method := req.Method

	// Execute request
	resp, err := t.base.RoundTrip(req)
	duration := time.Since(start).Seconds()

	// Determine status
	status := "error"
	if err == nil {
		status = statusCategory(resp.StatusCode)
	}

	// Record metric
	t.histogram.WithLabelValues(endpoint, status, method).Observe(duration)

	return resp, err
}

// statusCategory converts HTTP status code to human-friendly category
func statusCategory(code int) string {
	if code >= 200 && code < 300 {
		return "success"
	} else if code >= 400 && code < 500 {
		return "client_error"
	} else if code >= 500 {
		return "server_error"
	}
	return "unknown"
}
