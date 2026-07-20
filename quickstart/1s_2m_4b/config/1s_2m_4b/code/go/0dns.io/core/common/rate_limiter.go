package common

import (
	"net/http"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/spf13/viper"
)

/*ReqRespHandlerf - a type for the default hanlder signature */
type ReqRespHandlerf func(w http.ResponseWriter, r *http.Request)

type ratelimit struct {
	Limiter           *limiter.Limiter
	RateLimit         bool
	RequestsPerSecond float64
}

var userRateLimit *ratelimit

func (rl *ratelimit) init() {
	if rl.RequestsPerSecond == 0 {
		rl.RateLimit = false
		return
	}
	rl.RateLimit = true
	rl.Limiter = tollbooth.NewLimiter(rl.RequestsPerSecond, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour}).
		SetIPLookups([]string{"RemoteAddr", "X-Forwarded-For", "X-Real-IP"}).
		SetMethods([]string{"GET", "POST", "PUT", "DELETE"})
}

//ConfigRateLimits - configure the rate limits
func ConfigRateLimits() {
	userRl := viper.GetFloat64("handlers.rate_limit")
	userRateLimit = &ratelimit{RequestsPerSecond: userRl}
	userRateLimit.init()
}

//UserRateLimit - rate limiting for end user handlers
func UserRateLimit(handler ReqRespHandlerf) ReqRespHandlerf {
	if !userRateLimit.RateLimit {
		return handler
	}
	return func(writer http.ResponseWriter, request *http.Request) {
		tollbooth.LimitFuncHandler(userRateLimit.Limiter, handler).ServeHTTP(writer, request)
	}
}
