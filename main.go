package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go-gmail-notification/models"
	"go-gmail-notification/utils"
	"log"

	"cloud.google.com/go/pubsub"
	"github.com/spf13/cobra"
	"google.golang.org/api/gmail/v1"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

				srv, err := utils.GetClient(ctx, secretPath, tokenPath)
				if err != nil {
					log.Fatalf("Unable to get Gmail client: %v", err)
				}

				profile, err := srv.Users.GetProfile(user).Do()
				if err != nil {
					log.Fatalf("Unable to get user profile: %v", err)
				}

				request := &gmail.WatchRequest{
					LabelIds:            []string{"INBOX", "UNREAD"},
					LabelFilterBehavior: "include",
					TopicName:           fmt.Sprintf("projects/%s/topics/%s", projectID, topicID),
				}
				r, err := srv.Users.Watch(user, request).Do()
				if err != nil {
					log.Fatalf("Unable to watch inbox: %v", err)
				}

				email := models.Email{
					Email:           profile.EmailAddress,
					Expiration:      r.Expiration,
					LatestHistoryID: r.HistoryId,
				}
				if err := db.Clauses(
					clause.OnConflict{
						OnConstraint: "uni_emails_email",
						DoUpdates: clause.AssignmentColumns(
							[]string{"created_at", "updated_at", "deleted_at", "expiration", "latest_history_id"},
						),
					},
				).Create(&email).Error; err != nil {
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

				if err := db.Where("email = ?", profile.EmailAddress).Delete(&models.Email{}).Error; err != nil {
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

				sub, err := utils.GetSubscription(ctx, credentialsPath, projectID, subscriptionID)
				if err != nil {
					log.Fatalf("Unable to get subscription: %v", err)
				}

				if err := sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
					message := models.Message{}
					if err := json.Unmarshal(m.Data, &message); err != nil {
						log.Printf("Failed to unmarshal message: %v", err)
						m.Nack()
						return
					}

					lastEmail := models.Email{}
					if err := db.Where("email = ?", message.EmailAddress).First(&lastEmail).Error; err != nil {
						log.Printf("Failed to find email record: %v", err)
						m.Nack()
						return
					}

					// TODO: Get history list from Gmail API and fetch messages
					log.Printf("Processing message: %+v\n", message)

					if err := db.Model(&models.Email{}).Where("email = ?", message.EmailAddress).Updates(models.Email{
						LatestHistoryID: message.HistoryID,
					}).Error; err != nil {
						log.Printf("Failed to update email record: %v", err)
						m.Nack()
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
