package smartcitizen

import (
	"context"
	"fmt"
	"os"
)

type UserCredential struct {
	Username string
	Password string
	Token    string
}

type UserCredentialProvider interface {
	Retrieve(ctx context.Context) (UserCredential, error)
}

type UserCredentialEnvProvider struct {
	usernameEnvVar string
	passwordEnvVar string
	tokenEnvVar    string
}

func NewUserCredentialEnvProvider(usernameEnv, passwordEnv, tokenEnv string) *UserCredentialEnvProvider {
	return &UserCredentialEnvProvider{
		usernameEnvVar: usernameEnv,
		passwordEnvVar: passwordEnv,
		tokenEnvVar:    tokenEnv,
	}
}

func (p *UserCredentialEnvProvider) Retrieve(ctx context.Context) (UserCredential, error) {
	username := os.Getenv(p.usernameEnvVar)
	if username == "" {
		return UserCredential{}, fmt.Errorf("environment variable %s must be set", p.usernameEnvVar)
	}

	password := os.Getenv(p.passwordEnvVar)
	token := os.Getenv(p.tokenEnvVar)

	if password == "" && token == "" {
		return UserCredential{}, fmt.Errorf("either environment variable %s or %s must be set", p.passwordEnvVar, p.tokenEnvVar)
	}

	return UserCredential{
		Username: username,
		Password: password,
		Token:    token,
	}, nil
}
