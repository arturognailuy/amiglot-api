package http

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gnailuy/amiglot-api/internal/model"
	"github.com/gnailuy/amiglot-api/internal/repository"
	"github.com/gnailuy/amiglot-api/internal/service"
)

type profileHandler struct {
	svc *service.ProfileService
}

type profilePayload struct {
	Handle       string  `json:"handle"`
	BirthYear    *int    `json:"birth_year,omitempty"`
	BirthMonth   *int16  `json:"birth_month,omitempty"`
	CountryCode  *string `json:"country_code,omitempty"`
	Timezone     string  `json:"timezone"`
	Discoverable bool    `json:"discoverable"`
}

type languagePayload struct {
	LanguageCode string  `json:"language_code"`
	Level        int16   `json:"level"`
	IsNative     bool    `json:"is_native"`
	IsTarget     bool    `json:"is_target"`
	Description  *string `json:"description,omitempty"`
	Order        int     `json:"order,omitempty"`
}

type availabilityPayload struct {
	Weekday        int16  `json:"weekday"`
	StartLocalTime string `json:"start_local_time"`
	EndLocalTime   string `json:"end_local_time"`
	Timezone       string `json:"timezone"`
	Order          int    `json:"order,omitempty"`
}

type userPayload struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type profileResponse struct {
	Body struct {
		User         userPayload           `json:"user"`
		Profile      profilePayload        `json:"profile"`
		Languages    []languagePayload     `json:"languages"`
		Availability []availabilityPayload `json:"availability"`
	}
}

type languagesPutResponse struct {
	Body struct {
		Languages []languagePayload `json:"languages"`
	}
}

type availabilityPutResponse struct {
	Body struct {
		Availability []availabilityPayload `json:"availability"`
	}
}

type profileUpdateRequest struct {
	UserID string `header:"X-User-Id"`
	Body   struct {
		Handle      string  `json:"handle"`
		BirthYear   *int    `json:"birth_year,omitempty"`
		BirthMonth  *int16  `json:"birth_month,omitempty"`
		CountryCode *string `json:"country_code,omitempty"`
		Timezone    string  `json:"timezone"`
	}
}

type profileGetRequest struct {
	UserID string `header:"X-User-Id"`
}

type languagesPutRequest struct {
	UserID string `header:"X-User-Id"`
	Body   struct {
		Languages []languagePayload `json:"languages"`
	}
}

type availabilityPutRequest struct {
	UserID string `header:"X-User-Id"`
	Body   struct {
		Availability []availabilityPayload `json:"availability"`
	}
}

type handleCheckRequest struct {
	UserID string `header:"X-User-Id"`
	Handle string `query:"handle"`
}

type handleCheckResponse struct {
	Body struct {
		Available bool `json:"available"`
	}
}

func registerProfileRoutes(api huma.API, pool *pgxpool.Pool) {
	repo := repository.NewProfileRepository(pool)
	svc := service.NewProfileService(repo)
	h := &profileHandler{svc: svc}

	huma.Get(api, "/profile", h.getProfile)
	huma.Get(api, "/profile/handle/check", h.checkHandleAvailability)
	huma.Put(api, "/profile", h.putProfile)
	huma.Put(api, "/profile/languages", h.putLanguages)
	huma.Put(api, "/profile/availability", h.putAvailability)
}

func (h *profileHandler) getProfile(ctx context.Context, input *profileGetRequest) (*profileResponse, error) {
	user, profile, languages, availability, err := h.svc.GetProfile(ctx, input.UserID)
	if err != nil {
		return nil, toHumaError(ctx, err)
	}

	return &profileResponse{
		Body: struct {
			User         userPayload           `json:"user"`
			Profile      profilePayload        `json:"profile"`
			Languages    []languagePayload     `json:"languages"`
			Availability []availabilityPayload `json:"availability"`
		}{
			User:         toUserPayload(user),
			Profile:      toProfilePayload(profile),
			Languages:    toLanguagePayloads(languages),
			Availability: toAvailabilityPayloads(availability),
		},
	}, nil
}

func (h *profileHandler) checkHandleAvailability(ctx context.Context, input *handleCheckRequest) (*handleCheckResponse, error) {
	available, err := h.svc.CheckHandleAvailability(ctx, input.UserID, input.Handle)
	if err != nil {
		return nil, toHumaError(ctx, err)
	}

	return &handleCheckResponse{
		Body: struct {
			Available bool `json:"available"`
		}{Available: available},
	}, nil
}

