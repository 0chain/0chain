package chain

import "0chain.net/util"

/*StateResponse - a struct that returns a state as of a given finalized block */
type StateResponse struct {
	State     util.SecureSerializableValueI `json:"state"`
	Round     int64                         `json:"round"`
	BlockHash string                        `json:"block_hash"`
}
