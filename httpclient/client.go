package httpclient

import (
	"net"
	"net/http"
	"time"
)

// NewHTTPClient creates an HTTP client with sensible timeout defaults
func NewDefaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second, // Overall request timeout
		Transport: &http.Transport{
			// Connection settings
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second, // Time to establish connection
				KeepAlive: 30 * time.Second, // Keep-alive probe interval
			}).DialContext,

			// TLS handshake timeout
			TLSHandshakeTimeout: 10 * time.Second,

			// Timeouts for different phases
			ResponseHeaderTimeout: 10 * time.Second, // Time to receive response headers
			ExpectContinueTimeout: 1 * time.Second,  // Time to wait for 100-continue response

			// Connection pooling
			MaxIdleConns:        100,              // Max idle connections across all hosts
			MaxIdleConnsPerHost: 10,               // Max idle connections per host
			MaxConnsPerHost:     100,              // Max total connections per host
			IdleConnTimeout:     90 * time.Second, // Time before idle connection is closed

			// Avoid connection reuse issues
			DisableKeepAlives: false,
		},
	}
}

// ClientOption is a function that configures an HTTP client
type ClientOption func(*http.Client)

// WithTimeout sets the overall request timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *http.Client) {
		c.Timeout = timeout
	}
}

// WithDialTimeout sets the connection dial timeout
func WithDialTimeout(timeout time.Duration) ClientOption {
	return func(c *http.Client) {
		if transport, ok := c.Transport.(*http.Transport); ok {
			transport.DialContext = (&net.Dialer{
				Timeout:   timeout,
				KeepAlive: 30 * time.Second,
			}).DialContext
		}
	}
}

// WithTLSTimeout sets the TLS handshake timeout
func WithTLSTimeout(timeout time.Duration) ClientOption {
	return func(c *http.Client) {
		if transport, ok := c.Transport.(*http.Transport); ok {
			transport.TLSHandshakeTimeout = timeout
		}
	}
}

// WithResponseHeaderTimeout sets the response header timeout
func WithResponseHeaderTimeout(timeout time.Duration) ClientOption {
	return func(c *http.Client) {
		if transport, ok := c.Transport.(*http.Transport); ok {
			transport.ResponseHeaderTimeout = timeout
		}
	}
}

// WithMaxIdleConns sets the maximum idle connections
func WithMaxIdleConns(n int) ClientOption {
	return func(c *http.Client) {
		if transport, ok := c.Transport.(*http.Transport); ok {
			transport.MaxIdleConns = n
		}
	}
}

// WithMaxIdleConnsPerHost sets the maximum idle connections per host
func WithMaxIdleConnsPerHost(n int) ClientOption {
	return func(c *http.Client) {
		if transport, ok := c.Transport.(*http.Transport); ok {
			transport.MaxIdleConnsPerHost = n
		}
	}
}

// WithKeepAlive sets the keep-alive probe interval
func WithKeepAlive(interval time.Duration) ClientOption {
	return func(c *http.Client) {
		if transport, ok := c.Transport.(*http.Transport); ok {
			transport.DialContext = (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: interval,
			}).DialContext
		}
	}
}

// NewHTTPClientWithOptions creates an HTTP client with options pattern
func NewHTTPClientWithOptions(opts ...ClientOption) *http.Client {
	// Start with default client
	client := NewDefaultHTTPClient()

	// Apply all options
	for _, opt := range opts {
		opt(client)
	}

	return client
}
