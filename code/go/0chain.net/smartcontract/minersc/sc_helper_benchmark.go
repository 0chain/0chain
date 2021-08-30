package minersc

import (
	"strconv"

	"0chain.net/chaincore/state"

	"0chain.net/core/encryption"

	"0chain.net/core/common"
	"0chain.net/smartcontract/benchmark"

	cstate "0chain.net/chaincore/chain/state"
	"github.com/spf13/viper"
)

func AddMockNodes(
	nodeType NodeType,
	balances cstate.StateContextI,
) []string {
	var (
		err      error
		nodes    []string
		allNodes MinerNodes
		numNodes int
		key      string
	)

	if nodeType == NodeTypeMiner {
		numNodes = viper.GetInt(benchmark.NumMiners)
		key = AllMinersKey
	} else {
		numNodes = viper.GetInt(benchmark.NumSharders)
		key = AllShardersKey
	}

	for i := 0; i < numNodes; i++ {
		newNode := NewMinerNode()
		newNode.ID = getMockNodeId(i, nodeType)
		newNode.LastHealthCheck = common.Timestamp(viper.GetInt64(benchmark.Now))
		newNode.PublicKey = "mockPublicKey"
		newNode.ServiceCharge = viper.GetFloat64(benchmark.MinerMaxCharge)
		newNode.NumberOfDelegates = viper.GetInt(benchmark.MinerMaxDelegates)
		newNode.MinStake = state.Balance(viper.GetInt64(benchmark.MinerMinStake))
		newNode.MaxStake = state.Balance(viper.GetInt64(benchmark.MinerMaxStake))
		newNode.NodeType = NodeTypeMiner

		_, err := balances.InsertTrieNode(newNode.getKey(), newNode)
		if err != nil {
			panic(err)
		}

		allNodes.Nodes = append(allNodes.Nodes, newNode)
	}

	_, err = balances.InsertTrieNode(key, &allNodes)
	if err != nil {
		panic(err)
	}

	return nodes
}

func getMockNodeId(index int, nodeType NodeType) string {
	return encryption.Hash("mock" + nodeType.String() + strconv.Itoa(index))
}
