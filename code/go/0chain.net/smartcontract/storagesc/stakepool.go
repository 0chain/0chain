package storagesc

import (
	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/core/datastore"
)

// StakePool of a Blobber.
type StakePool struct {
	tokenpool.ZcnPool `json:"pool"`
}

// NewStakePool with key based on provided values.
func NewStakePool(upperKey, blobberID string) (sp *StakePool) {
	sp = new(StakePool)
	sp.SetKey(upperKey, blobberID)
	return
}

// Key of the pool.
func (sp *StakePool) Key() datastore.Key {
	return sp.ID
}

// Set key based on global (upper) key and blobber ID.
func (sp *StakePool) SetKey(upperKey, blobberID string) {
	sp.ID = datastore.Key(upperKey + ":stakepool:" + blobberID)
}

// Encode to bytes.
func (sp *StakePool) Encode() []byte {
	buff, _ := json.Marshal(sp)
	return buff
}

// Decode from bytes.
func (sp *StakePool) Decode(input []byte) error {
	return json.Unmarshal(input, sp)
}

// Load from MPT with actual key.
func (sp *StakePool) Load(balances chainState.StateContextI) (err error) {
	var b []byte
	if b, err = balances.GetTrieNode(sp.Key()); err != nil {
		return
	}
	return sp.Decode(b)
}

// Save or update in MPT with actual key.
func (sp *StakePool) Save(balances chainState.StateContextI) (err error) {
	_, err = balances.InsertTrieNode(sp.Key(), sp)
	return
}
