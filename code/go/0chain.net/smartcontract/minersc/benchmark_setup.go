package minersc

import (
	"fmt"
	"math/big"
	"strconv"

	"0chain.net/smartcontract/zbig"

	"0chain.net/smartcontract/benchmark/main/cmd/log"

	"0chain.net/chaincore/currency"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/benchmark"
	"github.com/rcrowley/go-metrics"
	"github.com/spf13/viper"
)

func AddMockNodes(
	clients []string,
	nodeType spenum.Provider,
	eventDb *event.EventDb,
	balances cstate.StateContextI,
) []string {
	var (
		err          error
		nodes        []string
		allNodes     MinerNodes
		nodeMap      = make(map[string]*SimpleNode)
		numNodes     int
		numActive    int
		numDelegates int
		key          string
	)

	if nodeType == spenum.Miner {
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
	lat := zbig.BigRat{big.NewRat(103, 2)}
	long := zbig.BigRat{big.NewRat(-1, 10)}
	for i := 0; i < numNodes; i++ {
		newNode := NewMinerNode()
		newNode.ID = GetMockNodeId(i, nodeType)
		newNode.LastHealthCheck = common.Timestamp(viper.GetInt64(benchmark.MptCreationTime))
		newNode.PublicKey = "mockPublicKey"
		newNode.Settings.ServiceChargeRatio = *zbig.BigRatFromFloat64(viper.GetFloat64(benchmark.MinerMaxCharge))
		newNode.Settings.MaxNumDelegates = viper.GetInt(benchmark.MinerMaxDelegates)
		newNode.Settings.MinStake = currency.Coin(viper.GetInt64(benchmark.MinerMinStake))
		newNode.Settings.MaxStake = currency.Coin(viper.GetFloat64(benchmark.MinerMaxStake) * 1e10)
		newNode.NodeType = NodeTypeMiner
		newNode.Settings.DelegateWallet = clients[0]
		newNode.Geolocation.Latitude = lat
		newNode.Geolocation.Longitude = long

		for j := 0; j < numDelegates; j++ {
			dId := (i + j) % numNodes
			pool := stakepool.DelegatePool{
				Balance:    100 * 1e10,
				Reward:     0.3 * 1e10,
				DelegateID: clients[dId],
			}
			if i < numActive {
				pool.Status = spenum.Active
				newNode.Pools[getMinerDelegatePoolId(i, dId, nodeType)] = &pool
			} else {
				pool.Status = spenum.Pending
				newNode.Pools[getMinerDelegatePoolId(i, dId, nodeType)] = &pool
			}
		}
		_, err := balances.InsertTrieNode(newNode.GetKey(), newNode)
		if err != nil {
			log.Fatal(err)
		}

		nodes = append(nodes, newNode.ID)
		nodeMap[newNode.ID] = newNode.SimpleNode
		allNodes.Nodes = append(allNodes.Nodes, newNode)

		if viper.GetBool(benchmark.EventDbEnabled) {
			if nodeType == spenum.Miner {
				minerDb := event.Miner{
					MinerID:           newNode.ID,
					LastHealthCheck:   newNode.LastHealthCheck,
					PublicKey:         newNode.PublicKey,
					ServiceCharge:     newNode.Settings.ServiceChargeRatio,
					NumberOfDelegates: newNode.Settings.MaxNumDelegates,
					MinStake:          newNode.Settings.MinStake,
					MaxStake:          newNode.Settings.MaxStake,
					Latitude:          lat,
					Longitude:         long,
				}
				result := eventDb.Store.Get().Create(&minerDb)
				fmt.Println("result put  in miner", result)
			} else {
				sharderDb := event.Sharder{
					SharderID:         newNode.ID,
					LastHealthCheck:   newNode.LastHealthCheck,
					PublicKey:         newNode.PublicKey,
					ServiceCharge:     newNode.Settings.ServiceChargeRatio,
					NumberOfDelegates: newNode.Settings.MaxNumDelegates,
					MinStake:          newNode.Settings.MinStake,
					MaxStake:          newNode.Settings.MaxStake,
					Latitude:          lat,
					Longitude:         long,
				}
				_ = eventDb.Store.Get().Create(&sharderDb)
			}
		}
	}
	if nodeType == spenum.Miner {
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
	for i := range miners {
		AddUserNodesForNode(i, spenum.Miner, miners, clients, balances)
	}
	for i := range sharders {
		AddUserNodesForNode(i, spenum.Sharder, sharders, clients, balances)
	}
}

func AddUserNodesForNode(
	nodeIndex int,
	nodeType spenum.Provider,
	nodes []string,
	clients []string,
	balances cstate.StateContextI,
) {
	var numDelegates = viper.GetInt(benchmark.NumSharderDelegates)
	for j := 0; j < numDelegates; j++ {
		delegate := (nodeIndex + j) % len(nodes)
		un := stakepool.NewUserStakePools()
		un.Providers = []string{nodes[nodeIndex]}

		_, _ = balances.InsertTrieNode(stakepool.UserStakePoolsKey(nodeType, clients[delegate]), un)
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

func getMinerDelegatePoolId(miner, delegate int, nodeType spenum.Provider) string {
	return encryption.Hash("delegate pool" +
		strconv.Itoa(miner) + strconv.Itoa(delegate) + strconv.Itoa(int(nodeType)))
}

func GetMockNodeId(index int, nodeType spenum.Provider) string {
	return encryption.Hash("mock" + nodeType.String() + strconv.Itoa(index))
}
