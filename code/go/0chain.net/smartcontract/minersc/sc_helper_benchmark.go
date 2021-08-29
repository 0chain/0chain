package minersc

import (
	"strconv"
	"testing"

	"0chain.net/chaincore/state"

	"0chain.net/core/encryption"

	"0chain.net/core/common"
	"0chain.net/smartcontract/benchmark"

	cstate "0chain.net/chaincore/chain/state"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func AddMockMiners(
	nodeType NodeType,
	b *testing.B,
	vi *viper.Viper,
	balances cstate.StateContextI,
) []string {
	var nodes []string
	var allNodes MinerNodes
	var err error
	for i := 0; i < vi.GetInt(benchmark.NumMiners); i++ {
		newNode := NewMinerNode()
		newNode.ID = getMockNodeId(i, nodeType)
		newNode.LastHealthCheck = common.Timestamp(vi.GetInt64(benchmark.Now))
		newNode.PublicKey = "mockPublicKey"
		newNode.ServiceCharge = vi.GetFloat64(benchmark.MinerMaxCharge)
		newNode.NumberOfDelegates = vi.GetInt(benchmark.MinerMaxDelegates)
		newNode.MinStake = state.Balance(vi.GetInt64(benchmark.MinerMinStake))
		newNode.MaxStake = state.Balance(vi.GetInt64(benchmark.MinerMaxStake))
		newNode.NodeType = NodeTypeMiner

		_, err := balances.InsertTrieNode(newNode.getKey(), newNode)
		require.NoError(b, err)

		allNodes.Nodes = append(allNodes.Nodes, newNode)
	}
	if nodeType == NodeTypeMiner {
		_, err = balances.InsertTrieNode(AllMinersKey, &allNodes)
	} else {
		_, err = balances.InsertTrieNode(AllShardersKey, &allNodes)
	}
	require.NoError(b, err)
	return nodes
}

func getMockNodeId(index int, nodeType NodeType) string {
	return encryption.Hash("mock" + nodeType.String() + strconv.Itoa(index))
}
