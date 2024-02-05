package storagesc

import (
	"strconv"
	"strings"

	"0chain.net/smartcontract/partitions"
	partitions_v_1 "0chain.net/smartcontract/partitions_v_1"
	partitions_v_2 "0chain.net/smartcontract/partitions_v_2"
	"github.com/0chain/common/core/currency"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
)

//go:generate msgp -io=false -tests=false -unexported=true -v

const blobberRewardsPartitionSize = 5

type BlobberRewardNode struct {
	ID                string        `json:"id"`
	SuccessChallenges int           `json:"success_challenges"`
	WritePrice        currency.Coin `json:"write_price"`
	ReadPrice         currency.Coin `json:"read_price"`
	TotalData         float64       `json:"total_data"`
	DataRead          float64       `json:"data_read"`
}

func (bn *BlobberRewardNode) GetID() string {
	return bn.ID
}

func BlobberRewardKey(round int64) datastore.Key {
	var sb strings.Builder
	sb.WriteString(BLOBBER_REWARD_KEY)
	sb.WriteString(":round:")
	sb.WriteString(strconv.Itoa(int(round)))

	return sb.String()
}

// getActivePassedBlobberRewardsPartitions gets blobbers passed challenge from last challenge period
func getActivePassedBlobberRewardsPartitions(balances c_state.StateContextI, period int64) (res partitions.Partitions, err error) {
	actError := c_state.WithActivation(balances, "apollo", func() error {
		res, err = getActivePassedBlobberRewardsPartitions_v_1(balances, period)
		return nil
	}, func() error {
		res, err = getActivePassedBlobberRewardsPartitions_v_2(balances, period)
		return nil
	})
	if actError != nil {
		return nil, actError
	}

	return
}

// getActivePassedBlobberRewardsPartitions gets blobbers passed challenge from last challenge period
func getActivePassedBlobberRewardsPartitions_v_1(balances c_state.StateContextI, period int64) (partitions.Partitions, error) {
	key := BlobberRewardKey(GetPreviousRewardRound(balances.GetBlock().Round, period))
	return partitions_v_1.CreateIfNotExists(balances, key, blobberRewardsPartitionSize)
}

// getActivePassedBlobberRewardsPartitions gets blobbers passed challenge from last challenge period
func getActivePassedBlobberRewardsPartitions_v_2(balances c_state.StateContextI, period int64) (partitions.Partitions, error) {
	key := BlobberRewardKey(GetPreviousRewardRound(balances.GetBlock().Round, period))
	return partitions_v_2.CreateIfNotExists(balances, key, blobberRewardsPartitionSize)
}

// getOngoingPassedBlobberRewardsPartitions gets blobbers passed challenge from ongoing challenge period
func getOngoingPassedBlobberRewardsPartitions(balances c_state.StateContextI, period int64) (res partitions.Partitions, err error) {
	actError := c_state.WithActivation(balances, "apollo", func() error {
		res, err = getOngoingPassedBlobberRewardsPartitions_v_1(balances, period)
		return nil
	}, func() error {
		res, err = getOngoingPassedBlobberRewardsPartitions_v_2(balances, period)
		return nil
	})
	if actError != nil {
		return nil, actError
	}

	return
}

// getOngoingPassedBlobberRewardsPartitions gets blobbers passed challenge from ongoing challenge period
func getOngoingPassedBlobberRewardsPartitions_v_1(balances c_state.StateContextI, period int64) (partitions.Partitions, error) {
	key := BlobberRewardKey(GetCurrentRewardRound(balances.GetBlock().Round, period))
	return partitions_v_1.CreateIfNotExists(balances, key, blobberRewardsPartitionSize)
}

// getOngoingPassedBlobberRewardsPartitions gets blobbers passed challenge from ongoing challenge period
func getOngoingPassedBlobberRewardsPartitions_v_2(balances c_state.StateContextI, period int64) (partitions.Partitions, error) {
	key := BlobberRewardKey(GetCurrentRewardRound(balances.GetBlock().Round, period))
	return partitions_v_2.CreateIfNotExists(balances, key, blobberRewardsPartitionSize)
}
