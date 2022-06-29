package common

import (
	"net/http"
	"strings"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"

	"0chain.net/core/viper"
)

type ratelimit struct {
	Limiter           *limiter.Limiter
	RateLimit         bool
	RequestsPerSecond float64
}

var userRateLimit *ratelimit
var n2nRateLimit *ratelimit

func (rl *ratelimit) init() {
	if rl.RequestsPerSecond == 0 {
		rl.RateLimit = false
		return
	}
	rl.RateLimit = true
	rl.Limiter = tollbooth.NewLimiter(rl.RequestsPerSecond, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour}).
		SetIPLookups([]string{"RemoteAddr", "X-Forwarded-For", "X-Real-IP"}).
		SetMethods([]string{"GET", "POST"})
}

//ConfigRateLimits - configure the rate limits
func ConfigRateLimits() {
	userRl := viper.GetFloat64("network.user_handlers.rate_limit")
	userRateLimit = &ratelimit{RequestsPerSecond: userRl}
	userRateLimit.init()

	n2nRl := viper.GetFloat64("network.n2n_handlers.rate_limit")
	n2nRateLimit = &ratelimit{RequestsPerSecond: n2nRl}
	n2nRateLimit.init()
}

//UserRateLimit - rate limiting for end user handlers
func UserRateLimit(handler ReqRespHandlerf) ReqRespHandlerf {
	if !userRateLimit.RateLimit {
		return Recover(handler)
	}
	return func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/_") {
			handler(writer, request)
		} else {
			tollbooth.LimitFuncHandler(userRateLimit.Limiter, Recover(handler)).ServeHTTP(writer, request)
		}
	}
}

//N2NRateLimit - rate limiting for n2n handlers
func N2NRateLimit(handler ReqRespHandlerf) ReqRespHandlerf {
	if !n2nRateLimit.RateLimit {
		return Recover(handler)
	}
	return func(writer http.ResponseWriter, request *http.Request) {
		tollbooth.LimitFuncHandler(n2nRateLimit.Limiter, Recover(handler)).ServeHTTP(writer, request)
	}
}
