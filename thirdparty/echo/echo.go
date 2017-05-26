package echo

import (
	"net/http"
	"strings"

	"github.com/labstack/echo"
	"github.com/wallstreetcn/tollbooth"
	"github.com/wallstreetcn/tollbooth/config"
	"github.com/wallstreetcn/tollbooth/errors"
	"github.com/wallstreetcn/tollbooth/libstring"
)

// LimitMiddleware builds an API limit middleware for labstack echo framework
func LimitMiddleware(limiter *config.Limiter) echo.MiddlewareFunc {
	return func(h echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			httpError := LimitByRequest(limiter, c.Request())
			if httpError != nil {
				return c.String(httpError.StatusCode, httpError.Message)
			}

			err = h(c)
			return err
		}
	}
}

// LimitHandler builds an API limit handler.
func LimitHandler(limiter *config.Limiter) echo.MiddlewareFunc {
	return LimitMiddleware(limiter)
}

// LimitByRequest builds keys based on http.Request struct,
// loops through all the keys, and check if any one of them returns HTTPError.
func LimitByRequest(limiter *config.Limiter, r *http.Request) *errors.HTTPError {
	sliceKeys := BuildKeys(limiter, r)

	// Loop sliceKeys and check if one of them has error.
	for _, keys := range sliceKeys {
		httpError := tollbooth.LimitByKeys(limiter, keys)
		if httpError != nil {
			return httpError
		}
	}

	return nil
}

// StringInSlice finds needle in a slice of strings.
func StringInSlice(sliceString []string, needle string) bool {
	for _, b := range sliceString {
		if b == needle {
			return true
		}
	}
	return false
}

func ipAddrFromRemoteAddr(s string) string {
	idx := strings.LastIndex(s, ":")
	if idx == -1 {
		return s
	}
	return s[:idx]
}

// RemoteIP finds IP Address given http.Request struct.
func RemoteIP(ipLookups []string, r *http.Request) string {
	realIP := r.Header.Get("X-Real-IP")
	forwardedFor := r.Header.Get("X-Forwarded-For")

	for _, lookup := range ipLookups {
		if lookup == "RemoteAddr" {
			return ipAddrFromRemoteAddr(r.RemoteAddr)
		}
		if lookup == "X-Forwarded-For" && forwardedFor != "" {
			// X-Forwarded-For is potentially a list of addresses separated with ","
			parts := strings.Split(forwardedFor, ",")
			for i, p := range parts {
				parts[i] = strings.TrimSpace(p)
			}
			return parts[0]
		}
		if lookup == "X-Real-IP" && realIP != "" {
			return realIP
		}
	}

	return ""
}

// BuildKeys generates a slice of keys to rate-limit by given config and request structs.
func BuildKeys(limiter *config.Limiter, r *http.Request) [][]string {
	remoteIP := RemoteIP(limiter.IPLookups, r)
	path := r.URL.Path
	sliceKeys := make([][]string, 0)

	// Don't BuildKeys if remoteIP is blank.
	if remoteIP == "" {
		return sliceKeys
	}

	if limiter.Methods != nil && limiter.Headers != nil && limiter.BasicAuthUsers != nil {
		// Limit by HTTP methods and HTTP headers+values and Basic Auth credentials.
		if StringInSlice(limiter.Methods, r.Method) {
			for headerKey, headerValues := range limiter.Headers {
				if (headerValues == nil || len(headerValues) <= 0) && r.Header.Get(headerKey) != "" {
					// If header values are empty, rate-limit all request with headerKey.
					username, _, ok := r.BasicAuth()
					if ok && libstring.StringInSlice(limiter.BasicAuthUsers, username) {
						sliceKeys = append(sliceKeys, []string{remoteIP, path, r.Method, headerKey, username})
					}
				} else if len(headerValues) > 0 && r.Header.Get(headerKey) != "" {
					// If header values are not empty, rate-limit all request with headerKey and headerValues.
					for _, headerValue := range headerValues {
						username, _, ok := r.BasicAuth()
						if ok && libstring.StringInSlice(limiter.BasicAuthUsers, username) {
							sliceKeys = append(sliceKeys, []string{remoteIP, path, r.Method, headerKey, headerValue, username})
						}
					}
				}
			}
		}
	} else if limiter.Methods != nil && limiter.Headers != nil {
		// Limit by HTTP methods and HTTP headers+values.
		if libstring.StringInSlice(limiter.Methods, r.Method) {
			for headerKey, headerValues := range limiter.Headers {
				if (headerValues == nil || len(headerValues) <= 0) && r.Header.Get(headerKey) != "" {
					// If header values are empty, rate-limit all request with headerKey.
					sliceKeys = append(sliceKeys, []string{remoteIP, path, r.Method, headerKey, r.Header.Get(headerKey)})
				} else if len(headerValues) > 0 && r.Header.Get(headerKey) != "" {
					// If header values are not empty, rate-limit all request with headerKey and headerValues.
					for _, headerValue := range headerValues {
						sliceKeys = append(sliceKeys, []string{remoteIP, path, r.Method, headerKey, headerValue})
					}
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
		for headerKey, headerValues := range limiter.Headers {
			if (headerValues == nil || len(headerValues) <= 0) && r.Header.Get(headerKey) != "" {
				// If header values are empty, rate-limit all request with headerKey.
				sliceKeys = append(sliceKeys, []string{remoteIP, path, headerKey})

			} else if len(headerValues) > 0 && r.Header.Get(headerKey) != "" {
				// If header values are not empty, rate-limit all request with headerKey and headerValues.
				for _, headerValue := range headerValues {
					sliceKeys = append(sliceKeys, []string{remoteIP, path, headerKey, headerValue})
				}
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
