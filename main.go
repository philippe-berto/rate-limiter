package ratelimiter

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type (
	RateLimiter struct {
		IPTotalRequests    int
		IPTimeRemaining    int
		TokenTotalRequests int
		TokenTimeRemaining int
		db                 Database
	}

	RateLimiterConfig struct {
		MaxRequestsPerIP   int `env:"MAX_REQUESTS_PER_IP" envDefault:"2"`
		TimePerIP          int `env:"TIME_PER_IP" envDefault:"1"` // in seconds
		MaxRequestPerToken int `env:"MAX_REQUESTS_PER_TOKEN" envDefault:"3"`
		TimePerToken       int `env:"TIME_PER_TOKEN" envDefault:"1"` // in seconds
	}

	Database interface {
		StoreIP(ip string, expireSec int) (int, error)
		StoreToken(token string, expireSec int) (int, error)
	}
)

func New(ctx context.Context, cfg RateLimiterConfig, db Database) *RateLimiter {
	return &RateLimiter{
		IPTotalRequests:    cfg.MaxRequestsPerIP,
		IPTimeRemaining:    cfg.TimePerIP,
		TokenTotalRequests: cfg.MaxRequestPerToken,
		TokenTimeRemaining: cfg.TimePerToken,
		db:                 db,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ip := strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]
		token := r.Header.Get("api_key")

		fmt.Println("IP:", ip)
		fmt.Println("Token:", token)

		if token != "" {
			count, err := rl.db.StoreToken(token, rl.TokenTimeRemaining)
			if err != nil {
				http.Error(w, "error on redis database", http.StatusInternalServerError)
				return
			}

			if count > rl.TokenTotalRequests {
				http.Error(w, "you have reached the maximum number of requests or actions allowed within a certain time frame", http.StatusTooManyRequests)
				return
			}
		} else if ip != "" {
			count, err := rl.db.StoreIP(ip, rl.IPTimeRemaining)
			if err != nil {
				http.Error(w, "error on redis database", http.StatusInternalServerError)
				return
			}

			if count > rl.IPTotalRequests {
				http.Error(w, "you have reached the maximum number of requests or actions allowed within a certain time frame", http.StatusTooManyRequests)
				return
			}
		} else {
			http.Error(w, "No valid token or IP found", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
