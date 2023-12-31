//go:build development
// +build development

package node

import (
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"time"

	"0chain.net/core/config"
)

type Route struct {
	To    string
	Delay time.Duration
}

var routes = make(map[string]*Route, 10)

// InduceDelay - incude network delay
func (n *Node) InduceDelay(toNode *Node) {
	if route, ok := routes[toNode.N2NHost]; ok {
		logging.Logger.Info("Jayash induce delay", zap.Any("route", route))
		time.Sleep(route.Delay)
	}
	return
}

func ReadNetworkDelays(file string) {
	delayConfig := config.ReadConfig(file)
	delay := delayConfig.Get("delay")

	configRoutes, ok := delay.([]interface{})

	logging.Logger.Info("Jayash read network delay1",
		zap.Any("file", file),
		zap.Any("delay", delay),
		zap.Any("n2n_host", Self.Underlying().N2NHost),
		zap.Any("delayConfig", delayConfig),
		zap.Any("configRoutes", configRoutes),
		zap.Any("ok", ok))

	if ok {
		for _, route := range configRoutes {
			if routeMap, ok := route.(map[interface{}]interface{}); ok {
				from := routeMap["from"].(string)
				to := routeMap["to"].(string)
				delayTime := routeMap["time"].(int)

				logging.Logger.Info("Jayash read network delay2",
					zap.Any("from", from),
					zap.Any("to", to),
					zap.Any("delayTime", delayTime),
					zap.Any("n2n_host", Self.Underlying().N2NHost))

				if Self.Underlying().N2NHost == from {
					routes[to] = &Route{To: to, Delay: time.Duration(delayTime) * time.Millisecond}
				}
			}
		}
	}
}
