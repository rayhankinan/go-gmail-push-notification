package utils

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

func GetClient(ctx context.Context, secretPath, tokenPath string) (*gmail.Service, error) {
	b, err := os.ReadFile(secretPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read client secret file: %w", err)
	}

	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse client secret file to config: %w", err)
	}

	cacheToken := NewCacheFile(tokenPath, func() (*oauth2.Token, error) {
		return GetToken(ctx, config)
	})
	token, err := cacheToken.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	client := config.Client(ctx, token)
	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail client: %w", err)
	}

	return srv, nil
}
