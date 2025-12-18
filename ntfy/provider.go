package ntfy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type Notifier interface {
	Send(ctx context.Context, msg Notification) error
}

type HTTPNotifier struct {
	endpoint string

	client      *http.Client
	logger      *slog.Logger
	credentials TokenCredentialProvider
}

func NewHTTPNotifier(endpoint string, client *http.Client, logger *slog.Logger) *HTTPNotifier {
	return &HTTPNotifier{
		endpoint: endpoint,
		client:   client,
		logger:   logger,
	}
}

func (n *HTTPNotifier) SetCredentialProvider(provider TokenCredentialProvider) error {
	n.credentials = provider
	return nil
}

func (n *HTTPNotifier) Send(ctx context.Context, msg Notification) error {
	// Implementation of sending notification
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication if credentials are provided
	if n.credentials != nil {
		token, err := n.credentials.Retrieve(ctx)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	n.logger.Info("Sending notification", "topic", msg.Topic)
	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			n.logger.Warn("Failed to close response body", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send notification, status code: %d", resp.StatusCode)
	}

	n.logger.Info("Notification sent successfully", "topic", msg.Topic)

	return nil
}
