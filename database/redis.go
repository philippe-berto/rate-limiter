package database

import (
	"context"

	rl "github.com/philippe-berto/rate-limiter"
	"github.com/redis/go-redis/v9"
)

var _ rl.Database = (*RedisClient)(nil)

type (
	RedisClient struct {
		ctx            context.Context
		client         *redis.Client
		IPKeyPrefix    string
		TokenKeyPrefix string
	}
	RedisConfig struct {
		Address        string `env:"REDIS_ADDRESS" envDefault:"localhost:6379"`
		Password       string `env:"REDIS_PASSWORD" envDefault:""`
		DB             int    `env:"REDIS_DB" envDefault:"0"`
		IPKeyPrefix    string `env:"REDIS_IP_KEY_PREFIX" envDefault:"/rl/ip/"`
		TokenKeyPrefix string `env:"REDIS_TOKEN_KEY_PREFIX" envDefault:"/rl/token/"`
	}
)

func New(ctx context.Context, cfg RedisConfig) *RedisClient {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return &RedisClient{
		ctx:            ctx,
		client:         redisClient,
		IPKeyPrefix:    cfg.IPKeyPrefix,
		TokenKeyPrefix: cfg.TokenKeyPrefix,
	}
}

func (r *RedisClient) StoreIP(ip string, expireSec int) (int, error) {
	expire := int64(expireSec)
	script := redis.NewScript(`
        local current = redis.call("INCR", KEYS[1])
        if current == 1 then
            redis.call("EXPIRE", KEYS[1], ARGV[1])
        end
        return current
    `)
	result, err := script.Run(r.ctx, r.client, []string{r.IPKeyPrefix + ip}, expire).Result()
	if err != nil {
		return 0, err
	}
	count, ok := result.(int64)
	if !ok {
		return 0, nil
	}
	return int(count), nil
}

func (r *RedisClient) StoreToken(token string, expireSec int) (int, error) {
	expire := int64(expireSec)
	script := redis.NewScript(`
        local current = redis.call("INCR", KEYS[1])
        if current == 1 then
            redis.call("EXPIRE", KEYS[1], ARGV[1])
        end
        return current
    `)
	result, err := script.Run(r.ctx, r.client, []string{r.TokenKeyPrefix + token}, expire).Result()
	if err != nil {
		return 0, err
	}

	count, ok := result.(int64)
	if !ok {
		return 0, nil
	}
	return int(count), nil
}
