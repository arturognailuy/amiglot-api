package service

import (
	"context"
	"strings"
	"time"

	"github.com/gnailuy/amiglot-api/internal/repository"
)

// baseLang returns the base language from a BCP-47 code (e.g. "zh-Hans" → "zh").
func baseLang(code string) string {
	if i := strings.IndexByte(code, '-'); i >= 0 {
		return code[:i]
	}
	return code
}

const (
	defaultMinOverlapMinutes = 60
	defaultLimit             = 20
	maxLimit                 = 50
)

// DiscoveryService handles match discovery business logic.
type DiscoveryService struct {
	repo              *repository.DiscoveryRepository
	minOverlapMinutes int
}

// NewDiscoveryService creates a new DiscoveryService.
func NewDiscoveryService(repo *repository.DiscoveryRepository, minOverlapMinutes int) *DiscoveryService {
	if minOverlapMinutes <= 0 {
		minOverlapMinutes = defaultMinOverlapMinutes
	}
	return &DiscoveryService{repo: repo, minOverlapMinutes: minOverlapMinutes}
}

// MatchItem represents a single match in the discovery response.
type MatchItem struct {
	UserID              string
	Handle              string
	CountryCode         *string
	Age                 *int
	MutualTeach         []MatchLanguage
	MutualLearn         []MatchLanguage
	BridgeLanguages     []BridgeLanguage
	AvailabilityOverlap []OverlapSlot
	TotalOverlapMinutes int
}

// MatchLanguage represents a language in mutual teach/learn context.
type MatchLanguage struct {
	LanguageCode string
	Level        int16
	IsNative     bool
	LearnerLevel int16
}

// BridgeLanguage represents a shared bridge language.
type BridgeLanguage struct {
	LanguageCode string
	Level        int16
}

// OverlapSlot represents a single availability overlap time slot.
type OverlapSlot struct {
	Weekday        int16
	StartUTC       string
	EndUTC         string
	OverlapMinutes int
}

// DiscoverResult is the paginated result of a discovery query.
type DiscoverResult struct {
	Items      []MatchItem
	NextCursor *string
}

// Discover returns paginated match results for the given user.
func (s *DiscoveryService) Discover(ctx context.Context, userID string, cursor *string, limit int) (*DiscoverResult, error) {
	if s.repo == nil || s.repo.Pool() == nil {
		return nil, &Error{Status: 503, Key: "errors.database_unavailable"}
	}

	// Validate profile
	discoverable, err := s.repo.IsDiscoverable(ctx, userID)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.failed_load_profile", Err: err}
	}
	if !discoverable {
		return nil, &Error{Status: 403, Key: "errors.profile_incomplete"}
	}

	// Validate target languages
	hasTargets, err := s.repo.HasTargetLanguages(ctx, userID)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.failed_load_languages", Err: err}
	}
	if !hasTargets {
		return nil, &Error{Status: 422, Key: "errors.no_target_languages"}
	}

	// Normalize limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	// Fetch one extra to determine if there's a next page
	matches, err := s.repo.DiscoverMatches(ctx, userID, s.minOverlapMinutes, limit+1, cursor)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}

	var nextCursor *string
	if len(matches) > limit {
		matches = matches[:limit]
		last := matches[len(matches)-1].UserID
		nextCursor = &last
	}

	// Get the requesting user's languages for intersection
	myLangs, err := s.repo.GetUserLanguages(ctx, userID)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.failed_load_languages", Err: err}
	}

	myTeach := make(map[string][]repository.LanguageRow)  // I can teach (level >= 4), keyed by base lang
	myTarget := make(map[string][]repository.LanguageRow) // I want to learn (is_target), keyed by base lang
	myBridge := make(map[string][]repository.LanguageRow) // I can bridge (level >= 3), keyed by base lang
	for _, l := range myLangs {
		base := baseLang(l.LanguageCode)
		if l.Level >= 4 {
			myTeach[base] = append(myTeach[base], l)
		}
		if l.IsTarget {
			myTarget[base] = append(myTarget[base], l)
		}
		if l.Level >= 3 {
			myBridge[base] = append(myBridge[base], l)
		}
	}

	items := make([]MatchItem, 0, len(matches))
	for _, m := range matches {
		// Get candidate languages
		candLangs, err := s.repo.GetUserLanguages(ctx, m.UserID)
		if err != nil {
			return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
		}

		// Get overlap details
		overlaps, err := s.repo.GetOverlapDetails(ctx, userID, m.UserID)
		if err != nil {
			return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
		}

		item := MatchItem{
			UserID:              m.UserID,
			Handle:              m.Handle,
			CountryCode:         m.CountryCode,
			Age:                 computeAge(m.BirthYear, m.BirthMonth),
			TotalOverlapMinutes: m.TotalOverlapMinutes,
		}

		// Compute mutual_teach (candidate teaches what I want to learn — base language match)
		for _, cl := range candLangs {
			base := baseLang(cl.LanguageCode)
			if rows, ok := myTarget[base]; ok && cl.Level >= 4 {
				// Find the best (highest-level) learner entry from my target languages
				var bestLearnerLevel int16
				for _, r := range rows {
					if r.Level > bestLearnerLevel {
						bestLearnerLevel = r.Level
					}
				}
				item.MutualTeach = append(item.MutualTeach, MatchLanguage{
					LanguageCode: cl.LanguageCode,
					Level:        cl.Level,
					IsNative:     cl.IsNative,
					LearnerLevel: bestLearnerLevel,
				})
			}
		}

		// Compute mutual_learn (I teach what candidate wants to learn — base language match)
		for _, cl := range candLangs {
			if cl.IsTarget {
				base := baseLang(cl.LanguageCode)
				if mls, ok := myTeach[base]; ok {
					// Use the highest-level entry from my teach languages
					best := mls[0]
					for _, ml := range mls[1:] {
						if ml.Level > best.Level {
							best = ml
						}
					}
					item.MutualLearn = append(item.MutualLearn, MatchLanguage{
						LanguageCode: best.LanguageCode,
						Level:        best.Level,
						IsNative:     best.IsNative,
						LearnerLevel: cl.Level,
					})
				}
			}
		}

		// Compute bridge languages (base language match)
		for _, cl := range candLangs {
			if cl.Level >= 3 {
				if _, ok := myBridge[baseLang(cl.LanguageCode)]; ok {
					item.BridgeLanguages = append(item.BridgeLanguages, BridgeLanguage{
						LanguageCode: cl.LanguageCode,
						Level:        cl.Level,
					})
				}
			}
		}

		// Overlap details
		for _, o := range overlaps {
			item.AvailabilityOverlap = append(item.AvailabilityOverlap, OverlapSlot{
				Weekday:        o.Weekday,
				StartUTC:       o.StartUTC,
				EndUTC:         o.EndUTC,
				OverlapMinutes: o.OverlapMinutes,
			})
		}

		items = append(items, item)
	}

	return &DiscoverResult{Items: items, NextCursor: nextCursor}, nil
}

func computeAge(birthYear *int, birthMonth *int16) *int {
	if birthYear == nil {
		return nil
	}
	now := time.Now()
	age := now.Year() - *birthYear
	if birthMonth != nil && int(now.Month()) < int(*birthMonth) {
		age--
	}
	if age < 0 {
		age = 0
	}
	return &age
}
