## echo

[Echo](https://github.com/labstack/echo) middleware for rate limiting HTTP requests.


## Five Minutes Tutorial

```
package main

import (
	"time"

	"github.com/labstack/echo"
	"github.com/wallstreetcn/tollbooth"
	tollbooth_echo "github.com/wallstreetcn/tollbooth/thirdparty/echo"
)

func main() {
	e := echo.New()

	// Create a limiter struct.
	limiter := tollbooth.NewLimiter(1, time.Second)

	e.GET("/", echo.HandlerFunc(func(c echo.Context) error {
		return c.String(200, "Hello, World!")
	}), tollbooth_echo.LimitHandler(limiter))

	e.Start(":4444")
}

```