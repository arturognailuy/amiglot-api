package http

import (
	"context"
	"errors"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gnailuy/amiglot-api/internal/service"
)

func TestToHumaError_AllCodes(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"400", &service.Error{Status: 400, Key: "errors.bad_request"}, 400},
		{"401", &service.Error{Status: 401, Key: "errors.unauthorized"}, 401},
		{"403", &service.Error{Status: 403, Key: "errors.profile_incomplete"}, 403},
		{"409", &service.Error{Status: 409, Key: "errors.conflict"}, 409},
		{"422", &service.Error{Status: 422, Key: "errors.no_target_languages"}, 422},
		{"503", &service.Error{Status: 503, Key: "errors.database_unavailable"}, 503},
		{"500 default", &service.Error{Status: 500, Key: "errors.internal_server_error"}, 500},
		{"999 fallback", &service.Error{Status: 999, Key: "errors.unknown"}, 500},
		{"non-service error", errors.New("generic"), 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			herr := toHumaError(ctx, tt.err)
			if herr == nil {
				t.Fatal("expected non-nil error")
			}
			statusErr, ok := herr.(huma.StatusError)
			if !ok {
				t.Fatalf("expected huma.StatusError, got %T", herr)
			}
			if statusErr.GetStatus() != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, statusErr.GetStatus())
			}
		})
	}
}
