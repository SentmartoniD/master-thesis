package services

import (
	"context"
	"errors"
	"redirect-service/internal/repositories"
)

var (
	ErrorLinkNotFound = errors.New("link not found")
)

type LinkRepository interface {
	GetLinkByCode(ctx context.Context, code string) (string, error)
}

type LinkService struct {
	repository LinkRepository
}

func NewLinkService(repository LinkRepository) *LinkService {
	return &LinkService{repository: repository}
}

// GetLinkByCode returns link for the specified code
func (s *LinkService) GetLinkByCode(ctx context.Context, code string) (string, error) {
	targetURL, err := s.repository.GetLinkByCode(ctx, code)
	if err != nil {
		if errors.Is(err, repositories.ErrorLinkNotFound) {
			return "", ErrorLinkNotFound
		}
		return "", err
	}

	return targetURL, nil
}
