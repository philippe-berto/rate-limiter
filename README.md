## Rate Limiter by IP and API_KEY

## How to Config - Database

You need instantiate a redis client according do the `database/redis.go` or another database that implements the `Database interface` you can see in the main file.

To get an instance of `database/redis.go` you need set these environmant variables:

```go
RedisConfig struct {
  Address        string `env:"REDIS_ADDRESS" envDefault:"localhost:6379"`
  Password       string `env:"REDIS_PASSWORD" envDefault:""`
  DB             int    `env:"REDIS_DB" envDefault:"0"`
  IPKeyPrefix    string `env:"REDIS_IP_KEY_PREFIX" envDefault:"/rl/ip/"`
  TokenKeyPrefix string `env:"REDIS_TOKEN_KEY_PREFIX" envDefault:"/rl/token/"`
}
```

## How to Config - Rate Limiter

Onde you have an db instance, you can instantiate the rate limiter. You also need set the these env vars:

```go
RateLimiterConfig struct {
  MaxRequestsPerIP   int `env:"MAX_REQUESTS_PER_IP" envDefault:"2"`
  TimePerIP          int `env:"TIME_PER_IP" envDefault:"1"` // in seconds
  MaxRequestPerToken int `env:"MAX_REQUESTS_PER_TOKEN" envDefault:"3"`
  TimePerToken       int `env:"TIME_PER_TOKEN" envDefault:"1"` // in seconds
}

// In your code import
"github.com/philippe-berto/rate-limiter/database/redis"

//intantiate DB
db := redis.New(ctx, cfg)
```

**Important**

Token limiter refers to requests that has `API_KEY` on headers. In this case, the rate limiter consider the token limit and not the IP limit.

## To use the RL Middleware

**Instantiate the Rate Limiter**

```go
// import
ratelimiter "github.com/philippe-berto/rate-limiter"

//instantiate
rl := ratelimiter.New(ctx, cfg.RateLimiterConfig, db)

// Apply the Middleware
router.Use(rl.Middleware)

```
