package minersc

import (
	"strconv"

	"0chain.net/chaincore/block"

	"0chain.net/chaincore/tokenpool"

	sci "0chain.net/chaincore/smartcontractinterface"

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
		err          error
		nodes        []string
		allNodes     MinerNodes
		nodeMap      = make(map[string]*SimpleNode)
		numNodes     int
		numDelegates int
		key          string
	)

	if nodeType == NodeTypeMiner {
		numNodes = viper.GetInt(benchmark.NumMiners)
		numDelegates = viper.GetInt(benchmark.NumMinerDelegates)
		key = AllMinersKey
	} else {
		numNodes = viper.GetInt(benchmark.NumSharders)
		numDelegates = viper.GetInt(benchmark.NumSharderDelegates)
		key = AllShardersKey
	}

	for i := 0; i < numNodes; i++ {
		newNode := NewMinerNode()
		newNode.ID = GetMockNodeId(i, nodeType)
		newNode.LastHealthCheck = common.Timestamp(viper.GetInt64(benchmark.Now))
		newNode.PublicKey = "mockPublicKey"
		newNode.ServiceCharge = viper.GetFloat64(benchmark.MinerMaxCharge)
		newNode.NumberOfDelegates = viper.GetInt(benchmark.MinerMaxDelegates)
		newNode.MinStake = state.Balance(viper.GetInt64(benchmark.MinerMinStake))
		newNode.MaxStake = state.Balance(viper.GetFloat64(benchmark.MinerMaxStake) * 1e10)
		newNode.NodeType = NodeTypeMiner
		newNode.DelegateWallet = newNode.ID

		for j := 0; j < numDelegates; j++ {
			pool := sci.DelegatePool{
				ZcnLockingPool: &tokenpool.ZcnLockingPool{
					ZcnPool: tokenpool.ZcnPool{
						TokenPool: tokenpool.TokenPool{
							ID:      getMockDelegateId(i, j),
							Balance: 100 * 1e10,
						},
					},
				},
				PoolStats: &sci.PoolStats{},
			}
			pool.DelegateID = newNode.ID
			newNode.Active[pool.ID] = &pool
		}

		_, err := balances.InsertTrieNode(newNode.getKey(), newNode)
		if err != nil {
			panic(err)
		}
		nodes = append(nodes, newNode.ID)
		nodeMap[newNode.ID] = newNode.SimpleNode
		allNodes.Nodes = append(allNodes.Nodes, newNode)

	}
	if nodeType == NodeTypeMiner {
		dkgMiners := NewDKGMinerNodes()
		dkgMiners.SimpleNodes = nodeMap
		dkgMiners.T = viper.GetInt(benchmark.InternalT)
		_, err = balances.InsertTrieNode(DKGMinersKey, dkgMiners)

		mpks := block.NewMpks()
		for key := range nodeMap {
			mpks.Mpks[key] = &block.MPK{
				ID:  key,
				Mpk: nodes,
			}

		}
		_, err = balances.InsertTrieNode(MinersMPKKey, mpks)
	} else {
		_, err = balances.InsertTrieNode(ShardersKeepKey, &MinerNodes{
			Nodes: allNodes.Nodes[1:],
		})
	}

	_, err = balances.InsertTrieNode(key, &allNodes)
	if err != nil {
		panic(err)
	}
	return nodes
}

func AddPhaseNode(balances cstate.StateContextI) {
	var pn = PhaseNode{
		Phase:        Contribute,
		StartRound:   1,
		CurrentRound: 2,
		Restarts:     0,
	}
	_, err := balances.InsertTrieNode(pn.GetKey(), &pn)
	if err != nil {
		panic(err)
	}
}

func GetMockNodeId(index int, nodeType NodeType) string {
	return encryption.Hash("mock" + nodeType.String() + strconv.Itoa(index))
}

func getMockDelegateId(miner, delegate int) string {
	return "node_id_" + strconv.Itoa(miner) + "_" + strconv.Itoa(delegate)
}
