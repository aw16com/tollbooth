package tollbooth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	rate "github.com/aw16com/rate/redis"
	"github.com/aw16com/tollbooth/config"
)

func setup() {
	rate.SetRedis(&rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	})

}

func teardown() {
}

func TestMain(m *testing.M) {
	setup()
	retCode := m.Run()
	teardown()
	os.Exit(retCode)
}

func TestLimitByKeys(t *testing.T) {
	rate.Client().FlushAll()

	limiter := NewLimiter(1, time.Second, &rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	}) // Only 1 request per second is allowed.

	httperror := LimitByKeys(limiter, []string{"127.0.0.1", "/"}, nil)
	if httperror != nil {
		t.Errorf("First time count should not return error. Error: %v", httperror.Error())
	}

	httperror = LimitByKeys(limiter, []string{"127.0.0.1", "/"}, nil)
	if httperror == nil {
		t.Errorf("Second time count should return error because it exceeds 1 request per second.")
	}

	<-time.After(1 * time.Second)
	httperror = LimitByKeys(limiter, []string{"127.0.0.1", "/"}, nil)
	if httperror != nil {
		t.Errorf("Third time count should not return error because the 1 second window has passed.")
	}
}

func TestDefaultBuildKeys(t *testing.T) {
	rate.Client().FlushAll()

	limiter := NewLimiter(1, time.Second, &rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	})
	limiter.IPLookups = []string{"X-Forwarded-For", "X-Real-IP", "RemoteAddr"}

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		t.Errorf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")

	sliceKeys := BuildKeys(limiter, request)
	if len(sliceKeys) == 0 {
		t.Error("Length of sliceKeys should never be empty.")
	}

	for _, keys := range sliceKeys {
		for i, keyChunk := range keys {
			if i == 0 && keyChunk != request.Header.Get("X-Real-IP") {
				t.Errorf("The first chunk should be remote IP. KeyChunk: %v", keyChunk)
			}
			if i == 1 && keyChunk != request.URL.Path {
				t.Errorf("The second chunk should be request path. KeyChunk: %v", keyChunk)
			}
		}
	}
}

