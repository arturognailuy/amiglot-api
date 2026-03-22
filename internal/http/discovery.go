package http

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gnailuy/amiglot-api/internal/config"
	"github.com/gnailuy/amiglot-api/internal/repository"
	"github.com/gnailuy/amiglot-api/internal/service"
)

type discoveryHandler struct {
	svc *service.DiscoveryService
}

type discoverRequest struct {
	UserID string `header:"X-User-Id"`
	Cursor string `query:"cursor"`
	Limit  int    `query:"limit"`
}

type matchLanguagePayload struct {
	LanguageCode string `json:"language_code"`
	Level        int16  `json:"level"`
	IsNative     bool   `json:"is_native"`
	LearnerLevel int16  `json:"learner_level"`
}

type bridgeLanguagePayload struct {
	LanguageCode string `json:"language_code"`
	Level        int16  `json:"level"`
}

type overlapSlotPayload struct {
	Weekday        int16  `json:"weekday"`
	StartUTC       string `json:"start_utc"`
	EndUTC         string `json:"end_utc"`
	OverlapMinutes int    `json:"overlap_minutes"`
}

type matchItemPayload struct {
	UserID              string                  `json:"user_id"`
	Handle              string                  `json:"handle"`
	CountryCode         *string                 `json:"country_code,omitempty"`
	Age                 *int                    `json:"age,omitempty"`
	MutualTeach         []matchLanguagePayload  `json:"mutual_teach"`
	MutualLearn         []matchLanguagePayload  `json:"mutual_learn"`
	BridgeLanguages     []bridgeLanguagePayload `json:"bridge_languages"`
	AvailabilityOverlap []overlapSlotPayload    `json:"availability_overlap"`
	TotalOverlapMinutes int                     `json:"total_overlap_minutes"`
}

type discoverResponse struct {
	Body struct {
		Items      []matchItemPayload `json:"items"`
		NextCursor *string            `json:"next_cursor"`
	}
}

func registerDiscoveryRoutes(api huma.API, cfg config.Config, pool *pgxpool.Pool) {
	minOverlap := 60
	if raw := cfg.MatchMinOverlapMinutes; raw > 0 {
		minOverlap = raw
	}

	repo := repository.NewDiscoveryRepository(pool)
	svc := service.NewDiscoveryService(repo, minOverlap)
	h := &discoveryHandler{svc: svc}

	huma.Get(api, "/matches/discover", h.discover)
}

func (h *discoveryHandler) discover(ctx context.Context, input *discoverRequest) (*discoverResponse, error) {
	var cursor *string
	if input.Cursor != "" {
		cursor = &input.Cursor
	}

	result, err := h.svc.Discover(ctx, input.UserID, cursor, input.Limit)
	if err != nil {
		return nil, toHumaError(ctx, err)
	}

	items := make([]matchItemPayload, 0, len(result.Items))
	for _, m := range result.Items {
		item := matchItemPayload{
			UserID:              m.UserID,
			Handle:              m.Handle,
			CountryCode:         m.CountryCode,
			Age:                 m.Age,
			TotalOverlapMinutes: m.TotalOverlapMinutes,
			MutualTeach:         toMatchLanguagePayloads(m.MutualTeach),
			MutualLearn:         toMatchLanguagePayloads(m.MutualLearn),
			BridgeLanguages:     toBridgeLanguagePayloads(m.BridgeLanguages),
			AvailabilityOverlap: toOverlapSlotPayloads(m.AvailabilityOverlap),
		}
		items = append(items, item)
	}

	return &discoverResponse{
		Body: struct {
			Items      []matchItemPayload `json:"items"`
			NextCursor *string            `json:"next_cursor"`
		}{
			Items:      items,
			NextCursor: result.NextCursor,
		},
	}, nil
}

func toMatchLanguagePayloads(langs []service.MatchLanguage) []matchLanguagePayload {
	if len(langs) == 0 {
		return []matchLanguagePayload{}
	}
	payloads := make([]matchLanguagePayload, 0, len(langs))
	for _, l := range langs {
		payloads = append(payloads, matchLanguagePayload{
			LanguageCode: l.LanguageCode,
			Level:        l.Level,
			IsNative:     l.IsNative,
			LearnerLevel: l.LearnerLevel,
		})
	}
	return payloads
}

func toBridgeLanguagePayloads(langs []service.BridgeLanguage) []bridgeLanguagePayload {
	if len(langs) == 0 {
		return []bridgeLanguagePayload{}
	}
	payloads := make([]bridgeLanguagePayload, 0, len(langs))
	for _, l := range langs {
		payloads = append(payloads, bridgeLanguagePayload{
			LanguageCode: l.LanguageCode,
			Level:        l.Level,
		})
	}
	return payloads
}

func toOverlapSlotPayloads(slots []service.OverlapSlot) []overlapSlotPayload {
	if len(slots) == 0 {
		return []overlapSlotPayload{}
	}
	payloads := make([]overlapSlotPayload, 0, len(slots))
	for _, s := range slots {
		payloads = append(payloads, overlapSlotPayload{
			Weekday:        s.Weekday,
			StartUTC:       s.StartUTC,
			EndUTC:         s.EndUTC,
			OverlapMinutes: s.OverlapMinutes,
		})
	}
	return payloads
}
