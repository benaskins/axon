package axon

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
)

type contextKey string

const (
	// UserIDKey is the context key for the authenticated user's ID.
	UserIDKey contextKey = "user_id"
	// UsernameKey is the context key for the authenticated user's username.
	UsernameKey contextKey = "username"
	// SessionInfoKey is the context key for the full SessionInfo.
	SessionInfoKey contextKey = "session_info"
)

type authConfig struct {
	tokenExtractor func(*http.Request) (string, error)
}

// AuthOption configures RequireAuth behavior.
type AuthOption func(*authConfig)

// WithCookieName sets the session cookie name. Defaults to "session".
func WithCookieName(name string) AuthOption {
	return func(c *authConfig) {
		c.tokenExtractor = cookieExtractor(name)
	}
}

// WithTokenExtractor sets a custom function to extract the session token
// from the request. Overrides the default cookie-based extraction.
func WithTokenExtractor(fn func(*http.Request) (string, error)) AuthOption {
	return func(c *authConfig) {
		c.tokenExtractor = fn
	}
}

func cookieExtractor(name string) func(*http.Request) (string, error) {
	return func(r *http.Request) (string, error) {
		cookie, err := r.Cookie(name)
		if err != nil {
			return "", err
		}
		return cookie.Value, nil
	}
}

// RequireAuth returns middleware that validates session tokens via a SessionValidator.
// On success, sets UserIDKey, UsernameKey, and SessionInfoKey in the request context.
// On failure, responds with 401 or 503.
func RequireAuth(sv SessionValidator, opts ...AuthOption) func(http.Handler) http.Handler {
	cfg := &authConfig{
		tokenExtractor: cookieExtractor("session"),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := cfg.tokenExtractor(r)
			if err != nil {
				slog.Warn("auth: no session token", "path", r.URL.Path)
				WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			session, err := sv.ValidateSession(token)
			if err != nil {
				slog.Warn("auth: session validation failed", "path", r.URL.Path, "error", err)
				if errors.Is(err, ErrServiceUnavailable) {
					slog.Error("auth service unavailable during request")
					WriteError(w, http.StatusServiceUnavailable, "auth service unavailable")
					return
				}

				WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			ctx := context.WithValue(r.Context(), SessionInfoKey, session)
			ctx = context.WithValue(ctx, UserIDKey, session.UserID())
			ctx = context.WithValue(ctx, UsernameKey, session.Username())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserID extracts the authenticated user ID from the request context.
// Returns empty string if not present.
func UserID(ctx context.Context) string {
	v, _ := ctx.Value(UserIDKey).(string)
	return v
}

// Username extracts the authenticated username from the request context.
// Returns empty string if not present.
func Username(ctx context.Context) string {
	v, _ := ctx.Value(UsernameKey).(string)
	return v
}

// Session extracts the full SessionInfo from the request context.
// Returns nil if not present.
func Session(ctx context.Context) *SessionInfo {
	v, _ := ctx.Value(SessionInfoKey).(*SessionInfo)
	return v
}
