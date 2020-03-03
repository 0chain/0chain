package storagesc

/*
import (
	"0chain.net/chaincore/tokenpool"
	"0chain.net/core/datastore"
)

// ChallengePool of a blobber associated with an allocation.
type ChallengePool struct {
	tokenpool.ZcnPool `json:"pool"`
}

// NewChallengePool with key based on provided values.
func NewChallengePool(upperKey, allocationID string) (cp *ChallengePool) {
	cp = new(ChallengePool)
	cp.SetKey(upperKey, allocationID)
	return
}

// Key of the pool.
func (cp *ChallengePool) Key(upperKey string) datastore.Key {
	return cp.ID
}

// SetKey based on global (upper) key and allocation ID.
func (cp *ChallengePool) SetKey(upperKey, allocationID string) {
	cp.ID = datastore.Key(upperKey + ":challengepool:" + allocationID)
}

// Encode to bytes.
func (cp *ChallengePool) Encode() []byte {
	buff, _ := json.Marshal(cp)
	return buff
}

// Decode from bytes.
func (cp *ChallengePool) Decode(input []byte) error {
	return json.Unmarshal(input, cp)
}

// Load from MPT with actual key.
func (cp *ChallengePool) Load(balances chainState.StateContextI) (err error) {
	var b []byte
	if b, err = balances.GetTrieNode(cp.Key()); err != nil {
		return
	}
	return cp.Decode(b)
}

// Save or update in MPT with actual key.
func (cp *ChallengePool) Save(balances chainState.StateContextI) (err error) {
	_, err = balances.InsertTrieNode(cp.Key(), cp)
	return
}
*/
