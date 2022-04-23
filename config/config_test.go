package config

import (
	"testing"
	"time"

	rate "github.com/aw16com/rate/redis"
)

func TestConstructor(t *testing.T) {
	limiter := NewLimiter(1, time.Second, &rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	})
	if limiter.Max != 1 {
		t.Errorf("Max field is incorrect. Value: %v", limiter.Max)
	}
	if limiter.TTL != time.Second {
		t.Errorf("TTL field is incorrect. Value: %v", limiter.TTL)
	}
	if limiter.Message != "You have reached maximum request limit." {
		t.Errorf("Message field is incorrect. Value: %v", limiter.Message)
	}
	if limiter.StatusCode != 429 {
		t.Errorf("StatusCode field is incorrect. Value: %v", limiter.StatusCode)
	}
}

func TestLimitReached(t *testing.T) {
	limiter := NewLimiter(1, time.Second, &rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	})
	key := "TestLimitReached"

	if limiter.LimitReached(key, nil) == true {
		t.Error("First time count should not reached the limit.")
	}

	if limiter.LimitReached(key, nil) == false {
		t.Error("Second time count should return true because it exceeds 1 request per second.")
	}

	<-time.After(1 * time.Second)
	if limiter.LimitReached(key, nil) == true {
		t.Error("Third time count should not reached the limit because the 1 second window has passed.")
	}
}

func TestMuchHigherMaxRequests(t *testing.T) {
	numRequests := 500
	limiter := NewLimiter(int64(numRequests), time.Second/time.Duration(numRequests), &rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	})
	key := "TestMuchHigherMaxRequests"

	for i := 0; i < numRequests; i++ {
		if limiter.LimitReached(key, nil) == true {
			t.Errorf("N(%v) limit should not be reached.", i)
		}
	}

	if limiter.LimitReached(key, nil) == false {
		t.Errorf("N(%v) limit should be reached because it exceeds %v request per second.", numRequests+2, numRequests)
	}

}
