package http

import (
	"context"
	"log"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gnailuy/amiglot-api/internal/config"
	"github.com/gnailuy/amiglot-api/internal/repository"
	"github.com/gnailuy/amiglot-api/internal/service"
)

type authHandler struct {
	svc *service.AuthService
}

func registerAuthRoutes(api huma.API, cfg config.Config, pool *pgxpool.Pool) {
	repo := repository.NewAuthRepository(pool)
	svc := service.NewAuthService(cfg, repo)
	h := &authHandler{svc: svc}

	huma.Post(api, "/auth/magic-link", h.requestMagicLink)
	huma.Post(api, "/auth/verify", h.verifyMagicLink)
	huma.Post(api, "/auth/logout", h.logout)
}

type magicLinkRequest struct {
	Body struct {
		Email string `json:"email"`
	}
}

type magicLinkResponse struct {
	Body struct {
		Ok          bool    `json:"ok"`
		DevLoginURL *string `json:"dev_login_url,omitempty"`
	}
}

func (h *authHandler) requestMagicLink(ctx context.Context, input *magicLinkRequest) (*magicLinkResponse, error) {
	devLoginURL, err := h.svc.RequestMagicLink(ctx, input.Body.Email)
	if err != nil {
		return nil, toHumaError(ctx, err)
	}

	if devLoginURL != nil {
		log.Printf("dev magic link for %s: %s", input.Body.Email, *devLoginURL)
	} else {
		log.Printf("magic link requested for %s", input.Body.Email)
	}

	return &magicLinkResponse{Body: struct {
		Ok          bool    `json:"ok"`
		DevLoginURL *string `json:"dev_login_url,omitempty"`
	}{Ok: true, DevLoginURL: devLoginURL}}, nil
}

type verifyRequest struct {
	Body struct {
		Token string `json:"token"`
	}
}

type verifyResponse struct {
	Body struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}
}

func (h *authHandler) verifyMagicLink(ctx context.Context, input *verifyRequest) (*verifyResponse, error) {
	accessToken, userID, email, err := h.svc.VerifyMagicLink(ctx, input.Body.Token)
	if err != nil {
		return nil, toHumaError(ctx, err)
	}

	resp := &verifyResponse{}
	resp.Body.AccessToken = accessToken
	resp.Body.User.ID = userID
	resp.Body.User.Email = email

	return resp, nil
}

type logoutResponse struct {
	Ok bool `json:"ok"`
}

func (h *authHandler) logout(ctx context.Context, input *struct{}) (*logoutResponse, error) {
	return &logoutResponse{Ok: true}, nil
}
