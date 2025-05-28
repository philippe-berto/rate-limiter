package test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	rl "github.com/philippe-berto/rate-limiter"
	"github.com/stretchr/testify/assert"
)

// MockDatabase implements the Database interface for testing
type MockDatabase struct {
	ipCounts    map[string]int
	tokenCounts map[string]int
	ipErr       error
	tokenErr    error
}

func (m *MockDatabase) StoreIP(ip string, expireSec int) (int, error) {
	if m.ipErr != nil {
		return 0, m.ipErr
	}
	m.ipCounts[ip]++
	return m.ipCounts[ip], nil
}

func (m *MockDatabase) StoreToken(token string, expireSec int) (int, error) {
	if m.tokenErr != nil {
		return 0, m.tokenErr
	}
	m.tokenCounts[token]++
	return m.tokenCounts[token], nil
}

func newMockDB() *MockDatabase {
	return &MockDatabase{
		ipCounts:    make(map[string]int),
		tokenCounts: make(map[string]int),
	}
}

func TestRateLimiter_TokenLimit(t *testing.T) {
	db := newMockDB()
	cfg := rl.RateLimiterConfig{
		MaxRequestsPerIP:   5,
		TimePerIP:          60,
		MaxRequestPerToken: 2,
		TimePerToken:       60,
	}
	rl := rl.New(context.Background(), cfg, db)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("api_key", "token123")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

func TestRateLimiter_IPLimit(t *testing.T) {
	db := newMockDB()
	cfg := rl.RateLimiterConfig{
		MaxRequestsPerIP:   2,
		TimePerIP:          60,
		MaxRequestPerToken: 5,
		TimePerToken:       60,
	}
	rl := rl.New(context.Background(), cfg, db)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

func TestRateLimiter_NoTokenOrIP(t *testing.T) {
	db := newMockDB()
	cfg := rl.RateLimiterConfig{
		MaxRequestsPerIP:   2,
		TimePerIP:          60,
		MaxRequestPerToken: 2,
		TimePerToken:       60,
	}
	rl := rl.New(context.Background(), cfg, db)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRateLimiter_DatabaseErrorOnToken(t *testing.T) {
	db := newMockDB()
	db.tokenErr = errors.New("db error")
	cfg := rl.RateLimiterConfig{
		MaxRequestsPerIP:   2,
		TimePerIP:          60,
		MaxRequestPerToken: 2,
		TimePerToken:       60,
	}
	rl := rl.New(context.Background(), cfg, db)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("api_key", "token123")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestRateLimiter_DatabaseErrorOnIP(t *testing.T) {
	db := newMockDB()
	db.ipErr = errors.New("db error")
	cfg := rl.RateLimiterConfig{
		MaxRequestsPerIP:   2,
		TimePerIP:          60,
		MaxRequestPerToken: 2,
		TimePerToken:       60,
	}
	rl := rl.New(context.Background(), cfg, db)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestRateLimiter_IPAndToken_PrioritizeToken(t *testing.T) {
	db := newMockDB()
	cfg := rl.RateLimiterConfig{
		MaxRequestsPerIP:   2,
		TimePerIP:          60,
		MaxRequestPerToken: 1,
		TimePerToken:       60,
	}
	rl := rl.New(context.Background(), cfg, db)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.Header.Set("api_key", "token123")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rr.Code)
	}
}
