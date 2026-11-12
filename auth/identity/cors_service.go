package identity

import (
	stdContext "context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/kataras/iris/v14/auth/identity/entity"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/middleware/cors"
	"github.com/kataras/iris/v14/rest/errors"
)

const hellenicDevelopmentPanelOrigin = "https://id.hellenic.dev"

type CORSService struct {
	r          *entity.Registry
	allowLocal bool

	patterns atomic.Value // holds the allowed CORS patterns, a slice of strings.
	allowAny atomic.Bool  // dynamic, if patterns contains at least one * or patterns are empty.
}

func NewCORSService(r *entity.Registry, allowLocal bool) *CORSService {
	s := &CORSService{
		r:          r,
		allowLocal: allowLocal,
	}

	ctx, cancel := stdContext.WithTimeout(stdContext.Background(), 30*time.Second)
	defer cancel()
	if err := s.syncPatterns(ctx); err != nil {
		panic(fmt.Errorf("failed to load and initialize CORS patterns: %w", err))
	}

	return s
}

func (s *CORSService) syncPatterns(ctx stdContext.Context) error {
	// Load patterns from the database.
	patterns, err := s.r.Origins().ListOriginPatterns(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve CORS patterns: %w", err)
	}

	// Allow empty patterns so that any external origin is blocked and so the UI doesn't send back the * always,
	// so it will skip any edits because of MergeAllowOrigins.
	// if len(patterns) == 0 || slices.Contains(patterns, "*") {
	// 	c.allowAny.Store(true)
	// 	c.patterns.Store([]string{"*"})
	// 	return nil // if patterns contains a single "*", then allow all origins.
	// }

	s.allowAny.Store(patternsAllowAny(patterns))
	s.patterns.Store(patterns)
	// s.mu.Lock()
	// defer c.mu.Unlock()
	// s.patterns = patterns

	return nil
}

// Patterns returns the currently stored CORS patterns.
func (s *CORSService) Patterns() []string {
	if patternsValue := s.patterns.Load(); patternsValue != nil {
		if patterns, ok := patternsValue.([]string); ok {
			return patterns
		}
	}

	return nil
}

// AllowOrigin completes the cors.AllowOriginFunc signature.
func (s *CORSService) AllowOrigin(ctx *context.Context, origin string) bool {
	if origin == hellenicDevelopmentPanelOrigin {
		return true // always allow the hellenic development panel origin.
	} else if s.allowLocal && ctx.IsLocal() {
		return true // always allow the local development origin.
		// The cors middleware supports local checking, but we don't do that yet as we need to know the dev port or the environment
		// this is running, maybe a TODO: is to pass the environment (prod or dev) prod on allowing ctx.IsLocal() and prod not.
		// For now this is hardcoded to allow the local development origin.
	}

	if s.allowAny.Load() {
		return true // allow all origins, fast check.
	}

	// c.mu.RLock()
	// defer c.mu.RUnlock()

	for _, pattern := range s.Patterns() {
		if cors.MatchPattern(pattern, origin) {
			return true
		}
	}

	return false
}

// patternsAllowAny reports whether a stored pattern list contains the "*" wildcard.
func patternsAllowAny(patterns []string) bool {
	for _, pattern := range patterns {
		if strings.TrimSpace(pattern) == "*" {
			return true
		}
	}
	return false
}

// validateOriginPatterns accepts the patterns a caller stored.
// "*" is a real pattern: it allows every origin.
func validateOriginPatterns(patterns []string) error {
	for _, pattern := range patterns {
		if strings.TrimSpace(pattern) == "" {
			return fmt.Errorf("%w: origin pattern is empty", entity.ErrInvalidArgument)
		}
	}
	return nil
}

func (s *CORSService) UpsertOriginPatterns(ctx stdContext.Context, patterns ...string) (bool, error) {
	if len(patterns) == 0 {
		return false, nil // no patterns to upsert.
	}
	if err := validateOriginPatterns(patterns); err != nil {
		return false, err
	}

	modified, err := s.r.Origins().UpsertOriginPatterns(ctx, patterns...)
	if modified {
		err = s.syncPatterns(ctx)
	}
	return modified, err
}

func (s *CORSService) DeleteOriginPatterns(ctx stdContext.Context, patterns ...string) (bool, error) {
	if len(patterns) == 0 {
		return false, nil // no patterns to delete.
	}

	modified, err := s.r.Origins().DeleteOriginPatterns(ctx, patterns...)
	if modified {
		err = s.syncPatterns(ctx)
	}
	return modified, err
}

func (s *CORSService) ListOriginPatterns(ctx stdContext.Context) ([]string, error) {
	patterns, err := s.r.Origins().ListOriginPatterns(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve CORS patterns: %w", err)
	}

	return patterns, nil
}

func (s *CORSService) Handler() context.Handler {
	return cors.Strict().
		ExtractOriginFunc(cors.DefaultOriginExtractor).
		ReferrerPolicy(cors.NoReferrerWhenDowngrade).
		HandleErrorFunc(errors.DefaultCORSErrorHandler).
		// SkipLocalRequests(config.Server.Addr == ":8080").
		// AllowHeaders("Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "Accept", "Origin", "Cache-Control", "X-Requested-With", "X-Token").
		// AllowOrigins("https://id.hellenic.dev", "http://localhost:5173"). // this middleware contains the jwks too, I don't know if we must protect it with CORS.
		AllowOriginFunc(s.AllowOrigin).
		Handler()
}
