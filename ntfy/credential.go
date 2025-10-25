package ntfy

import (
	"context"
	"fmt"
	"os"
)

type TokenCredentialProvider interface {
	Retrieve(ctx context.Context) (string, error)
}

type TokenCredentialEnvProvider struct {
	envVar string
}

func NewTokenCredentialEnvProvider(envVar string) *TokenCredentialEnvProvider {
	return &TokenCredentialEnvProvider{
		envVar: envVar,
	}
}

func (p *TokenCredentialEnvProvider) Retrieve(ctx context.Context) (string, error) {
	token := os.Getenv(p.envVar)
	if token == "" {
		return "", fmt.Errorf("environment variable %s must be set", p.envVar)
	}

	return token, nil
}
