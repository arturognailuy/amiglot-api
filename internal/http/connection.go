package http

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gnailuy/amiglot-api/internal/config"
	"github.com/gnailuy/amiglot-api/internal/repository"
	"github.com/gnailuy/amiglot-api/internal/service"
)

type connectionHandler struct {
	svc *service.ConnectionService
}

// --- Request / Response types ---

type createMatchRequestInput struct {
	UserID string `header:"X-User-Id"`
	Body   struct {
		RecipientID    string  `json:"recipient_id" required:"true"`
		InitialMessage *string `json:"initial_message,omitempty"`
	}
}

type createMatchRequestResponse struct {
	Body struct {
		ID             string  `json:"id"`
		RequesterID    string  `json:"requester_id"`
		RecipientID    string  `json:"recipient_id"`
		Status         string  `json:"status"`
		InitialMessage *string `json:"initial_message,omitempty"`
		CreatedAt      string  `json:"created_at"`
	}
}

type listMatchRequestsInput struct {
	UserID    string `header:"X-User-Id"`
	Direction string `query:"direction"`
	Status    string `query:"status"`
	Cursor    string `query:"cursor"`
	Limit     int    `query:"limit"`
}

type requestUserPayload struct {
	UserID      string  `json:"user_id"`
	Handle      string  `json:"handle"`
	CountryCode *string `json:"country_code,omitempty"`
	Age         *int    `json:"age,omitempty"`
}

type matchRequestPayload struct {
	ID           string             `json:"id"`
	Requester    requestUserPayload `json:"requester"`
	Recipient    requestUserPayload `json:"recipient"`
	Status       string             `json:"status"`
	MessageCount int                `json:"message_count"`
	LastMessageAt *string           `json:"last_message_at,omitempty"`
	CreatedAt    string             `json:"created_at"`
}

type listMatchRequestsResponse struct {
	Body struct {
		Items      []matchRequestPayload `json:"items"`
		NextCursor *string               `json:"next_cursor"`
	}
}

type getMatchRequestInput struct {
	UserID    string `header:"X-User-Id"`
	RequestID string `path:"id"`
}

type getMatchRequestResponse struct {
	Body matchRequestPayload
}

type resolveRequestInput struct {
	UserID    string `header:"X-User-Id"`
	RequestID string `path:"id"`
}

type acceptResponse struct {
	Body struct {
		Ok      bool   `json:"ok"`
		MatchID string `json:"match_id"`
	}
}

type okResponse struct {
	Body struct {
		Ok bool `json:"ok"`
	}
}

type listMessagesInput struct {
	UserID    string `header:"X-User-Id"`
	RequestID string `path:"id"`
	Cursor    string `query:"cursor"`
	Limit     int    `query:"limit"`
}

