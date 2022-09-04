//go:build development
// +build development

package node

import (
	"time"

	"0chain.net/chaincore/config"
)

type Route struct {
	To    string
	Delay time.Duration
}

var routes = make(map[string]*Route, 10)

//InduceDelay - incude network delay
func (n *Node) InduceDelay(toNode *Node) {
	if route, ok := routes[toNode.N2NHost]; ok {
		time.Sleep(route.Delay)
	}
	return
}

func ReadNetworkDelays(file string) {
	delayConfig := config.ReadConfig(file)
	delay := delayConfig.Get("delay")
	if configRoutes, ok := delay.([]interface{}); ok {
		for _, route := range configRoutes {
			if routeMap, ok := route.(map[interface{}]interface{}); ok {
				from := routeMap["from"].(string)
				to := routeMap["to"].(string)
				delayTime := routeMap["time"].(int)
				if Self.Underlying().N2NHost == from {
					routes[to] = &Route{To: to, Delay: time.Duration(delayTime) * time.Millisecond}
				}
			}
		}
	}
}
