package http

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/gnailuy/amiglot-api/internal/i18n"
	"github.com/gnailuy/amiglot-api/internal/service"
)

func toHumaError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	if svcErr, ok := err.(*service.Error); ok {
		switch svcErr.Status {
		case 400:
			return huma.Error400BadRequest(i18n.T(ctx, svcErr.Key))
		case 401:
			return huma.Error401Unauthorized(i18n.T(ctx, svcErr.Key))
		case 403:
			return huma.Error403Forbidden(i18n.T(ctx, svcErr.Key))
		case 404:
			return huma.Error404NotFound(i18n.T(ctx, svcErr.Key))
		case 409:
			return huma.Error409Conflict(i18n.T(ctx, svcErr.Key))
		case 422:
			return huma.Error422UnprocessableEntity(i18n.T(ctx, svcErr.Key))
		case 429:
			return huma.Error429TooManyRequests(i18n.T(ctx, svcErr.Key))
		case 503:
			return huma.Error503ServiceUnavailable(i18n.T(ctx, svcErr.Key))
		default:
			return huma.Error500InternalServerError(i18n.T(ctx, svcErr.Key))
		}
	}

	return huma.Error500InternalServerError(i18n.T(ctx, "errors.internal_server_error"))
}
