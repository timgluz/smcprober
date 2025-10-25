package smartcitizen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

var DefaultEndpoint = "https://api.smartcitizen.me"
var DefaultAPIVersion = "v0"

type OauthSession struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type Provider interface {
	Authenticate(ctx context.Context, credential UserCredential) (OauthSession, error)
	HasSession() bool

	GetMe(ctx context.Context) (User, error)
	GetDevice(ctx context.Context, deviceID int) (*DeviceDetail, error)
}

type HTTPProvider struct {
	session *OauthSession

	client *http.Client
	logger *slog.Logger
}

func NewHTTPProvider(client *http.Client, logger *slog.Logger) *HTTPProvider {
	return &HTTPProvider{
		client: client,
		logger: logger,
	}
}

func (p *HTTPProvider) Authenticate(ctx context.Context, credential UserCredential) error {
	if p.client == nil {
		return fmt.Errorf("http client is not initialized")
	}

	if p.logger == nil {
		return fmt.Errorf("logger is not initialized")
	}

	p.logger.Info("Authenticating user", "username", credential.Username)
	authData := url.Values{}
	authData.Set("username", credential.Username)
	authData.Set("password", credential.Password)

	authEndpoint, err := url.JoinPath(DefaultEndpoint, DefaultAPIVersion, "/sessions")
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authEndpoint, strings.NewReader(authData.Encode()))
	if err != nil {
		return err
	}

	// important: set content type to application/x-www-form-urlencoded
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status code: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var session OauthSession
	if err := json.Unmarshal(content, &session); err != nil {
		return err
	}

	p.session = &session
	return nil
}

func (p *HTTPProvider) HasSession() bool {
	return p.session != nil
}

func (p *HTTPProvider) GetMe(ctx context.Context) (User, error) {
	if !p.HasSession() {
		return User{}, fmt.Errorf("no active session, please authenticate first")
	}

	meEndpoint, err := url.JoinPath(DefaultEndpoint, DefaultAPIVersion, "/me")
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

	deviceEndpoint, err := url.JoinPath(DefaultEndpoint, DefaultAPIVersion, fmt.Sprintf("/devices/%d", deviceID))
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
