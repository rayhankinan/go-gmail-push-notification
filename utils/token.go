package utils

import (
	"context"
	"fmt"
	"log"

	"github.com/int128/oauth2cli"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
)

func GetToken(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	ready := make(chan string, 1)
	defer close(ready)

	pkceVerifier := oauth2.GenerateVerifier()
	cli := oauth2cli.Config{
		OAuth2Config: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  config.Endpoint.AuthURL,
				TokenURL: config.Endpoint.TokenURL,
			},
			Scopes: config.Scopes,
		},
		AuthCodeOptions:      []oauth2.AuthCodeOption{oauth2.S256ChallengeOption(pkceVerifier)},
		TokenRequestOptions:  []oauth2.AuthCodeOption{oauth2.VerifierOption(pkceVerifier)},
		LocalServerReadyChan: ready,
		Logf:                 log.Printf,
	}

	go func() {
		select {
		case url := <-ready:
			if err := browser.OpenURL(url); err != nil {
				log.Printf("Failed to open browser: %v", err)
			}
		case <-ctx.Done():
			log.Printf("Context cancelled: %v", ctx.Err())
		}
	}()

	token, err := oauth2cli.GetToken(ctx, cli)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	return token, nil
}
