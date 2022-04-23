package tollbooth

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	rate "github.com/aw16com/rate/redis"
)

func BenchmarkLimitByKeys(b *testing.B) {
	limiter := NewLimiter(1, time.Second, &rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	}) // Only 1 request per second is allowed.

	for i := 0; i < b.N; i++ {
		LimitByKeys(limiter, []string{"127.0.0.1", "/"}, nil)
	}
}

func BenchmarkBuildKeys(b *testing.B) {
	limiter := NewLimiter(1, time.Second, &rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	})
	limiter.IPLookups = []string{"X-Real-IP", "RemoteAddr", "X-Forwarded-For"}
	limiter.Headers = make([]string, 0)
	limiter.Headers = append(limiter.Headers, "X-Real-IP")

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		fmt.Printf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Real-IP", "193.22.33.3")
	for i := 0; i < b.N; i++ {
		sliceKeys := BuildKeys(limiter, request)
		if len(sliceKeys) == 0 {
			fmt.Print("Length of sliceKeys should never be empty.")
		}
	}
}

func BenchmarkBuildKeysWithLongKey(b *testing.B) {
	limiter := NewLimiter(1, time.Second, &rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	})
	limiter.IPLookups = []string{"X-Real-IP", "X-Forwarded-For", "RemoteAddr"}
	limiter.Headers = []string{"X-Auth-Token"}

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		fmt.Printf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Auth-Token", "VI2YiplZDAAdit1gwgctj//avY52fFGsKVTC7nh8ewa2PMtw0eveUxiFtPHemTX8wG++z38Mwn5rXSwahn5gK1bEJOvQ8VUX2O2w4XA4ljA0xikiAJbcO75YCyHxuM7pV4F/Kz9TnritmuQSo8A0qB85Yq9MNamlXxuyNtRfP/neZwXjgrEFctDV6cSVCl71")
	request.Header.Set("X-Forwarded-For", "193.22.33.3")
	for i := 0; i < b.N; i++ {
		sliceKeys := BuildKeys(limiter, request)
		if len(sliceKeys) == 0 {
			fmt.Print("Length of sliceKeys should never be empty.")
		}
	}
}
