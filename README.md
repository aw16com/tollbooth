[![Build Status](https://travis-ci.org/wallstreetcn/tollbooth.svg?branch=master)](https://travis-ci.org/wallstreetcn/tollbooth)
[![GoDoc](https://godoc.org/github.com/wallstreetcn/tollbooth?status.svg)](http://godoc.org/github.com/wallstreetcn/tollbooth)
[![Coverage Status](https://coveralls.io/repos/github/wallstreetcn/tollbooth/badge.svg?branch=master)](https://coveralls.io/github/wallstreetcn/tollbooth?branch=master)
[![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/wallstreetcn/tollbooth/master/LICENSE)

## Tollbooth

This is a generic middleware to rate-limit HTTP requests.

**NOTE:** This library is considered finished, any new activities are probably centered around `thirdparty` modules.


## Five Minutes Tutorial
```go
package main

import (
    "github.com/wallstreetcn/tollbooth"
    "net/http"
    "time"
)

func HelloHandler(w http.ResponseWriter, req *http.Request) {
    w.Write([]byte("Hello, World!"))
}

func main() {
    // Create a request limiter per handler.
    http.Handle("/", tollbooth.LimitFuncHandler(tollbooth.NewLimiter(1, time.Second), HelloHandler))
    http.ListenAndServe(":12345", nil)
}
```

## Features

1. Rate-limit by request's remote IP, path, methods, custom headers, & basic auth usernames.  
Register API limit for the specified `path` and `method` using regexp.
    ```go
    limiter := tollbooth.NewLimiter(1, time.Second)

    // Configure list of places to look for IP address.
    // By default it's: "RemoteAddr", "X-Forwarded-For", "X-Real-IP"
    // If your application is behind a proxy, set "X-Forwarded-For" first.
    limiter.IPLookups = []string{"RemoteAddr", "X-Forwarded-For", "X-Real-IP"}

    // Limit only GET and POST requests.
    limiter.Methods = []string{"GET", "POST"}

    // Limit request headers containing certain values.
    // Typically, you prefetched these values from the database.
    limiter.Headers = []string{"X-Access-Token"}

    // Limit based on basic auth usernames.
    // Typically, you prefetched these values from the database.
    limiter.BasicAuthUsers = []string{"bob", "joe", "wallstreetcn"}

    // Rate-Limit the expensive API with 1 ops/min.
    tollbooth.RegisterAPI("/some-expensive-api", "POST", 1, time.Minute)
    ```

2. Each request handler can be rate-limited individually.

3. Compose your own middleware by using `LimitByKeys()`.

4. Tollbooth does not require external storage since it uses an algorithm called [Token Bucket](http://en.wikipedia.org/wiki/Token_bucket) [(Go library: golang.org/x/time/rate)](//godoc.org/golang.org/x/time/rate).


# Other Web Frameworks

Support for other web frameworks are defined under `/thirdparty` directory.
