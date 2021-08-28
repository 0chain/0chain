package minersc

import (
	"strconv"
	"testing"

	"0chain.net/chaincore/state"

	"0chain.net/core/encryption"

	"0chain.net/core/common"
	"0chain.net/smartcontract"

	cstate "0chain.net/chaincore/chain/state"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func AddMockMiners(
	b *testing.B,
	vi *viper.Viper,
	balances cstate.StateContextI,
) []string {
	var nodes []string
	var allMiners MinerNodes
	for i := 0; i < vi.GetInt(smartcontract.NumMiners); i++ {
		newNode := NewMinerNode()
		newNode.ID = getMockMinerId(i)
		newNode.LastHealthCheck = common.Timestamp(vi.GetInt64(smartcontract.Now))
		newNode.PublicKey = "mockPublicKey"
		newNode.ServiceCharge = vi.GetFloat64(smartcontract.MinerMaxCharge)
		newNode.NumberOfDelegates = vi.GetInt(smartcontract.MinerMaxDelegates)
		newNode.MinStake = state.Balance(vi.GetInt64(smartcontract.MinerMinStake))
		newNode.MaxStake = state.Balance(vi.GetInt64(smartcontract.MinerMaxStake))
		newNode.NodeType = NodeTypeMiner

		_, err := balances.InsertTrieNode(newNode.getKey(), newNode)
		require.NoError(b, err)

		allMiners.Nodes = append(allMiners.Nodes, newNode)
	}
	_, err := balances.InsertTrieNode(AllMinersKey, &allMiners)
	require.NoError(b, err)
	return nodes
}

func getMockMinerId(index int) string {
	return encryption.Hash("mockMiner_" + strconv.Itoa(index))
}
