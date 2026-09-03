package ratelimit

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gogf/gf/v2/net/ghttp"
	"one_adventure_servicekit/httpresponse"
	servicetoken "one_adventure_servicekit/token"
)

type Middleware struct {
	tokenBucket *TokenBucket
	leakyBucket *LeakyBucket
	userWindow  *UserWindow
}

func New(ctx context.Context) (*Middleware, error) {
	config, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	middleware := &Middleware{}
	if config.TokenBucket.Enabled {
		middleware.tokenBucket = NewTokenBucket(config.TokenBucket.Capacity, config.TokenBucket.Rate)
	}
	if config.LeakyBucket.Enabled {
		middleware.leakyBucket = NewLeakyBucket(config.LeakyBucket.Capacity, config.LeakyBucket.Rate)
	}
	if config.UserWindow.Enabled {
		middleware.userWindow = NewUserWindow(config.UserWindow.Limit, config.UserWindow.Window)
	}
	return middleware, nil
}

func (m *Middleware) TokenBucket(request *ghttp.Request) {
	if m.tokenBucket != nil && !m.tokenBucket.Allow() {
		reject(request, "token bucket rate limit exceeded", 1)
		return
	}
	request.Middleware.Next()
}

func (m *Middleware) LeakyBucket(request *ghttp.Request) {
	if m.leakyBucket != nil && !m.leakyBucket.Allow() {
		reject(request, "leaky bucket rate limit exceeded", 1)
		return
	}
	request.Middleware.Next()
}

// UserWindow must run after authentication because auth places UserInfo in
// the request context. Public endpoints without an identity are not counted.
func (m *Middleware) UserWindow(request *ghttp.Request) {
	if m.userWindow != nil {
		if user, ok := servicetoken.UserInfoFromContext(request.Context()); ok && user.ID > 0 && !m.userWindow.Allow(user.ID) {
			reject(request, "user rate limit exceeded", int(m.userWindow.window.Seconds()))
			return
		}
	}
	request.Middleware.Next()
}

func reject(request *ghttp.Request, message string, retryAfter int) {
	if retryAfter < 1 {
		retryAfter = 1
	}
	request.Response.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	request.Response.Status = http.StatusTooManyRequests
	request.Response.WriteJson(httpresponse.Failure(http.StatusTooManyRequests, message))
}
