package smartcitizen

import (
	"context"
	"fmt"
	"os"
)

type UserCredential struct {
	Username string
	Password string
}

type UserCredentialProvider interface {
	Retrieve(ctx context.Context) (UserCredential, error)
}

type UserCredentialEnvProvider struct {
	usernameEnvVar string
	passwordEnvVar string
}

func NewUserCredentialEnvProvider(usernameEnv, passworEnv string) *UserCredentialEnvProvider {
	return &UserCredentialEnvProvider{
		usernameEnvVar: usernameEnv,
		passwordEnvVar: passworEnv,
	}
}

func (p *UserCredentialEnvProvider) Retrieve(ctx context.Context) (UserCredential, error) {
	username := os.Getenv(p.usernameEnvVar)
	if username == "" {
		return UserCredential{}, fmt.Errorf("environment variable SMARTCITIZEN_USERNAME must be set")
	}

	password := os.Getenv(p.passwordEnvVar)
	if password == "" {
		return UserCredential{}, fmt.Errorf("environment variable SMARTCITIZEN_PASSWORD must be set")
	}

	return UserCredential{
		Username: username,
		Password: password,
	}, nil
}
