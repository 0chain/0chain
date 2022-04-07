package rest

import (
	"encoding/json"
	"fmt"
	"time"

	"0chain.net/smartcontract/interestpoolsc"

	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

// swagger:model poolStats
type poolStats struct {
	Stats []*poolStat `json:"stats"`
}

func (ps *poolStats) encode() []byte {
	buff, _ := json.Marshal(ps)
	return buff
}

func (ps *poolStats) decode(input []byte) error {
	err := json.Unmarshal(input, ps)
	return err
}

func (ps *poolStats) addStat(p *poolStat) {
	ps.Stats = append(ps.Stats, p)
}

type poolStat struct {
	ID           datastore.Key    `json:"pool_id"`
	StartTime    common.Timestamp `json:"start_time"`
	Duartion     time.Duration    `json:"duration"`
	TimeLeft     time.Duration    `json:"time_left"`
	Locked       bool             `json:"locked"`
	APR          float64          `json:"apr"`
	TokensEarned state.Balance    `json:"tokens_earned"`
	Balance      state.Balance    `json:"balance"`
}

func (ps *poolStat) encode() []byte {
	buff, _ := json.Marshal(ps)
	return buff
}

func (ps *poolStat) decode(input []byte) error {
	err := json.Unmarshal(input, ps)
	return err
}

func getPoolStats(pool *interestpoolsc.InterestPool, t time.Time) (*poolStat, error) {
	stat := &poolStat{}
	statBytes := pool.LockStats(t)
	err := stat.decode(statBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	stat.ID = pool.ID
	stat.Locked = pool.IsLocked(t)
	stat.Balance = pool.Balance
	stat.APR = pool.APR
	stat.TokensEarned = pool.TokensEarned
	return stat, nil
}
