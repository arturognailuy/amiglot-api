package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/gnailuy/amiglot-api/internal/config"
	"github.com/gnailuy/amiglot-api/internal/repository"
)

type AuthService struct {
	repo *repository.AuthRepository
	cfg  config.Config
}

func NewAuthService(cfg config.Config, repo *repository.AuthRepository) *AuthService {
	return &AuthService{cfg: cfg, repo: repo}
}

func (s *AuthService) RequestMagicLink(ctx context.Context, email string) (*string, error) {
	if s.repo == nil || s.repo.Pool() == nil {
		return nil, &Error{Status: 503, Key: "errors.database_unavailable"}
	}

	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, &Error{Status: 400, Key: "errors.email_required"}
	}

	userID, err := s.repo.EnsureUser(ctx, email)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.failed_load_user", Err: err}
	}

	token, tokenHash, err := GenerateToken()
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.failed_generate_token", Err: err}
	}

	expiresAt := time.Now().Add(s.cfg.MagicLinkTTL)
	if err := s.repo.CreateMagicLinkToken(ctx, userID, tokenHash, expiresAt); err != nil {
		return nil, &Error{Status: 500, Key: "errors.failed_store_token", Err: err}
	}

	if s.cfg.Env == "dev" {
		link := s.cfg.MagicLinkBaseURL + "?token=" + token
		return &link, nil
	}

	return nil, nil
}

func (s *AuthService) VerifyMagicLink(ctx context.Context, token string) (string, string, string, error) {
	if s.repo == nil || s.repo.Pool() == nil {
		return "", "", "", &Error{Status: 503, Key: "errors.database_unavailable"}
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", "", "", &Error{Status: 400, Key: "errors.token_required"}
	}

	tokenHash := sha256.Sum256([]byte(token))

	userID, email, err := s.repo.ConsumeMagicLinkToken(ctx, tokenHash[:])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", &Error{Status: 401, Key: "errors.token_invalid"}
		}
		return "", "", "", &Error{Status: 500, Key: "errors.failed_load_token", Err: err}
	}

	accessToken, _, err := GenerateToken()
	if err != nil {
		return "", "", "", &Error{Status: 500, Key: "errors.failed_generate_access_token", Err: err}
	}

	return accessToken, userID, email, nil
}

func GenerateToken() (string, []byte, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", nil, err
	}
	encoded := base64.RawURLEncoding.EncodeToString(bytes)
	hash := sha256.Sum256([]byte(encoded))
	return encoded, hash[:], nil
}
