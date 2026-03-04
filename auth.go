package axon

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

// SessionValidator validates a session token and returns session info.
type SessionValidator interface {
	ValidateSession(token string) (*SessionInfo, error)
}

// SessionInfo holds claims from a validated session.
type SessionInfo struct {
	Claims map[string]any
}

// UserID returns the "user_id" claim as a string, or empty if absent.
func (s *SessionInfo) UserID() string {
	v, _ := s.Claims["user_id"].(string)
	return v
}

// Username returns the "username" claim as a string, or empty if absent.
func (s *SessionInfo) Username() string {
	v, _ := s.Claims["username"].(string)
	return v
}

// Claim returns a single claim value by key, or nil if absent.
func (s *SessionInfo) Claim(key string) any {
	return s.Claims[key]
}

type cachedSession struct {
	claims    map[string]any
	expiresAt time.Time
}

type authClientConfig struct {
	endpointPath string
	tokenSender  func(*http.Request, string)
	decodeFunc   func(*http.Response) (map[string]any, error)
	cacheTTL     time.Duration
}

// AuthClientOption configures AuthClient behavior.
type AuthClientOption func(*authClientConfig)

// WithEndpointPath sets the validation endpoint path. Defaults to "/api/validate".
func WithEndpointPath(path string) AuthClientOption {
	return func(c *authClientConfig) {
		c.endpointPath = path
	}
}

// WithTokenSender sets how the session token is attached to the validation
// request. Defaults to sending as a cookie named "session".
func WithTokenSender(fn func(*http.Request, string)) AuthClientOption {
	return func(c *authClientConfig) {
		c.tokenSender = fn
	}
}

// WithDecodeFunc sets how the validation response is decoded into claims.
// Defaults to JSON decoding the response body.
func WithDecodeFunc(fn func(*http.Response) (map[string]any, error)) AuthClientOption {
	return func(c *authClientConfig) {
		c.decodeFunc = fn
	}
}

// WithCacheTTL sets the duration cached sessions remain valid.
// Defaults to 30 seconds.
func WithCacheTTL(d time.Duration) AuthClientOption {
	return func(c *authClientConfig) {
		c.cacheTTL = d
	}
}

// AuthClient validates session tokens against a remote auth service.
type AuthClient struct {
	authServiceURL string
	httpClient     *http.Client
	cache          sync.Map // sessionToken -> cachedSession
	config         authClientConfig
	closeOnce      sync.Once
	stopSweep      chan struct{}
}

// NewAuthClient creates an AuthClient with mTLS from environment variables.
// Returns an error if client TLS configuration fails.
func NewAuthClient(authServiceURL string, opts ...AuthClientOption) (*AuthClient, error) {
	tlsConfig, err := LoadClientTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("load client TLS config: %w", err)
	}

	ac := &AuthClient{
		authServiceURL: authServiceURL,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
		config:    defaultAuthClientConfig(),
		stopSweep: make(chan struct{}),
	}
	for _, opt := range opts {
		opt(&ac.config)
	}
	go ac.sweepExpired()
	return ac, nil
}

// NewAuthClientPlain creates an AuthClient without mTLS (plain HTTP).
// Use for services that talk to auth over localhost.
func NewAuthClientPlain(authServiceURL string, opts ...AuthClientOption) *AuthClient {
	ac := &AuthClient{
		authServiceURL: authServiceURL,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
		config:    defaultAuthClientConfig(),
		stopSweep: make(chan struct{}),
	}
	for _, opt := range opts {
		opt(&ac.config)
	}
	go ac.sweepExpired()
	return ac
}

func defaultAuthClientConfig() authClientConfig {
	return authClientConfig{
		endpointPath: "/api/validate",
		tokenSender: func(req *http.Request, token string) {
			req.AddCookie(&http.Cookie{Name: "session", Value: token})
		},
		decodeFunc: func(resp *http.Response) (map[string]any, error) {
			var claims map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
				return nil, err
			}
			return claims, nil
		},
		cacheTTL: 30 * time.Second,
	}
}

// Close stops the cache cleanup goroutine. Safe to call multiple times.
func (ac *AuthClient) Close() {
	ac.closeOnce.Do(func() {
		close(ac.stopSweep)
	})
}

func (ac *AuthClient) sweepExpired() {
	ticker := time.NewTicker(ac.config.cacheTTL)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			ac.cache.Range(func(key, value any) bool {
				cached, ok := value.(cachedSession)
				if !ok {
					ac.cache.Delete(key)
					return true
				}
				if now.After(cached.expiresAt) {
					ac.cache.Delete(key)
				}
				return true
			})
		case <-ac.stopSweep:
			return
		}
	}
}

// LoadClientTLSConfig loads mTLS client certificates from environment variables
// CLIENT_CERT, CLIENT_KEY, and CA_CERT.
func LoadClientTLSConfig() (*tls.Config, error) {
	certFile := os.Getenv("CLIENT_CERT")
	keyFile := os.Getenv("CLIENT_KEY")
	caFile := os.Getenv("CA_CERT")

	if certFile == "" || keyFile == "" || caFile == "" {
		return nil, fmt.Errorf("client cert environment variables not set")
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load client cert: %w", err)
	}

	caCert, err := os.ReadFile(caFile) // #nosec G304 G703 -- path from operator-controlled env vars, not user input
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA cert")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func (ac *AuthClient) ValidateSession(sessionToken string) (*SessionInfo, error) {
	// Check cache first
	if entry, ok := ac.cache.Load(sessionToken); ok {
		cached, valid := entry.(cachedSession)
		if !valid {
			ac.cache.Delete(sessionToken)
		} else if time.Now().Before(cached.expiresAt) {
			return &SessionInfo{Claims: cached.claims}, nil
		}
		ac.cache.Delete(sessionToken)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ac.authServiceURL+ac.config.endpointPath, nil)
	if err != nil {
		return nil, ErrServiceUnavailable
	}

	ac.config.tokenSender(req, sessionToken)

	resp, err := ac.httpClient.Do(req) // #nosec G704 -- URL from service config, not user input
	if err != nil {
		slog.Error("auth service unavailable", "error", err)
		return nil, ErrServiceUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrUnauthorized
	}

	claims, err := ac.config.decodeFunc(resp)
	if err != nil {
		return nil, ErrServiceUnavailable
	}

	// Cache successful validation
	ac.cache.Store(sessionToken, cachedSession{
		claims:    claims,
		expiresAt: time.Now().Add(ac.config.cacheTTL),
	})

	return &SessionInfo{Claims: claims}, nil
}
