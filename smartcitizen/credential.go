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

func NewUserCredentialEnvProvider(usernameEnv, passworEnv string) *UserCredentialEnvProvider {
	return &UserCredentialEnvProvider{
		usernameEnvVar: usernameEnv,
		passwordEnvVar: passworEnv,
		tokenEnvVar:    "SMARTCITIZEN_TOKEN",
	}
}

func (p *UserCredentialEnvProvider) Retrieve(ctx context.Context) (UserCredential, error) {
	username := os.Getenv(p.usernameEnvVar)
	if username == "" {
		return UserCredential{}, fmt.Errorf("environment variable SMARTCITIZEN_USERNAME must be set")
	}

	password := os.Getenv(p.passwordEnvVar)
	token := os.Getenv(p.tokenEnvVar)

	if password == "" && token == "" {
		return UserCredential{}, fmt.Errorf("either environment variable SMARTCITIZEN_PASSWORD or SMARTCITIZEN_TOKEN must be set")
	}

	return UserCredential{
		Username: username,
		Password: password,
		Token:    token,
	}, nil
}
