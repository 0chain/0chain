package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/util"
	"encoding/json"
	"fmt"
)

// blobber id x delegate id
type blobberStakes map[string]state.Balance

func newBlobberStakes() blobberStakes {
	return blobberStakes(make(map[string]state.Balance))
}

func (bs *blobberStakes) Encode() []byte {
	var b, err = json.Marshal(bs)
	if err != nil {
		panic(err) // must never happens
	}
	return b
}

func (bs *blobberStakes) Decode(p []byte) error {
	return json.Unmarshal(p, bs)
}

func (bs *blobberStakes) save(balances cstate.StateContextI) error {
	_, err := balances.InsertTrieNode(ALL_BLOBBER_STAKES_KEY, bs)
	return err
}

func getBlobberStakesBytes(balances cstate.StateContextI) ([]byte, error) {
	var val util.Serializable
	val, err := balances.GetTrieNode(ALL_BLOBBER_STAKES_KEY)
	if err != nil {
		return nil, err
	}
	return val.Encode(), nil
}

func getBlobberStakes(balances cstate.StateContextI) (blobberStakes, error) {
	var bsBytes []byte
	var err error
	bs := newBlobberStakes()
	if bsBytes, err = getBlobberStakesBytes(balances); err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		return bs, nil
	}
	err = bs.Decode(bsBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return bs, nil
}
