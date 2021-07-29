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
type blobberStakeTotals struct {
	Totals map[string]state.Balance `json:"Totals"`
}

func newBlobberStakeTotals() *blobberStakeTotals {
	return &blobberStakeTotals{Totals: make(map[string]state.Balance)}
}

func (bs *blobberStakeTotals) Encode() []byte {
	var b, err = json.Marshal(bs)
	ss := string(b)
	fmt.Println("bst mashal", ss)

	if err != nil {
		panic(err) // must never happens
	}
	return b
}

func (bs *blobberStakeTotals) Decode(p []byte) error {
	return json.Unmarshal(p, bs)
}

func (bs *blobberStakeTotals) save(balances cstate.StateContextI) error {
	_, err := balances.InsertTrieNode(ALL_BLOBBER_STAKES_KEY, bs)
	return err
}

func getBlobberStakeTotalsBytes(balances cstate.StateContextI) ([]byte, error) {
	var val util.Serializable
	val, err := balances.GetTrieNode(ALL_BLOBBER_STAKES_KEY)
	if err != nil {
		return nil, err
	}
	return val.Encode(), nil
}

func getBlobberStakeTotals(balances cstate.StateContextI) (*blobberStakeTotals, error) {
	var bsBytes []byte
	var err error
	bs := newBlobberStakeTotals()
	if bsBytes, err = getBlobberStakeTotalsBytes(balances); err != nil {
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