func TestBasicAuthBuildKeys(t *testing.T) {
	rate.Client().FlushAll()

	limiter := NewLimiter(1, time.Second, &rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	})
	limiter.BasicAuthUsers = []string{"bro"}

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		t.Errorf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")

	request.SetBasicAuth("bro", "tato")

	for _, keys := range BuildKeys(limiter, request) {
		if len(keys) != 3 {
			t.Error("Keys should be made of 3 parts.")
		}
		for i, keyChunk := range keys {
			if i == 0 && keyChunk != request.Header.Get("X-Real-IP") {
				t.Errorf("The (%v) chunk should be remote IP. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 1 && keyChunk != request.URL.Path {
				t.Errorf("The (%v) chunk should be request path. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 2 && keyChunk != "bro" {
				t.Errorf("The (%v) chunk should be request username. KeyChunk: %v", i+1, keyChunk)
			}
		}
	}
}

func TestCustomHeadersBuildKeys(t *testing.T) {
	rate.Client().FlushAll()

	limiter := NewLimiter(1, time.Second, &rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	})
	limiter.Headers = []string{"X-Auth-Token"}

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		t.Errorf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")
	request.Header.Set("X-Auth-Token", "totally-top-secret")

	for _, keys := range BuildKeys(limiter, request) {
		if len(keys) != 4 {
			t.Errorf("Keys should be made of 4 parts. Keys: %v", keys)
		}
		for i, keyChunk := range keys {
			if i == 0 && keyChunk != request.Header.Get("X-Real-IP") {
				t.Errorf("The (%v) chunk should be remote IP. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 1 && keyChunk != request.URL.Path {
				t.Errorf("The (%v) chunk should be request path. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 2 && keyChunk != "X-Auth-Token" {
				t.Errorf("The (%v) chunk should be request header. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 3 && keyChunk != "totally-top-secret" {
				t.Errorf("The (%v) chunk should be request path. KeyChunk: %v", i+1, keyChunk)
			}
		}
	}
}

func TestRequestMethodBuildKeys(t *testing.T) {
	rate.Client().FlushAll()

	limiter := NewLimiter(1, time.Second, &rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	})
	limiter.Methods = []string{"GET"}

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		t.Errorf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")

	for _, keys := range BuildKeys(limiter, request) {
		if len(keys) != 3 {
			t.Errorf("Keys should be made of 3 parts. Keys: %v", keys)
		}
		for i, keyChunk := range keys {
			if i == 0 && keyChunk != request.Header.Get("X-Real-IP") {
				t.Errorf("The (%v) chunk should be remote IP. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 1 && keyChunk != request.URL.Path {
				t.Errorf("The (%v) chunk should be request path. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 2 && keyChunk != "GET" {
				t.Errorf("The (%v) chunk should be request method. KeyChunk: %v", i+1, keyChunk)
			}
		}
	}
}

func TestRequestMethodAndCustomHeadersBuildKeys(t *testing.T) {
	rate.Client().FlushAll()

	limiter := NewLimiter(1, time.Second, &rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	})
	limiter.Methods = []string{"GET"}
	limiter.Headers = []string{"X-Auth-Token"}

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		t.Errorf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")
	request.Header.Set("X-Auth-Token", "totally-top-secret")

	for _, keys := range BuildKeys(limiter, request) {
		if len(keys) != 5 {
			t.Errorf("Keys should be made of 4 parts. Keys: %v", keys)
		}
		for i, keyChunk := range keys {
			if i == 0 && keyChunk != request.Header.Get("X-Real-IP") {
				t.Errorf("The (%v) chunk should be remote IP. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 1 && keyChunk != request.URL.Path {
				t.Errorf("The (%v) chunk should be request path. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 2 && keyChunk != "GET" {
				t.Errorf("The (%v) chunk should be request method. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 3 && keyChunk != "X-Auth-Token" {
				t.Errorf("The (%v) chunk should be request header. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 4 && keyChunk != "totally-top-secret" {
				t.Errorf("The (%v) chunk should be request path. KeyChunk: %v", i+1, keyChunk)
			}
		}
	}
}

func TestRequestMethodAndBasicAuthUsersBuildKeys(t *testing.T) {
	rate.Client().FlushAll()

	limiter := NewLimiter(1, time.Second, &rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	})
	limiter.Methods = []string{"GET"}
	limiter.BasicAuthUsers = []string{"bro"}

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		t.Errorf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")
	request.SetBasicAuth("bro", "tato")

	for _, keys := range BuildKeys(limiter, request) {
		if len(keys) != 4 {
			t.Errorf("Keys should be made of 4 parts. Keys: %v", keys)
		}
		for i, keyChunk := range keys {
			if i == 0 && keyChunk != request.Header.Get("X-Real-IP") {
				t.Errorf("The (%v) chunk should be remote IP. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 1 && keyChunk != request.URL.Path {
				t.Errorf("The (%v) chunk should be request path. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 2 && keyChunk != "GET" {
				t.Errorf("The (%v) chunk should be request method. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 3 && keyChunk != "bro" {
				t.Errorf("The (%v) chunk should be basic auth user. KeyChunk: %v", i+1, keyChunk)
			}
		}
	}
}

func TestRequestMethodCustomHeadersAndBasicAuthUsersBuildKeys(t *testing.T) {
	rate.Client().FlushAll()

	limiter := NewLimiter(1, time.Second, &rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	})
	limiter.Methods = []string{"GET"}
	limiter.Headers = []string{"X-Auth-Token"}
	limiter.BasicAuthUsers = []string{"bro"}

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		t.Errorf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")
	request.Header.Set("X-Auth-Token", "totally-top-secret")
	request.SetBasicAuth("bro", "tato")

	for _, keys := range BuildKeys(limiter, request) {
		if len(keys) != 6 {
			t.Errorf("Keys should be made of 4 parts. Keys: %v", keys)
		}
		for i, keyChunk := range keys {
			if i == 0 && keyChunk != request.Header.Get("X-Real-IP") {
				t.Errorf("The (%v) chunk should be remote IP. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 1 && keyChunk != request.URL.Path {
				t.Errorf("The (%v) chunk should be request path. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 2 && keyChunk != "GET" {
				t.Errorf("The (%v) chunk should be request method. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 3 && keyChunk != "X-Auth-Token" {
				t.Errorf("The (%v) chunk should be request header. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 4 && keyChunk != "totally-top-secret" {
				t.Errorf("The (%v) chunk should be request path. KeyChunk: %v", i+1, keyChunk)
			}
			if i == 5 && keyChunk != "bro" {
				t.Errorf("The (%v) chunk should be basic auth user. KeyChunk: %v", i+1, keyChunk)
			}
		}
	}

}

func TestLimitHandler(t *testing.T) {
	rate.Client().FlushAll()

	limiter := config.NewLimiter(1, time.Second, &rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	})
	limiter.IPLookups = []string{"X-Real-IP", "RemoteAddr", "X-Forwarded-For"}
	limiter.Methods = []string{"POST"}

	handler := LimitHandler(limiter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`hello world`))
	}))

	req, err := http.NewRequest("POST", "/doesntmatter", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// Should not be limited
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	//Should be limited
	if status := rr.Code; status != http.StatusTooManyRequests {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusTooManyRequests)
	}
}

func TestLimitHandlerAndSetExactAPIRateLimit(t *testing.T) {
	rate.Client().FlushAll()

	limiter := config.NewLimiter(1, time.Second, &rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	})
	limiter.IPLookups = []string{"X-Real-IP", "RemoteAddr", "X-Forwarded-For"}
	limiter.Methods = []string{"POST"}

	Reset()
	mattersMax := 2
	RegisterAPI("/matters", "POST", int64(mattersMax), time.Second)

	handler := LimitHandler(limiter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`hello world`))
	}))

	req, err := http.NewRequest("POST", "/doesntmatter", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")

	req2, err := http.NewRequest("POST", "/matters", nil)
	if err != nil {
		t.Fatal(err)
	}
	req2.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// Should not be limited
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	//Should be limited
	if status := rr.Code; status != http.StatusTooManyRequests {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusTooManyRequests)
	}

	for i := 0; i < mattersMax; i++ {
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req2)
		// Should not be limited
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req2)
	// Should be limited
	if status := rr.Code; status != http.StatusTooManyRequests {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusTooManyRequests)
	}
}

