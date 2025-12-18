package smartcitizen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/timgluz/smcprober/httpclient"
	"github.com/timgluz/smcprober/metric"
)

type OauthSession struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type Provider interface {
	Authenticate(ctx context.Context, credential UserCredential) error
	HasSession() bool

	// Ping checks if the endpoint is reachable
	Ping(ctx context.Context) error
	GetMe(ctx context.Context) (User, error)
	GetDevice(ctx context.Context, deviceID int) (*DeviceDetail, error)
}

type HTTPProvider struct {
	config   Config
	session  *OauthSession
	registry metric.Registry

	client *http.Client
	logger *slog.Logger
}

func NewHTTPProvider(config Config, client *http.Client, registry metric.Registry, logger *slog.Logger) *HTTPProvider {
	// Create histogram for request duration
	histogram := registry.GetOrCreateHistogramVec(
		"api_request_duration_seconds",
		"Duration of HTTP requests to SmartCitizen API",
		[]float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0},
		[]string{"endpoint", "status", "method"},
	)

	// Wrap the client's transport with instrumentation
	if transport, ok := client.Transport.(*http.Transport); ok {
		client.Transport = httpclient.NewInstrumentedTransport(transport, histogram)
	}

	return &HTTPProvider{
		config:   config,
		client:   client,
		registry: registry,
		logger:   logger,
	}
}

func (p *HTTPProvider) Ping(ctx context.Context) error {
	p.logger.Info("Pinging the SmartCitizen API endpoint")

	pingEndpoint, err := url.JoinPath(p.config.Endpoint, p.config.APIVersion)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pingEndpoint, nil)
	if err != nil {
		return err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	// Drain the response body to allow connection reuse
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ping failed with status code: %d", resp.StatusCode)
	}

	p.logger.Info("Ping successful")
	return nil
}

func (p *HTTPProvider) Authenticate(ctx context.Context, credential UserCredential) error {
	if p.client == nil {
		return fmt.Errorf("http client is not initialized")
	}

	if p.logger == nil {
		return fmt.Errorf("logger is not initialized")
	}

	if credential.Token != "" {
		p.session = &OauthSession{
			AccessToken: credential.Token,
		}
		p.logger.Info("Using provided token for authentication")
		// Validate the token by calling GetMe
		if _, err := p.GetMe(ctx); err != nil {
			p.session = nil
			return fmt.Errorf("provided token is invalid: %w", err)
		}
		return nil
	}

	p.logger.Info("No token provided, proceeding with username/password authentication")
	session, err := p.fetchOauthSession(ctx, credential)
	if err != nil {
		return err
	}

	p.session = session
	p.logger.Info("User authenticated successfully")
	return nil
}

func (p *HTTPProvider) fetchOauthSession(ctx context.Context, credential UserCredential) (*OauthSession, error) {
	p.logger.Info("Authenticating user", "username", credential.Username)
	authData := url.Values{}
	authData.Set("username", credential.Username)
	authData.Set("password", credential.Password)

	authEndpoint, err := url.JoinPath(p.config.Endpoint, p.config.APIVersion, "/sessions")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authEndpoint, strings.NewReader(authData.Encode()))
	if err != nil {
		return nil, err
	}

	// important: set content type to application/x-www-form-urlencoded
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authentication failed with status code: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var session OauthSession
	if err := json.Unmarshal(content, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (p *HTTPProvider) HasSession() bool {
	return p.session != nil
}

func (p *HTTPProvider) GetMe(ctx context.Context) (User, error) {
	if !p.HasSession() {
		return User{}, fmt.Errorf("no active session, please authenticate first")
	}

	meEndpoint, err := url.JoinPath(p.config.Endpoint, p.config.APIVersion, "/me")
	if err != nil {
		return User{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, meEndpoint, nil)
	if err != nil {
		return User{}, err
	}

	req.Header.Set("Authorization", "Bearer "+p.session.AccessToken)

	resp, err := p.client.Do(req)
	if err != nil {
		return User{}, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return User{}, fmt.Errorf("failed to get user info with status code: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return User{}, err
	}

	var user User
	if err := json.Unmarshal(content, &user); err != nil {
		return User{}, err
	}

	return user, nil
}

func (p *HTTPProvider) GetDevice(ctx context.Context, deviceID int) (*DeviceDetail, error) {
	if !p.HasSession() {
		return nil, fmt.Errorf("no active session, please authenticate first")
	}

	deviceEndpoint, err := url.JoinPath(p.config.Endpoint,
		p.config.APIVersion,
		"/devices",
		strconv.Itoa(deviceID),
	)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, deviceEndpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+p.session.AccessToken)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get device info with status code: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var device DeviceDetail
	if err := json.Unmarshal(content, &device); err != nil {
		return nil, err
	}

	return &device, nil
}
