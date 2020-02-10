package node

import (
	"sync"
	"time"
)

//Info - (informal) info of a node that can be shared with other nodes
type Info struct {
	mx sync.RWMutex

	AsOf                    time.Time     `json:"-"`
	BuildTag                string        `json:"build_tag"`
	StateMissingNodes       int64         `json:"state_missing_nodes"`
	MinersMedianNetworkTime time.Duration `json:"miners_median_network_time"`
	AvgBlockTxns            int           `json:"avg_block_txns"`
}

// SetMinersMedianNetworkTime asynchronously.
func (i *Info) SetMinersMedianNetworkTime(mmnt time.Duration) {
	i.mx.Lock()
	defer i.mx.Unlock()

	i.MinersMedianNetworkTime = mmnt
}

// Copy returns copy of the Info.
func (i *Info) Copy() (cp Info) {
	i.mx.Lock()
	defer i.mx.Unlock()

	cp = *i
	return
}