func TestLimitHandlerAndSetRegexpAPIRateLimit(t *testing.T) {
	rate.Client().FlushAll()

	limiter := config.NewLimiter(1, time.Second, &rate.ConfigRedis{
		Host: "127.0.0.1",
		Port: 6379,
		Auth: "",
	})
	limiter.IPLookups = []string{"X-Real-IP", "RemoteAddr", "X-Forwarded-For"}
	limiter.Methods = []string{"POST"}

	Reset()
	mattersMax := 2
	RegisterAPI("/matters/.*", "POST", int64(mattersMax), time.Second)

	handler := LimitHandler(limiter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`hello world`))
	}))

	req, err := http.NewRequest("POST", "/doesntmatter", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")

	req2, err := http.NewRequest("POST", "/matters/0", nil)
	if err != nil {
		t.Fatal(err)
	}
	req2.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")

	req3, err := http.NewRequest("POST", "/matters/1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req3.Header.Set("X-Real-IP", "2601:7:1c82:4097:59a0:a80b:2841:b8c8")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// Should not be limited
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	//Should be limited
	if status := rr.Code; status != http.StatusTooManyRequests {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusTooManyRequests)
	}

	for i := 0; i < mattersMax; i++ {
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req2)
		// Should not be limited
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req2)
	// Should be limited
	if status := rr.Code; status != http.StatusTooManyRequests {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusTooManyRequests)
	}

	for i := 0; i < mattersMax; i++ {
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req3)
		// Should not be limited
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req3)
	// Should be limited
	if status := rr.Code; status != http.StatusTooManyRequests {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusTooManyRequests)
	}
}
