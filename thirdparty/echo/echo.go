package echo

import (
	"github.com/aw16com/tollbooth"
	"github.com/aw16com/tollbooth/config"
)

// LimitMiddleware builds an API limit middleware for labstack echo framework
func LimitMiddleware(limiter *config.Limiter) echo.MiddlewareFunc {
	return func(h echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			httpError := tollbooth.LimitByRequest(limiter, c.Request())
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