func (h *profileHandler) putProfile(ctx context.Context, input *profileUpdateRequest) (*profileResponse, error) {
	profile := model.Profile{
		Handle:      input.Body.Handle,
		BirthYear:   input.Body.BirthYear,
		BirthMonth:  input.Body.BirthMonth,
		CountryCode: input.Body.CountryCode,
		Timezone:    input.Body.Timezone,
	}

	user, updatedProfile, languages, availability, err := h.svc.UpdateProfile(ctx, input.UserID, profile)
	if err != nil {
		return nil, toHumaError(ctx, err)
	}

	return &profileResponse{
		Body: struct {
			User         userPayload           `json:"user"`
			Profile      profilePayload        `json:"profile"`
			Languages    []languagePayload     `json:"languages"`
			Availability []availabilityPayload `json:"availability"`
		}{
			User:         toUserPayload(user),
			Profile:      toProfilePayload(updatedProfile),
			Languages:    toLanguagePayloads(languages),
			Availability: toAvailabilityPayloads(availability),
		},
	}, nil
}

func (h *profileHandler) putLanguages(ctx context.Context, input *languagesPutRequest) (*languagesPutResponse, error) {
	languages := toModelLanguages(input.Body.Languages)

	updated, err := h.svc.UpdateLanguages(ctx, input.UserID, languages)
	if err != nil {
		return nil, toHumaError(ctx, err)
	}

	return &languagesPutResponse{
		Body: struct {
			Languages []languagePayload `json:"languages"`
		}{Languages: toLanguagePayloads(updated)},
	}, nil
}

func (h *profileHandler) putAvailability(ctx context.Context, input *availabilityPutRequest) (*availabilityPutResponse, error) {
	slots := toModelAvailability(input.Body.Availability)

	updated, err := h.svc.UpdateAvailability(ctx, input.UserID, slots)
	if err != nil {
		return nil, toHumaError(ctx, err)
	}

	return &availabilityPutResponse{
		Body: struct {
			Availability []availabilityPayload `json:"availability"`
		}{Availability: toAvailabilityPayloads(updated)},
	}, nil
}

func toUserPayload(user model.User) userPayload {
	return userPayload{ID: user.ID, Email: user.Email}
}

func toProfilePayload(profile model.Profile) profilePayload {
	return profilePayload{
		Handle:       profile.Handle,
		BirthYear:    profile.BirthYear,
		BirthMonth:   profile.BirthMonth,
		CountryCode:  profile.CountryCode,
		Timezone:     profile.Timezone,
		Discoverable: profile.Discoverable,
	}
}

func toLanguagePayloads(languages []model.Language) []languagePayload {
	if len(languages) == 0 {
		return []languagePayload{}
	}
	payloads := make([]languagePayload, 0, len(languages))
	for _, lang := range languages {
		payloads = append(payloads, languagePayload{
			LanguageCode: lang.LanguageCode,
			Level:        lang.Level,
			IsNative:     lang.IsNative,
			IsTarget:     lang.IsTarget,
			Description:  lang.Description,
			Order:        lang.SortOrder,
		})
	}
	return payloads
}

func toAvailabilityPayloads(slots []model.AvailabilitySlot) []availabilityPayload {
	if len(slots) == 0 {
		return []availabilityPayload{}
	}
	payloads := make([]availabilityPayload, 0, len(slots))
	for _, slot := range slots {
		payloads = append(payloads, availabilityPayload{
			Weekday:        slot.Weekday,
			StartLocalTime: slot.StartLocalTime,
			EndLocalTime:   slot.EndLocalTime,
			Timezone:       slot.Timezone,
			Order:          slot.SortOrder,
		})
	}
	return payloads
}

func toModelLanguages(languages []languagePayload) []model.Language {
	if len(languages) == 0 {
		return []model.Language{}
	}
	models := make([]model.Language, 0, len(languages))
	for _, lang := range languages {
		models = append(models, model.Language{
			LanguageCode: lang.LanguageCode,
			Level:        lang.Level,
			IsNative:     lang.IsNative,
			IsTarget:     lang.IsTarget,
			Description:  lang.Description,
			SortOrder:    lang.Order,
		})
	}
	return models
}

func toModelAvailability(slots []availabilityPayload) []model.AvailabilitySlot {
	if len(slots) == 0 {
		return []model.AvailabilitySlot{}
	}
	models := make([]model.AvailabilitySlot, 0, len(slots))
	for _, slot := range slots {
		models = append(models, model.AvailabilitySlot{
			Weekday:        slot.Weekday,
			StartLocalTime: slot.StartLocalTime,
			EndLocalTime:   slot.EndLocalTime,
			Timezone:       slot.Timezone,
			SortOrder:      slot.Order,
		})
	}
	return models
}
