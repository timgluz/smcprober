package httpclient

import (
	"net/http"
	"strconv"
	"strings"
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

	// Extract endpoint name from URL path
	endpoint := extractEndpoint(req.URL.Path)
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

// extractEndpoint converts URL path to logical endpoint name
// Examples:
//   /v0 -> "ping"
//   /v0/me -> "me"
//   /v0/devices/123 -> "devices"
//   /v0/sessions -> "sessions"
func extractEndpoint(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Filter out API version prefix (v0, v1, etc.)
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.HasPrefix(part, "v") && len(part) <= 3 {
			continue // Skip version prefix
		}
		filtered = append(filtered, part)
	}

	if len(filtered) == 0 {
		return "ping"
	}

	// First segment after version is the endpoint
	endpoint := filtered[0]

	// Check if it's a resource ID (numeric) - return collection name for /devices/123
	if len(filtered) > 1 && isNumeric(filtered[1]) {
		return endpoint
	}

	return endpoint
}

// statusCategory converts HTTP status code to category
func statusCategory(code int) string {
	if code >= 200 && code < 300 {
		return "2xx"
	} else if code >= 400 && code < 500 {
		return "4xx"
	} else if code >= 500 {
		return "5xx"
	}
	return "other"
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
