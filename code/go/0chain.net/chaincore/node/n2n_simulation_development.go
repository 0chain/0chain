// +build development

package node

import (
	"0chain.net/chaincore/config"
	"time"
)

type Route struct {
	To    string
	Delay time.Duration
}

var routes = make(map[string]*Route, 10)

//InduceDelay - incude network delay
func (nd *Node) InduceDelay(toNode *Node) {
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
				if Self.Node.N2NHost == from {
					routes[to] = &Route{To: to, Delay: time.Duration(delayTime) * time.Millisecond}
				}
			}
		}
	}
}