type messagePayload struct {
	ID        string `json:"id"`
	SenderID  string `json:"sender_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

type listMessagesResponse struct {
	Body struct {
		Items      []messagePayload `json:"items"`
		NextCursor *string          `json:"next_cursor"`
	}
}

type createMessageInput struct {
	UserID    string `header:"X-User-Id"`
	RequestID string `path:"id"`
	Body      struct {
		Body string `json:"body" required:"true"`
	}
}

type createMessageResponse struct {
	Body messagePayload
}

// --- Route registration ---

func registerConnectionRoutes(api huma.API, cfg config.Config, pool *pgxpool.Pool) {
	repo := repository.NewConnectionRepository(pool)
	svc := service.NewConnectionService(repo, cfg.PreMatchMessageLimit, cfg.MatchRequestMessageMaxLen)
	h := &connectionHandler{svc: svc}

	huma.Post(api, "/match-requests", h.createMatchRequest)
	huma.Get(api, "/match-requests", h.listMatchRequests)
	huma.Get(api, "/match-requests/{id}", h.getMatchRequest)
	huma.Post(api, "/match-requests/{id}/accept", h.acceptMatchRequest)
	huma.Post(api, "/match-requests/{id}/decline", h.declineMatchRequest)
	huma.Post(api, "/match-requests/{id}/cancel", h.cancelMatchRequest)
	huma.Get(api, "/match-requests/{id}/messages", h.listMessages)
	huma.Post(api, "/match-requests/{id}/messages", h.createMessage)
}

// --- Handlers ---

func (h *connectionHandler) createMatchRequest(ctx context.Context, input *createMatchRequestInput) (*createMatchRequestResponse, error) {
	result, err := h.svc.CreateMatchRequest(ctx, input.UserID, input.Body.RecipientID, input.Body.InitialMessage)
	if err != nil {
		return nil, toHumaError(ctx, err)
	}

	resp := &createMatchRequestResponse{}
	resp.Body.ID = result.ID
	resp.Body.RequesterID = result.RequesterID
	resp.Body.RecipientID = result.RecipientID
	resp.Body.Status = result.Status
	resp.Body.InitialMessage = result.InitialMessage
	resp.Body.CreatedAt = result.CreatedAt.Format(time.RFC3339)
	return resp, nil
}

func (h *connectionHandler) listMatchRequests(ctx context.Context, input *listMatchRequestsInput) (*listMatchRequestsResponse, error) {
	var cursor *string
	if input.Cursor != "" {
		cursor = &input.Cursor
	}

	result, err := h.svc.ListMatchRequests(ctx, input.UserID, input.Direction, input.Status, cursor, input.Limit)
	if err != nil {
		return nil, toHumaError(ctx, err)
	}

	items := make([]matchRequestPayload, 0, len(result.Items))
	for _, r := range result.Items {
		items = append(items, toMatchRequestPayload(r))
	}

	resp := &listMatchRequestsResponse{}
	resp.Body.Items = items
	resp.Body.NextCursor = result.NextCursor
	return resp, nil
}

func (h *connectionHandler) getMatchRequest(ctx context.Context, input *getMatchRequestInput) (*getMatchRequestResponse, error) {
	result, err := h.svc.GetMatchRequest(ctx, input.RequestID, input.UserID)
	if err != nil {
		return nil, toHumaError(ctx, err)
	}

	return &getMatchRequestResponse{
		Body: toMatchRequestPayload(result.MatchRequestListItem),
	}, nil
}

func (h *connectionHandler) acceptMatchRequest(ctx context.Context, input *resolveRequestInput) (*acceptResponse, error) {
	matchID, err := h.svc.AcceptMatchRequest(ctx, input.RequestID, input.UserID)
	if err != nil {
		return nil, toHumaError(ctx, err)
	}

	resp := &acceptResponse{}
	resp.Body.Ok = true
	resp.Body.MatchID = matchID
	return resp, nil
}

func (h *connectionHandler) declineMatchRequest(ctx context.Context, input *resolveRequestInput) (*okResponse, error) {
	if err := h.svc.DeclineMatchRequest(ctx, input.RequestID, input.UserID); err != nil {
		return nil, toHumaError(ctx, err)
	}
	resp := &okResponse{}
	resp.Body.Ok = true
	return resp, nil
}

func (h *connectionHandler) cancelMatchRequest(ctx context.Context, input *resolveRequestInput) (*okResponse, error) {
	if err := h.svc.CancelMatchRequest(ctx, input.RequestID, input.UserID); err != nil {
		return nil, toHumaError(ctx, err)
	}
	resp := &okResponse{}
	resp.Body.Ok = true
	return resp, nil
}

func (h *connectionHandler) listMessages(ctx context.Context, input *listMessagesInput) (*listMessagesResponse, error) {
	var cursor *string
	if input.Cursor != "" {
		cursor = &input.Cursor
	}

	result, err := h.svc.ListPreAcceptMessages(ctx, input.RequestID, input.UserID, cursor, input.Limit)
	if err != nil {
		return nil, toHumaError(ctx, err)
	}

	items := make([]messagePayload, 0, len(result.Items))
	for _, m := range result.Items {
		items = append(items, messagePayload{
			ID:        m.ID,
			SenderID:  m.SenderID,
			Body:      m.Body,
			CreatedAt: m.CreatedAt.Format(time.RFC3339),
		})
	}

	resp := &listMessagesResponse{}
	resp.Body.Items = items
	resp.Body.NextCursor = result.NextCursor
	return resp, nil
}

func (h *connectionHandler) createMessage(ctx context.Context, input *createMessageInput) (*createMessageResponse, error) {
	result, err := h.svc.CreatePreAcceptMessage(ctx, input.RequestID, input.UserID, input.Body.Body)
	if err != nil {
		return nil, toHumaError(ctx, err)
	}

	return &createMessageResponse{
		Body: messagePayload{
			ID:        result.ID,
			SenderID:  result.SenderID,
			Body:      result.Body,
			CreatedAt: result.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

// --- Helpers ---

func toMatchRequestPayload(r service.MatchRequestListItem) matchRequestPayload {
	p := matchRequestPayload{
		ID:           r.ID,
		Status:       r.Status,
		MessageCount: r.MessageCount,
		CreatedAt:    r.CreatedAt.Format(time.RFC3339),
		Requester: requestUserPayload{
			UserID:      r.RequesterID,
			Handle:      r.RequesterHandle,
			CountryCode: r.RequesterCountry,
			Age:         r.RequesterAge,
		},
		Recipient: requestUserPayload{
			UserID:      r.RecipientID,
			Handle:      r.RecipientHandle,
			CountryCode: r.RecipientCountry,
			Age:         r.RecipientAge,
		},
	}
	if r.LastMessageAt != nil {
		t := r.LastMessageAt.Format(time.RFC3339)
		p.LastMessageAt = &t
	}
	return p
}
