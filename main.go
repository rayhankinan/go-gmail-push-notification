package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"go-gmail-notification/models"
	"go-gmail-notification/usecase"
	"go-gmail-notification/utils"

	"cloud.google.com/go/pubsub"
	"github.com/spf13/cobra"
	"google.golang.org/api/gmail/v1"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	user            = "me"
	dsn             = "host=localhost port=5432 user=postgres password=password dbname=email-push-notifications sslmode=disable"
	secretPath      = "./secrets/client_secret_494104920232-6smddih7ahdcgpvd6fun9srertn4l5qs.apps.googleusercontent.com.json"
	tokenPath       = "./secrets/token.json"
	credentialsPath = "./secrets/devlab-404500-b2f9a112f6db.json"
	projectID       = "devlab-404500"
	topicID         = "gmail-notification"
	subscriptionID  = "gmail-notification-sub"
)

var (
	labelIds = []string{"UNREAD"}
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	cli := &cobra.Command{}

	cli.AddCommand(
		&cobra.Command{
			Use:   "migrate",
			Short: "Migrate schema",
			Long:  "Migrates the database schema",
			Run: func(cmd *cobra.Command, args []string) {
				db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
				if err != nil {
					log.Fatalf("Failed to connect to database: %v", err)
				}

				if err := db.AutoMigrate(&models.Email{}); err != nil {
					log.Fatalf("Failed to migrate schema: %v", err)
				}

				log.Println("Database migrated successfully")
			},
		},
		&cobra.Command{
			Use:   "start-watch",
			Short: "Start a watch process",
			Long:  "Starts a watch process that monitors gmail inbox",
			Run: func(cmd *cobra.Command, args []string) {
				ctx := cmd.Context()

				db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
				if err != nil {
					log.Fatalf("Failed to connect to database: %v", err)
				}
				u := usecase.NewEmailUsecase(db)

				srv, err := utils.GetClient(ctx, secretPath, tokenPath)
				if err != nil {
					log.Fatalf("Unable to get Gmail client: %v", err)
				}

				profile, err := srv.Users.GetProfile(user).Do()
				if err != nil {
					log.Fatalf("Unable to get user profile: %v", err)
				}

				request := &gmail.WatchRequest{
					LabelIds:            labelIds,
					LabelFilterBehavior: "include",
					TopicName:           fmt.Sprintf("projects/%s/topics/%s", projectID, topicID),
				}
				r, err := srv.Users.Watch(user, request).Do()
				if err != nil {
					log.Fatalf("Unable to watch inbox: %v", err)
				}

				if err := u.CreateOrUpdate(
					ctx,
					models.Email{
						Email:           profile.EmailAddress,
						Expiration:      r.Expiration,
						LatestHistoryID: r.HistoryId,
					},
				); err != nil {
					log.Fatalf("Failed to create email record: %v", err)
				}

				log.Printf("Watch started successfully for %s\n", profile.EmailAddress)
			},
		},
		&cobra.Command{
			Use:   "stop-watch",
			Short: "Stop a watch process",
			Long:  "Stops a watch process that monitors gmail inbox",
			Run: func(cmd *cobra.Command, args []string) {
				ctx := cmd.Context()

				db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
				if err != nil {
					log.Fatalf("Failed to connect to database: %v", err)
				}
				u := usecase.NewEmailUsecase(db)

				srv, err := utils.GetClient(ctx, secretPath, tokenPath)
				if err != nil {
					log.Fatalf("Unable to get Gmail client: %v", err)
				}

				profile, err := srv.Users.GetProfile(user).Do()
				if err != nil {
					log.Fatalf("Unable to get user profile: %v", err)
				}

				if err := srv.Users.Stop(user).Do(); err != nil {
					log.Fatalf("Unable to stop watch: %v", err)
				}

				if err := u.Delete(ctx, profile.EmailAddress); err != nil {
					log.Fatalf("Failed to delete email record: %v", err)
				}

				log.Printf("Watch stopped successfull for %s\n", profile.EmailAddress)
			},
		},
		&cobra.Command{
			Use:   "process-message",
			Short: "Process a message",
			Long:  "Processes a message from the gmail inbox",
			Run: func(cmd *cobra.Command, args []string) {
				ctx := cmd.Context()

				db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
				if err != nil {
					log.Fatalf("Failed to connect to database: %v", err)
				}
				u := usecase.NewEmailUsecase(db)

				srv, err := utils.GetClient(ctx, secretPath, tokenPath)
				if err != nil {
					log.Fatalf("Unable to get Gmail client: %v", err)
				}

				sub, err := utils.GetSubscription(ctx, credentialsPath, projectID, subscriptionID)
				if err != nil {
					log.Fatalf("Unable to get subscription: %v", err)
				}

				if err := sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
					message := models.Message{}
					if err := json.Unmarshal(m.Data, &message); err != nil {
						log.Printf("Failed to unmarshal message: %v", err)
						return
					}

					currentEmail, err := u.GetEmailByAddress(ctx, message.EmailAddress)
					if err != nil {
						log.Printf("Failed to find email record: %v", err)
						return
					}

					envelopes, err := utils.GetUniqueMessagesFromHistory(srv, user, currentEmail.LatestHistoryID)
					if err != nil {
						log.Printf("Failed to get unique messages from history: %v", err)
						return
					}

					for _, envelope := range envelopes {
						message, err := srv.Users.Messages.Get(user, envelope.Id).Format("full").Do()
						if err != nil {
							log.Printf("Failed to get message: %v", err)
							return
						}

						value, err := json.MarshalIndent(message, "", "\t")
						if err != nil {
							log.Printf("Failed to marshal message: %v", err)
							return
						}

						log.Printf("Received a new message: %s\n", value)
					}

					if err := u.UpdateLastHistoryID(ctx, message.EmailAddress, currentEmail.LatestHistoryID, message.HistoryID); err != nil {
						log.Printf("Failed to update last history ID: %v", err)
						return
					}

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
