package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
)

type blobberTotalsField int

const (
	BsStakeTotals blobberTotalsField = iota
	BsCapacities
	BsUsed
)

// blobber id x delegate id
type blobberStakeTotals struct {
	StakeTotals map[string]state.Balance `json:"stake_totals"`
	Capacities  map[string]int64         `json:"capacities"`
	Used        map[string]int64         `json:"used"`
}

func newBlobberStakeTotals() *blobberStakeTotals {
	return &blobberStakeTotals{
		StakeTotals: make(map[string]state.Balance),
		Capacities:  make(map[string]int64),
		Used:        make(map[string]int64),
	}
}

func (bs *blobberStakeTotals) Encode() []byte {
	var b, err = json.Marshal(bs)
	if err != nil {
		panic(err)
	}
	return b
}

func (bs *blobberStakeTotals) Decode(p []byte) error {
	return json.Unmarshal(p, bs)
}

func (bs blobberStakeTotals) add(id string, value int64, field blobberTotalsField) {
	logging.Logger.Info("blobberStakeTotals add",
		zap.Any("id", id),
		zap.Any("value", value),
		zap.Any("field", field),
	)

	if _, ok := bs.StakeTotals[id]; !ok {
		bs.StakeTotals[id] = 0
		bs.Used[id] = 0
		bs.Capacities[id] = 0
	}

	switch field {
	case BsStakeTotals:
		bs.StakeTotals[id] = state.Balance(value)
	case BsCapacities:
		bs.Capacities[id] = value
	case BsUsed:
		bs.Used[id] = value
	}
}

func (bs *blobberStakeTotals) remove(blobberId string) {
	logging.Logger.Info("blobberStakeTotals remove",
		zap.Any("blobberId", blobberId),
	)

	delete(bs.StakeTotals, blobberId)
	delete(bs.Capacities, blobberId)
	delete(bs.Used, blobberId)
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
