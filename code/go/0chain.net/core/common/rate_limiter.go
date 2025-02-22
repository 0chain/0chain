package common

import (
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"net/http"
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

// ConfigRateLimits - configure the rate limits
func ConfigRateLimits() {
	userRl := viper.GetFloat64("network.user_handlers.rate_limit")
	logging.Logger.Info("Jayash_debug ConfigRateLimits", zap.Any("userRl", userRl))
	userRateLimit = &ratelimit{RequestsPerSecond: userRl}
	userRateLimit.init()

	n2nRl := viper.GetFloat64("user_handlersnetwork.n2n_handlers.rate_limit")
	logging.Logger.Info("Jayash_debug ConfigRateLimits", zap.Any("n2nRl", n2nRl))
	n2nRateLimit = &ratelimit{RequestsPerSecond: n2nRl}
	n2nRateLimit.init()
}

// UserRateLimit - rate limiting for end user handlers
func UserRateLimit(handler ReqRespHandlerf) ReqRespHandlerf {
	logging.Logger.Info("Jayash_debug UserRateLimit", zap.Any("userRateLimit", userRateLimit))
	if !userRateLimit.RateLimit {
		return Recover(handler)
	}
	return func(writer http.ResponseWriter, request *http.Request) {
		tollbooth.LimitFuncHandler(userRateLimit.Limiter, Recover(handler)).ServeHTTP(writer, request)
	}
}

// N2NRateLimit - rate limiting for n2n handlers
func N2NRateLimit(handler ReqRespHandlerf) ReqRespHandlerf {
	if !n2nRateLimit.RateLimit {
		return Recover(handler)
	}
	return func(writer http.ResponseWriter, request *http.Request) {
		tollbooth.LimitFuncHandler(n2nRateLimit.Limiter, Recover(handler)).ServeHTTP(writer, request)
	}
}
