package event

import (
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
)

type RewardMint struct {
	gorm.Model
	Amount       int64  `json:"amount"`
	BlockNumber  int64  `json:"block_number"`
	ClientID     string `json:"client_id"`     // wallet ID
	PoolID       string `json:"pool_id"`       // stake pool ID
	ProviderType string `json:"provider_type"` // blobber or validator
	ProviderID   string `json:"provider_id"`
}

type RewardMintQuery struct {
	StartBlock   int       `json:"start_block"`
	EndBlock     int       `json:"end_block"`
	DataPoints   int64     `json:"data_points"`
	StartDate    time.Time `json:"start_date"`
	EndDate      time.Time `json:"end_date"`
	ClientID     string    `json:"client_id"`
	PoolID       string    `json:"pool_id"`
	ProviderType string    `json:"provider_type"`
	ProviderID   string    `json:"provider_id"`
}

// GetRewardClaimedTotalBetweenBlocks returns the sum of amounts
// from rewards table  matching the given query
func (edb *EventDb) GetRewardClaimedTotalBetweenBlocks(query RewardMintQuery) ([]int64, error) {
	var rewards []int64

	step := math.Ceil(float64(query.EndBlock-query.StartBlock) / float64(query.DataPoints))
	rawQuery := fmt.Sprintf(`
		WITH
		ranges AS (
			SELECT t AS r_min, t + %[3]v - 1 AS r_max
			FROM generate_series(%[1]v, %[2]v, %[3]v) as t
		)
		SELECT coalesce(sum(amount), 0) as val
		FROM ranges r
		LEFT JOIN reward_mints rw ON rw.block_number BETWEEN r.r_min AND r.r_max AND client_id = '%[4]v'
		GROUP BY r.r_min
		ORDER BY r.r_min;
	`, query.StartBlock, query.EndBlock, step, query.ClientID)

	return rewards, edb.Store.Get().Raw(rawQuery).Scan(&rewards).Error
}

func (edb *EventDb) GetRewardClaimedTotalBetweenDates(query RewardMintQuery) ([]int64, error) {
	var rewards []int64
	rawQuery := fmt.Sprintf(`
		WITH
		block_info as (
			select b.from as from, b.to as to, ceil(((b.to::FLOAT - b.from::FLOAT)/ %d) + 1)::INTEGER as step from
				(select min(round) as from, max(round) as to from blocks where creation_date between %d and %d) as b
		),
		ranges AS (
			SELECT t AS r_min, t+(select step from block_info)-1 AS r_max
			FROM generate_series((select "from" from block_info), (select "to" from block_info), (select step from block_info)) as t
		)
		SELECT coalesce(%s, 0) as val
		FROM ranges r
		LEFT JOIN reward_mints rw ON rw.block_number BETWEEN r.r_min AND r.r_max AND client_id = '%s'
		GROUP BY r.r_min
		ORDER BY r.r_min;
	`, query.DataPoints, query.StartDate.UnixNano(), query.EndDate.UnixNano(), "sum(amount)", query.ClientID)

	return rewards, edb.Store.Get().Raw(rawQuery).Scan(&rewards).Error
}

func (edb *EventDb) addRewardMint(reward RewardMint) error {
	return edb.Store.Get().Create(&reward).Error
}
