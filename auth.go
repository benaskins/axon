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

const sessionCacheTTL = 30 * time.Second

type SessionInfo struct {
	UserID   string
	Username string
}

type cachedSession struct {
	userID    string
	username  string
	expiresAt time.Time
}

type AuthClient struct {
	authServiceURL string
	httpClient     *http.Client
	cache          sync.Map // sessionToken -> cachedSession
	stopSweep      chan struct{}
}

func NewAuthClient(authServiceURL string) *AuthClient {
	tlsConfig, err := LoadClientTLSConfig()
	if err != nil {
		slog.Warn("failed to load client TLS config, using insecure", "error", err)
		tlsConfig = &tls.Config{InsecureSkipVerify: true}
	}

	ac := &AuthClient{
		authServiceURL: authServiceURL,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
		stopSweep: make(chan struct{}),
	}
	go ac.sweepExpired()
	return ac
}

// NewAuthClientPlain creates an AuthClient without mTLS (plain HTTP).
// Use for services that talk to auth over localhost.
func NewAuthClientPlain(authServiceURL string) *AuthClient {
	ac := &AuthClient{
		authServiceURL: authServiceURL,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
		stopSweep: make(chan struct{}),
	}
	go ac.sweepExpired()
	return ac
}

// StartSweep starts the cache cleanup goroutine. Only needed in tests
// where NewAuthClient wasn't used.
func (ac *AuthClient) StartSweep() {
	ac.stopSweep = make(chan struct{})
	go ac.sweepExpired()
}

// StopSweep stops the cache cleanup goroutine.
func (ac *AuthClient) StopSweep() {
	close(ac.stopSweep)
}

func (ac *AuthClient) sweepExpired() {
	ticker := time.NewTicker(sessionCacheTTL)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			ac.cache.Range(func(key, value any) bool {
				if now.After(value.(cachedSession).expiresAt) {
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

	caCert, err := os.ReadFile(caFile)
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
		cached := entry.(cachedSession)
		if time.Now().Before(cached.expiresAt) {
			return &SessionInfo{UserID: cached.userID, Username: cached.username}, nil
		}
		ac.cache.Delete(sessionToken)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ac.authServiceURL+"/api/validate", nil)
	if err != nil {
		return nil, ErrServiceUnavailable
	}

	req.AddCookie(&http.Cookie{Name: "session", Value: sessionToken})

	resp, err := ac.httpClient.Do(req)
	if err != nil {
		slog.Error("auth service unavailable", "error", err)
		return nil, ErrServiceUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrUnauthorized
	}

	var result struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ErrServiceUnavailable
	}

	// Cache successful validation
	ac.cache.Store(sessionToken, cachedSession{
		userID:    result.UserID,
		username:  result.Username,
		expiresAt: time.Now().Add(sessionCacheTTL),
	})

	return &SessionInfo{UserID: result.UserID, Username: result.Username}, nil
}
