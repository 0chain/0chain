package node

import "time"

//Info - (informal) info of a node that can be shared with other nodes
type Info struct {
	AsOf                     time.Time     `json:"-"`
	BuildTag                 string        `json:"build_tag"`
	StateMissingNodes        int64         `json:"state_missing_nodes"`
	MinersMedianNetworkTime  time.Duration `json:"miners_median_network_time"`
}
