package interestpoolsc

import (
	"encoding/json"
	"time"

	"0chain.net/pkg/currency"

	// "0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

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

// swagger:model poolStat
type poolStat struct {
	ID           datastore.Key    `json:"pool_id"`
	StartTime    common.Timestamp `json:"start_time"`
	Duartion     time.Duration    `json:"duration"`
	TimeLeft     time.Duration    `json:"time_left"`
	Locked       bool             `json:"locked"`
	APR          float64          `json:"apr"`
	TokensEarned currency.Coin    `json:"tokens_earned"`
	Balance      currency.Coin    `json:"balance"`
}

func (ps *poolStat) encode() []byte {
	buff, _ := json.Marshal(ps)
	return buff
}

func (ps *poolStat) decode(input []byte) error {
	err := json.Unmarshal(input, ps)
	return err
}
