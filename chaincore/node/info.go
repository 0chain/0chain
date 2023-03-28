package node

import (
	"time"
)

//go:generate msgp -io=false -tests=false -v

//Info - (informal) info of a node that can be shared with other nodes
type Info struct {
	AsOf                    time.Time     `json:"-" msgpack:"-" msg:"-"`
	BuildTag                string        `json:"build_tag"`
	StateMissingNodes       int64         `json:"state_missing_nodes"`
	MinersMedianNetworkTime time.Duration `json:"miners_median_network_time"`
	AvgBlockTxns            int           `json:"avg_block_txns"`
}
