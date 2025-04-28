package utils

import (
	"context"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
)

func GetSubscription(ctx context.Context, credentialsPath, projectID, subscriptionID string) (*pubsub.Subscription, error) {
	client, err := pubsub.NewClient(ctx, projectID, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		return nil, err
	}
	return client.Subscription(subscriptionID), nil
}
