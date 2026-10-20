package services

import (
	"analytics-service/internal/models"
	"context"
)

// ClickRepository is the persistence this service needs.
type ClickRepository interface {
	InsertBatch(ctx context.Context, clicks []models.Click) error
	GetCodeAnalytics(ctx context.Context, code string) (*models.CodeAnalytics, error)
}

type ClickService struct {
	repository ClickRepository
}

func NewClickService(repository ClickRepository) *ClickService {
	return &ClickService{repository: repository}
}

// InsertClickEventBatch converts rabbitmq messages and stores them to the db.
func (s *ClickService) InsertClickEventBatch(ctx context.Context, events []models.ClickEvent) error {
	if len(events) == 0 {
		return nil
	}

	clicks := make([]models.Click, len(events))
	for i, event := range events {
		clicks[i] = event.ToClick()
	}

	return s.repository.InsertBatch(ctx, clicks)
}

// GetCodeAnalytics returns aggregate click statistics for one code.
func (s *ClickService) GetCodeAnalytics(ctx context.Context, code string) (*models.CodeAnalytics, error) {
	return s.repository.GetCodeAnalytics(ctx, code)
}
