package storagesc

import (
	"strconv"
	"strings"

	c_state "0chain.net/chaincore/chain/state"
	cstate "0chain.net/chaincore/state"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/partitions"
)

//go:generate msgp -io=false -tests=false -unexported=true -v

const blobberRewardsPartitionSize = 5

type BlobberRewardNode struct {
	ID                string         `json:"id"`
	SuccessChallenges int            `json:"success_challenges"`
	WritePrice        cstate.Balance `json:"write_price"`
	ReadPrice         cstate.Balance `json:"read_price"`
	TotalData         float64        `json:"total_data"`
	DataRead          float64        `json:"data_read"`
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
func getActivePassedBlobberRewardsPartitions(balances c_state.StateContextI, period int64) (*partitions.Partitions, error) {
	name := BlobberRewardKey(GetPreviousRewardRound(balances.GetBlock().Round, period))
	return partitions.CreateIfNotExists(balances, name, blobberRewardsPartitionSize)
}

// getOngoingPassedBlobberRewardsPartitions gets blobbers passed challenge from ongoing challenge period
func getOngoingPassedBlobberRewardsPartitions(balances c_state.StateContextI, period int64) (*partitions.Partitions, error) {
	name := BlobberRewardKey(GetCurrentRewardRound(balances.GetBlock().Round, period))
	return partitions.CreateIfNotExists(balances, name, blobberRewardsPartitionSize)
}
