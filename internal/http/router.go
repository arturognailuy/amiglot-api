package http

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gnailuy/amiglot-api/internal/config"
)

// Router builds the HTTP routes.
func Router(cfg config.Config, pool *pgxpool.Pool) http.Handler {
	root := http.NewServeMux()
	apiMux := http.NewServeMux()
	api := humago.New(apiMux, huma.DefaultConfig("Amiglot API", "1.0.0"))

	huma.Get(api, "/healthz", func(ctx context.Context, input *struct{}) (*struct {
		Ok bool `json:"ok"`
	}, error) {
		return &struct {
			Ok bool `json:"ok"`
		}{Ok: true}, nil
	})

	registerAuthRoutes(api, cfg, pool)
	registerProfileRoutes(api, pool)

	root.Handle("/api/v1/", http.StripPrefix("/api/v1", apiMux))

	return root
}
