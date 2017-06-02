// Package config provides data structure to configure rate-limiter.
package config

import (
	"log"
	"sort"
	"sync"
	"time"

	rate "github.com/wallstreetcn/rate/redis"
)

// NewLimiter is a constructor for Limiter.
func NewLimiter(max int64, ttl time.Duration, conf *rate.ConfigRedis) *Limiter {
	limiter := &Limiter{Max: max, TTL: ttl}
	limiter.MessageContentType = "text/plain; charset=utf-8"
	limiter.Message = "You have reached maximum request limit."
	limiter.StatusCode = 429
	limiter.tokenBuckets = make(map[string]*rate.Limiter)
	limiter.IPLookups = []string{"RemoteAddr", "X-Forwarded-For", "X-Real-IP"}

	if err := rate.SetRedis(conf); err != nil {
		log.Println("fail to set rate limiter's redis: ", err)
	}

	return limiter
}

// Limiter is a config struct to limit a particular request handler.
type Limiter struct {
	// HTTP message when limit is reached.
	Message string

	// Content-Type for Message
	MessageContentType string

	// HTTP status code when limit is reached.
	StatusCode int

	// Maximum number of requests to limit per duration.
	Max int64

	// Duration of rate-limiter.
	TTL time.Duration

	// List of places to look up IP address.
	// Default is "RemoteAddr", "X-Forwarded-For", "X-Real-IP".
	// You can rearrange the order as you like.
	IPLookups []string

	// List of HTTP Methods to limit (GET, POST, PUT, etc.).
	// Empty means limit all methods.
	Methods []string

	// List of HTTP headers to limit.
	// Empty means skip headers checking.
	Headers []string

	// List of basic auth usernames to limit.
	BasicAuthUsers []string

	// Throttler struct
	tokenBuckets map[string]*rate.Limiter

	sync.RWMutex
}

// By is the type of a "less" function that defines the ordering of its RateLimit arguments.
type By func(l1, l2 *RateLimit) bool

// Sort sorts the rate limits by specified order.
func (by By) Sort(limits []RateLimit) {
	ls := &limitSorter{
		limits: limits,
		by:     by,
	}
	sort.Sort(ls)
}

type limitSorter struct {
	limits []RateLimit
	by     func(l1, l2 *RateLimit) bool
}

func (s *limitSorter) Len() int {
	return len(s.limits)
}

func (s *limitSorter) Swap(i, j int) {
	s.limits[i], s.limits[j] = s.limits[j], s.limits[i]
}

func (s *limitSorter) Less(i, j int) bool {
	return s.by(&s.limits[i], &s.limits[j])
}

// RateLimit defines the API's rate limit.
type RateLimit struct {
	Key LimitKey
	Val LimitValue
}

// LimitKey defines the limited API's key.
type LimitKey struct {
	Path   string
	Method string
}

// LimitValue defines the API's rate limit.
type LimitValue struct {
	Max int64
	TTL time.Duration
}

// LimitReached returns a bool indicating if the Bucket identified by key ran out of tokens.
func (l *Limiter) LimitReached(key string, limitVal *LimitValue) bool {
	l.Lock()
	defer l.Unlock()
	if _, found := l.tokenBuckets[key]; !found {
		var (
			TTL = l.TTL
			Max = l.Max
		)
		if limitVal != nil {
			TTL = limitVal.TTL
			Max = limitVal.Max
		}
		l.tokenBuckets[key] = rate.NewLimiter(rate.Every(TTL), int(Max), key)
	}

	return !l.tokenBuckets[key].AllowN(time.Now(), 1)
}
