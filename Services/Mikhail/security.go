package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"os"

	"golang.org/x/oauth2"
)

var oauth2Config *oauth2.Config

func init() {
	// Environment variables will be loaded by Docker Compose or external system
	oauth2Config = &oauth2.Config{
		ClientID:     os.Getenv("YANDEX_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("YANDEX_OAUTH_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("OAUTH_REDIRECTION_URL"),
		Scopes:       []string{"login:email", "login:info"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://oauth.yandex.com/authorize",
			TokenURL: "https://oauth.yandex.com/token",
		},
	}
}

// ExchangeCode exchanges an OAuth2 code for a token
func ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return oauth2Config.Exchange(ctx, code)
}

func generate_auth_token(PhoneNumber string, PasswordHash string) string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatalf("failed to generate auth token: %v", err)
	}
	return base64.URLEncoding.EncodeToString(b)
}

func generate_refresh_token() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatalf("failed to generate refresh token: %v", err)
	}
	return base64.URLEncoding.EncodeToString(b)
}

// GetOAuth2LoginURL returns the URL for OAuth2 login
func GetOAuth2LoginURL(state string) string {
	return oauth2Config.AuthCodeURL(state)
}
