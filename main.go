package main

import (
	"context"
	"go-gmail-notification/utils"
	"log"

	"cloud.google.com/go/pubsub"
	"github.com/spf13/cobra"
	"google.golang.org/api/gmail/v1"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	cli := &cobra.Command{}

	cli.AddCommand(
		&cobra.Command{
			Use:   "start-watch",
			Short: "Start a watch process",
			Long:  "Starts a watch process that monitors gmail inbox",
			Run: func(cmd *cobra.Command, args []string) {
				ctx := cmd.Context()

				srv, err := utils.GetClient(ctx, "./secrets/client_secret_494104920232-6smddih7ahdcgpvd6fun9srertn4l5qs.apps.googleusercontent.com.json", "./secrets/token.json")
				if err != nil {
					log.Fatalf("Unable to get Gmail client: %v", err)
				}

				user := "me"
				request := &gmail.WatchRequest{
					LabelIds:            []string{"INBOX", "UNREAD"},
					LabelFilterBehavior: "include",
					TopicName:           "projects/devlab-404500/topics/gmail-notification",
				}
				r, err := srv.Users.Watch(user, request).Do()
				if err != nil {
					log.Fatalf("Unable to watch inbox: %v", err)
				}

				log.Printf("Watch started successfully: %+v", r)
			},
		},
		&cobra.Command{
			Use:   "stop-watch",
			Short: "Stop a watch process",
			Long:  "Stops a watch process that monitors gmail inbox",
			Run: func(cmd *cobra.Command, args []string) {
				ctx := cmd.Context()

				srv, err := utils.GetClient(ctx, "./secrets/client_secret_494104920232-6smddih7ahdcgpvd6fun9srertn4l5qs.apps.googleusercontent.com.json", "./secrets/token.json")
				if err != nil {
					log.Fatalf("Unable to get Gmail client: %v", err)
				}

				user := "me"
				if err := srv.Users.Stop(user).Do(); err != nil {
					log.Fatalf("Unable to stop watch: %v", err)
				}

				log.Println("Watch stopped successfully")
			},
		},
		&cobra.Command{
			Use:   "process-message",
			Short: "Process a message",
			Long:  "Processes a message from the gmail inbox",
			Run: func(cmd *cobra.Command, args []string) {
				ctx := cmd.Context()

				sub, err := utils.GetSubscription(ctx, "./secrets/devlab-404500-b2f9a112f6db.json", "devlab-404500", "gmail-notification-sub")
				if err != nil {
					log.Fatalf("Unable to get subscription: %v", err)
				}

				if err := sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
					log.Printf("Received message: %s", string(m.Data))
					m.Ack()
				}); err != nil {
					log.Fatalf("Unable to receive messages: %v", err)
				}
			},
		},
	)

	if err := cli.ExecuteContext(context.Background()); err != nil {
		log.Fatal(err)
	}
}
