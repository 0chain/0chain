package storagesc

import (
	"strconv"
	"strings"

	"0chain.net/smartcontract/provider"

	"0chain.net/chaincore/currency"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/partitions"
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

func passedCallback(id string, data []byte, toPartition, _ int, sCtx c_state.StateContextI) error {
	replace := &StorageNode{
		Provider: &provider.Provider{
			ID: id,
		},
	}
	if err := sCtx.GetTrieNode(replace.GetKey(ADDRESS), replace); err != nil {
		return err
	}
	replace.LastRewardPartition.Index = toPartition
	if _, err := sCtx.InsertTrieNode(replace.GetKey(ADDRESS), replace); err != nil {
		return err
	}
	return nil
}

// getActivePassedBlobberRewardsPartitions gets blobbers passed challenge from last challenge period
func getActivePassedBlobberRewardsPartitions(balances c_state.StateContextI, period int64) (*partitions.Partitions, error) {
	name := BlobberRewardKey(GetPreviousRewardRound(balances.GetBlock().Round, period))
	blobbers, err := partitions.CreateIfNotExists(balances, name, blobberRewardsPartitionSize)
	if err != nil {
		return nil, err
	}
	blobbers.SetCallback(passedCallback)
	return blobbers, nil
}

func ongoingCallback(id string, data []byte, toPartition, _ int, sCtx c_state.StateContextI) error {
	replace := &StorageNode{
		Provider: &provider.Provider{
			ID: id,
		},
	}
	if err := sCtx.GetTrieNode(replace.GetKey(ADDRESS), replace); err != nil {
		return err
	}
	replace.RewardPartition.Index = toPartition
	if _, err := sCtx.InsertTrieNode(replace.GetKey(ADDRESS), replace); err != nil {
		return err
	}
	return nil
}

// getOngoingPassedBlobberRewardsPartitions gets blobbers passed challenge from ongoing challenge period
func getOngoingPassedBlobberRewardsPartitions(balances c_state.StateContextI, period int64) (*partitions.Partitions, error) {
	name := BlobberRewardKey(GetCurrentRewardRound(balances.GetBlock().Round, period))
	blobbers, err := partitions.CreateIfNotExists(balances, name, blobberRewardsPartitionSize)
	if err != nil {
		return nil, err
	}
	blobbers.SetCallback(ongoingCallback)
	return blobbers, nil
}
