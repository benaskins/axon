package axon

import (
	"context"
	"log/slog"
	"net/http"
)

type contextKey string

const (
	// UserIDKey is the context key for the authenticated user's ID.
	UserIDKey contextKey = "user_id"
	// UsernameKey is the context key for the authenticated user's username.
	UsernameKey contextKey = "username"
)

type authConfig struct {
	cookieName string
}

// AuthOption configures RequireAuth behavior.
type AuthOption func(*authConfig)

// WithCookieName sets the session cookie name. Defaults to "session".
func WithCookieName(name string) AuthOption {
	return func(c *authConfig) {
		c.cookieName = name
	}
}

// RequireAuth returns middleware that validates session cookies via the AuthClient.
// On success, sets UserIDKey and UsernameKey in the request context.
// On failure, responds with 401 or 503.
func RequireAuth(ac *AuthClient, opts ...AuthOption) func(http.Handler) http.Handler {
	cfg := &authConfig{cookieName: "session"}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cfg.cookieName)
			if err != nil {
				slog.Warn("auth: no session cookie", "path", r.URL.Path)
				WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			session, err := ac.ValidateSession(cookie.Value)
			if err != nil {
				slog.Warn("auth: session validation failed", "path", r.URL.Path, "error", err)
				if err == ErrServiceUnavailable {
					slog.Error("auth service unavailable during request")
					WriteError(w, http.StatusServiceUnavailable, "auth service unavailable")
					return
				}

				WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, session.UserID)
			ctx = context.WithValue(ctx, UsernameKey, session.Username)
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
