package usecase

import (
	"context"
	"fmt"

	"go-gmail-notification/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EmailUsecase interface {
	CreateOrUpdate(ctx context.Context, email models.Email) error
	GetEmailByAddress(ctx context.Context, address string) (models.Email, error)
	UpdateLastHistoryID(ctx context.Context, address string, lastHistoryID, newHistoryID uint64) error
	Delete(ctx context.Context, address string) error
}

type EmailUsecaseImpl struct {
	db *gorm.DB
}

func NewEmailUsecase(db *gorm.DB) EmailUsecase {
	return &EmailUsecaseImpl{db: db}
}

func (m *EmailUsecaseImpl) CreateOrUpdate(ctx context.Context, email models.Email) error {
	if err := m.db.Clauses(
		clause.OnConflict{
			OnConstraint: "uni_emails_email",
			DoUpdates: clause.AssignmentColumns(
				[]string{"created_at", "updated_at", "deleted_at", "expiration", "latest_history_id"},
			),
		},
	).Create(&email).Error; err != nil {
		return fmt.Errorf(("failed to create or update email: %w"), err)
	}

	return nil
}

func (m *EmailUsecaseImpl) GetEmailByAddress(ctx context.Context, address string) (models.Email, error) {
	email := models.Email{}
	if err := m.db.Where("email = ?", address).First(&email).Error; err != nil {
		return models.Email{}, fmt.Errorf("failed to get email by address: %w", err)
	}
	return email, nil
}

func (m *EmailUsecaseImpl) UpdateLastHistoryID(ctx context.Context, address string, lastHistoryID, newHistoryID uint64) error {
	if err := m.db.Model(&models.Email{}).Where("email = ?", address).Updates(models.Email{
		LatestHistoryID: max(lastHistoryID, newHistoryID),
	}).Error; err != nil {
		return fmt.Errorf("failed to update last history ID: %w", err)
	}
	return nil
}

func (m *EmailUsecaseImpl) Delete(ctx context.Context, address string) error {
	if err := m.db.Where("email = ?", address).Delete(&models.Email{}).Error; err != nil {
		return fmt.Errorf("failed to delete email: %w", err)
	}
	return nil
}
