package config

import (
	"testing"
	"time"

	rate "github.com/wallstreetcn/rate/redis"
)

func BenchmarkLimitReached(b *testing.B) {
	limiter := NewLimiter(1, time.Second, &rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	})
	key := "127.0.0.1|/"

	for i := 0; i < b.N; i++ {
		limiter.LimitReached(key, nil)
	}
}
