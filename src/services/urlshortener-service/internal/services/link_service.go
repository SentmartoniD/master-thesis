package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/url"
	"time"
	"urlshortener-service/internal/config"
	"urlshortener-service/internal/models"
	"urlshortener-service/internal/repositories"
)

const (
	codeLength      = 7
	MaxCodeAttempts = 100
)

var (
	ErrorInvalidTargetURL = errors.New("target URL must be an absolute http or https URL")
	ErrorCodeExhausted    = errors.New("could not allocate a unique code")
	ErrorLinkNotFound     = errors.New("link not found")
)

type LinkRepository interface {
	Create(ctx context.Context, code string, targetURL string, expiresAt *time.Time) (*models.Link, error)
	GetByCode(ctx context.Context, code string) (*models.Link, error)
}

type LinkService struct {
	repository LinkRepository
	cfg        *config.Config
}

func NewLinkService(repository LinkRepository, cfg *config.Config) *LinkService {
	return &LinkService{repository: repository, cfg: cfg}
}

// CreateLink creates a link.
func (s *LinkService) CreateLink(ctx context.Context, req models.CreateLinkRequest) (*models.LinkResponse, error) {

	// valide fields
	err := validateTargetURL(req.URL)
	if err != nil {
		return nil, err
	}

	// the generated code could already be used
	// so generate in a loop until it succeeds
	for attempt := 0; attempt < MaxCodeAttempts; attempt++ {
		code, err := generateCode(codeLength)
		if err != nil {
			return nil, fmt.Errorf("failed generating code: %w", err)
		}

		link, err := s.repository.Create(ctx, code, req.URL, req.ExpiresAt)
		// success then return link
		if err == nil {
			return link.ToLinkResponse(s.cfg.ShortBaseURL), nil
			// code already used then continue
		} else if errors.Is(err, repositories.ErrorCodeUsed) {
			continue
		}

		return nil, err
	}

	return nil, ErrorCodeExhausted
}

// GetLink returns a link by its code.
func (s *LinkService) GetLink(ctx context.Context, code string) (*models.LinkResponse, error) {
	link, err := s.repository.GetByCode(ctx, code)
	if err != nil {
		if errors.Is(err, repositories.ErrorLinkNotFound) {
			return nil, ErrorLinkNotFound
		}
		return nil, err
	}

	return link.ToLinkResponse(s.cfg.ShortBaseURL), nil
}

// validateTargetURL checks if url is valid
func validateTargetURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ErrorInvalidTargetURL
	}
	if parsed.Host == "" {
		return ErrorInvalidTargetURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrorInvalidTargetURL
	}
	return nil
}

// generateCode returns a random base62 string of the given length.
func generateCode(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	codeAlphabet := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	for i, b := range buf {
		buf[i] = codeAlphabet[int(b)%len(codeAlphabet)]
	}

	return string(buf), nil
}
