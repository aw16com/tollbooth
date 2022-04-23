// Package tollbooth provides rate-limiting logic to HTTP request handler.
package tollbooth

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	rate "github.com/aw16com/rate/redis"
	"github.com/aw16com/tollbooth/config"
	"github.com/aw16com/tollbooth/errors"
	"github.com/aw16com/tollbooth/libstring"
)

var (
	settings []config.RateLimit
)

func init() {
	settings = make([]config.RateLimit, 0)
}

// NewLimiter is a convenience function to config.NewLimiter.
func NewLimiter(max int64, ttl time.Duration, conf *rate.ConfigRedis) *config.Limiter {
	return config.NewLimiter(max, ttl, conf)
}

// LimitByKeys keeps track number of request made by keys separated by pipe.
// It returns HTTPError when limit is exceeded.
func LimitByKeys(limiter *config.Limiter, keys []string, limitVal *config.LimitValue) *errors.HTTPError {
	if limiter.LimitReached(strings.Join(keys, "|"), limitVal) {
		return &errors.HTTPError{Message: limiter.Message, StatusCode: limiter.StatusCode}
	}

	return nil
}

// LimitByRequest builds keys based on http.Request struct,
// loops through all the keys, and check if any one of them returns HTTPError.
func LimitByRequest(limiter *config.Limiter, r *http.Request) *errors.HTTPError {
	sliceKeys := BuildKeys(limiter, r)

	// Loop sliceKeys and check if one of them has error.
	for _, keys := range sliceKeys {
		httpError := LimitByKeys(limiter, keys, matchLimit(r))
		if httpError != nil {
			return httpError
		}
	}

	return nil
}

// BuildKeys generates a slice of keys to rate-limit by given config and request structs.
func BuildKeys(limiter *config.Limiter, r *http.Request) [][]string {
	remoteIP := libstring.RemoteIP(limiter.IPLookups, r)
	path := r.URL.Path
	sliceKeys := make([][]string, 0)

	// Don't BuildKeys if remoteIP is blank.
	if remoteIP == "" {
		return sliceKeys
	}

	if limiter.Methods != nil && limiter.Headers != nil && limiter.BasicAuthUsers != nil {
		// Limit by HTTP methods and HTTP headers+values and Basic Auth credentials.
		if libstring.StringInSlice(limiter.Methods, r.Method) {
			for _, headerKey := range limiter.Headers {
				if r.Header.Get(headerKey) != "" {
					// If header values are empty, rate-limit all request with headerValue.
					username, _, ok := r.BasicAuth()
					if ok && libstring.StringInSlice(limiter.BasicAuthUsers, username) {
						sliceKeys = append(sliceKeys, []string{remoteIP, path, r.Method, headerKey, r.Header.Get(headerKey), username})
					}

				}
			}
		}
	} else if limiter.Methods != nil && limiter.Headers != nil {
		// Limit by HTTP methods and HTTP headers+values.
		if libstring.StringInSlice(limiter.Methods, r.Method) {
			for _, headerKey := range limiter.Headers {
				if r.Header.Get(headerKey) != "" {
					// If header values are empty, rate-limit all request with headerKey.
					sliceKeys = append(sliceKeys, []string{remoteIP, path, r.Method, headerKey, r.Header.Get(headerKey)})
				}
			}
		}
	} else if limiter.Methods != nil && limiter.BasicAuthUsers != nil {
		// Limit by HTTP methods and Basic Auth credentials.
		if libstring.StringInSlice(limiter.Methods, r.Method) {
			username, _, ok := r.BasicAuth()
			if ok && libstring.StringInSlice(limiter.BasicAuthUsers, username) {
				sliceKeys = append(sliceKeys, []string{remoteIP, path, r.Method, username})
			}
		}
	} else if limiter.Methods != nil {
		// Limit by HTTP methods.
		if libstring.StringInSlice(limiter.Methods, r.Method) {
			sliceKeys = append(sliceKeys, []string{remoteIP, path, r.Method})
		}
	} else if limiter.Headers != nil {
		// Limit by HTTP headers+values.
		for _, headerKey := range limiter.Headers {
			if r.Header.Get(headerKey) != "" {
				// If header values are empty, rate-limit all request with headerKey.
				sliceKeys = append(sliceKeys, []string{remoteIP, path, headerKey, r.Header.Get(headerKey)})
			}
		}
	} else if limiter.BasicAuthUsers != nil {
		// Limit by Basic Auth credentials.
		username, _, ok := r.BasicAuth()
		if ok && libstring.StringInSlice(limiter.BasicAuthUsers, username) {
			sliceKeys = append(sliceKeys, []string{remoteIP, path, username})
		}
	} else {
		// Default: Limit by remoteIP and path.
		sliceKeys = append(sliceKeys, []string{remoteIP, path})
	}

	return sliceKeys
}

// SetResponseHeaders configures X-Rate-Limit-Limit and X-Rate-Limit-Duration.
func SetResponseHeaders(limiter *config.Limiter, w http.ResponseWriter) {
	w.Header().Add("X-Rate-Limit-Limit", strconv.FormatInt(limiter.Max, 10))
	w.Header().Add("X-Rate-Limit-Duration", limiter.TTL.String())
}

// LimitHandler is a middleware that performs rate-limiting given http.Handler struct.
func LimitHandler(limiter *config.Limiter, next http.Handler) http.Handler {
	middle := func(w http.ResponseWriter, r *http.Request) {
		SetResponseHeaders(limiter, w)

		httpError := LimitByRequest(limiter, r)
		if httpError != nil {
			// w.Header().Add("Content-Type", limiter.MessageContentType)
			w.WriteHeader(httpError.StatusCode)
			w.Write([]byte(httpError.Message))
			return
		}

		// There's no rate-limit error, serve the next handler.
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(middle)
}

// RegisterAPI registers rate limit for the specified API.
func RegisterAPI(path string, method string, max int64, duration time.Duration) {
	settings = append(settings, config.RateLimit{
		Key: config.LimitKey{
			Path:   path,
			Method: method,
		},
		Val: config.LimitValue{
			Max: max,
			TTL: duration,
		},
	})

	config.By(func(l1, l2 *config.RateLimit) bool {
		return len(l1.Key.Path) > len(l2.Key.Path)
	}).Sort(settings)
}

// Reset resets the rate limit settings.
func Reset() {
	settings = make([]config.RateLimit, 0)
}

func matchLimit(r *http.Request) *config.LimitValue {
	path := r.URL.Path
	method := r.Method
	for i, ratelimit := range settings {
		if ratelimit.Key.Method == method {
			matched, _ := regexp.MatchString(ratelimit.Key.Path, path)
			if matched {
				return &settings[i].Val
			}
		}
	}
	return nil
}

// LimitFuncHandler is a middleware that performs rate-limiting given request handler function.
func LimitFuncHandler(limiter *config.Limiter, nextFunc func(http.ResponseWriter, *http.Request)) http.Handler {
	return LimitHandler(limiter, http.HandlerFunc(nextFunc))
}
