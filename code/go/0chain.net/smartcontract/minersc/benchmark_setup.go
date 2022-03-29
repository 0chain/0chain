package minersc

import (
	"strconv"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/benchmark"
	"github.com/rcrowley/go-metrics"
	"github.com/spf13/viper"
)

func AddMockNodes(
	clients []string,
	nodeType NodeType,
	balances cstate.StateContextI,
) []string {
	var (
		err          error
		nodes        []string
		allNodes     MinerNodes
		numActive    int
		nodeMap      = make(map[string]*SimpleNode)
		numNodes     int
		numDelegates int
		key          string
	)

	if nodeType == NodeTypeMiner {
		numActive = viper.GetInt(benchmark.NumActiveMiners)
		numNodes = viper.GetInt(benchmark.NumMiners)
		numDelegates = viper.GetInt(benchmark.NumMinerDelegates)
		key = AllMinersKey
	} else {
		numActive = viper.GetInt(benchmark.NumActiveSharders)
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
			dId := (i + j) % numNodes
			pool := sci.DelegatePool{
				ZcnLockingPool: &tokenpool.ZcnLockingPool{
					ZcnPool: tokenpool.ZcnPool{
						TokenPool: tokenpool.TokenPool{
							ID:      getMinerDelegatePoolId(i, dId, nodeType),
							Balance: 100 * 1e10,
						},
					},
				},
				PoolStats: &sci.PoolStats{},
			}

			pool.DelegateID = clients[dId]
			if i < numActive {
				newNode.Active[getMinerDelegatePoolId(i, dId, nodeType)] = &pool
			} else {
				newNode.Pending[getMinerDelegatePoolId(i, dId, nodeType)] = &pool
			}
		}
		_, err := balances.InsertTrieNode(newNode.GetKey(), newNode)
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
		if err != nil {
			panic(err)
		}

		mpks := block.NewMpks()
		for key := range nodeMap {
			mpks.Mpks[key] = &block.MPK{
				ID:  key,
				Mpk: nodes,
			}

		}
		_, err = balances.InsertTrieNode(MinersMPKKey, mpks)
		if err != nil {
			panic(err)
		}
	} else {
		_, err = balances.InsertTrieNode(ShardersKeepKey, &MinerNodes{
			Nodes: allNodes.Nodes[1:],
		})
		if err != nil {
			panic(err)
		}
	}
	_, err = balances.InsertTrieNode(key, &allNodes)
	if err != nil {
		panic(err)
	}
	return nodes
}

func AddNodeDelegates(
	clients, miners, sharders []string,
	balances cstate.StateContextI,
) {
	var cns = make(map[string]UserNode)
	for i := range miners {
		AddUserNodesForNode(i, NodeTypeMiner, miners, clients, cns)
	}
	for i := range sharders {
		AddUserNodesForNode(i, NodeTypeSharder, sharders, clients, cns)
	}
	for _, un := range cns {
		_, _ = balances.InsertTrieNode(un.GetKey(), &un)
	}
}

func AddUserNodesForNode(
	nodeIndex int,
	nodeType NodeType,
	nodes []string,
	clients []string, cns map[string]UserNode,
) {
	var numDelegates = viper.GetInt(benchmark.NumSharderDelegates)
	for j := 0; j < numDelegates; j++ {
		delegate := (nodeIndex + j) % len(nodes)
		var un UserNode
		un, ok := cns[clients[delegate]]
		if !ok {
			un = UserNode{
				ID:    clients[delegate],
				Pools: make(map[datastore.Key][]datastore.Key),
			}
		}
		un.Pools[nodes[nodeIndex]] = append(un.Pools[nodes[nodeIndex]],
			getMinerDelegatePoolId(nodeIndex, delegate, nodeType))
		cns[clients[delegate]] = un
	}
}

func SetUpNodes(
	miners, sharders []string,
) {
	activeMiners := viper.GetInt(benchmark.NumActiveMiners)
	for i, miner := range miners {
		nextMiner := &node.Node{}
		nextMiner.TimersByURI = make(map[string]metrics.Timer, 10)
		nextMiner.SizeByURI = make(map[string]metrics.Histogram, 10)
		// if necessary we coule create a real (id, public key, private key)
		// triplet here, but we would need to provide it to the tests as
		// they would change each run. No test seems to need this so leaving it out.
		nextMiner.ID = miner
		nextMiner.PublicKey = "mockPublicKey"
		nextMiner.Type = node.NodeTypeMiner
		if i < activeMiners {
			nextMiner.Status = node.NodeStatusActive
		} else {
			nextMiner.Status = node.NodeStatusInactive
		}
		node.RegisterNode(nextMiner)
	}
	activeSharders := viper.GetInt(benchmark.NumActiveSharders)
	for i, sharder := range sharders {
		nextSharder := &node.Node{}
		nextSharder.TimersByURI = make(map[string]metrics.Timer, 10)
		nextSharder.SizeByURI = make(map[string]metrics.Histogram, 10)
		nextSharder.ID = sharder
		nextSharder.PublicKey = "mockPublicKey"
		nextSharder.Type = node.NodeTypeMiner
		if i < activeSharders {
			nextSharder.Status = node.NodeStatusActive
		} else {
			nextSharder.Status = node.NodeStatusInactive
		}
		node.RegisterNode(nextSharder)
	}
}

func AddMagicBlock(
	miners, sharders []string,
	balances cstate.StateContextI,
) {
	var magicBlock block.MagicBlock
	_, _ = balances.InsertTrieNode(MagicBlockKey, &magicBlock)

	var gsos = block.NewGroupSharesOrSigns()
	_, _ = balances.InsertTrieNode(GroupShareOrSignsKey, gsos)
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

func getMinerDelegatePoolId(miner, delegate int, nodeType NodeType) string {
	return encryption.Hash("delegate pool" +
		strconv.Itoa(miner) + strconv.Itoa(delegate) + strconv.Itoa(int(nodeType)))
}

func GetMockNodeId(index int, nodeType NodeType) string {
	return encryption.Hash("mock" + nodeType.String() + strconv.Itoa(index))
}
