package tollbooth

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func BenchmarkLimitByKeys(b *testing.B) {
	limiter := NewLimiter(1, time.Second) // Only 1 request per second is allowed.

	for i := 0; i < b.N; i++ {
		LimitByKeys(limiter, []string{"127.0.0.1", "/"})
	}
}

func BenchmarkBuildKeys(b *testing.B) {
	limiter := NewLimiter(1, time.Second)
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
