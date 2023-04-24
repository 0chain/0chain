package minersc

import (
	"strconv"

	"0chain.net/smartcontract/benchmark/main/cmd/log"

	"github.com/0chain/common/core/currency"

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

var mockRewardAmount currency.Coin = 1680000000
var mockRewardType = spenum.BlockRewardMiner

func AddMockGlobalNode(balances cstate.StateContextI) {
	var gn GlobalNode
	gn.readConfig()
	_, err := balances.InsertTrieNode(GlobalNodeKey, &gn)
	if err != nil {
		log.Fatal(err)
	}
}

func AddMockNodes(
	clients []string,
	nodeType spenum.Provider,
	eventDb *event.EventDb,
	balances cstate.StateContextI,
	getIdAndPublicKey func() (string, string, error),
) ([]string, []string) {
	const (
		delegateReward      = 0.3 * 1e10
		providerReward      = 0.1 * 1e10
		delegatePoolBalance = 100 * 1e10
	)
	var (
		err                error
		nodes, publickKeys []string
		allNodes           NodeIDs
		nodeMap            = make(map[string]*SimpleNode)
		numNodes           int
		numActive          int
		numDelegates       int
		key                string
		dRewards           []event.RewardDelegate
		dps                []event.DelegatePool
		providerType       spenum.Provider
	)

	if nodeType == spenum.Miner {
		numActive = viper.GetInt(benchmark.NumActiveMiners)
		numNodes = viper.GetInt(benchmark.NumMiners)
		numDelegates = viper.GetInt(benchmark.NumMinerDelegates)
		key = AllMinersKey
		providerType = spenum.Miner
	} else {
		numActive = viper.GetInt(benchmark.NumActiveSharders)
		numNodes = viper.GetInt(benchmark.NumSharders)
		numDelegates = viper.GetInt(benchmark.NumSharderDelegates)
		key = AllShardersKey
		providerType = spenum.Sharder
	}

	for i := 0; i < numNodes; i++ {
		newNode := NewMinerNode()
		newNode.ID, newNode.PublicKey, err = getIdAndPublicKey()
		if err != nil {
			log.Fatal(err)
		}
		newNode.ProviderType = providerType
		newNode.LastHealthCheck = common.Timestamp(viper.GetInt64(benchmark.MptCreationTime))
		newNode.Settings.ServiceChargeRatio = viper.GetFloat64(benchmark.MinerMaxCharge)
		newNode.Settings.MaxNumDelegates = viper.GetInt(benchmark.MinerMaxDelegates)
		newNode.NodeType = NodeTypeMiner
		newNode.Settings.DelegateWallet = clients[0]
		newNode.Reward = providerReward
		publickKeys = append(publickKeys, newNode.PublicKey)
		for j := 0; j < numDelegates; j++ {
			dId := (i + j) % numNodes
			poolId := getMinerDelegatePoolId(i, dId, nodeType)
			pool := stakepool.DelegatePool{
				Balance:      delegatePoolBalance,
				Reward:       delegateReward,
				DelegateID:   poolId,
				RoundCreated: 1,
				Status:       spenum.Active,
			}
			if i < numActive {
				pool.Status = spenum.Active
			} else {
				pool.Status = spenum.Pending
			}
			newNode.Pools[poolId] = &pool
			if eventDb.Debug() {
				for bk := int64(1); bk <= viper.GetInt64(benchmark.NumBlocks); bk++ {
					dRewards = append(dRewards, event.RewardDelegate{
						Amount:      mockRewardAmount,
						BlockNumber: bk,
						PoolID:      poolId,
						RewardType:  mockRewardType,
					})
				}
			}
		}
		_, err = balances.InsertTrieNode(newNode.GetKey(), newNode)
		if err != nil {
			panic(err)
		}
		nodes = append(nodes, newNode.ID)
		nodeMap[newNode.ID] = newNode.SimpleNode
		allNodes = append(allNodes, newNode.ID)

		if viper.GetBool(benchmark.EventDbEnabled) {
			if nodeType == spenum.Miner {
				minerDb := event.Miner{

					PublicKey: newNode.PublicKey,
					Provider: event.Provider{
						ID:            newNode.ID,
						ServiceCharge: newNode.Settings.ServiceChargeRatio,
						NumDelegates:  newNode.Settings.MaxNumDelegates,
						Rewards: event.ProviderRewards{
							ProviderID:                    newNode.ID,
							RoundServiceChargeLastUpdated: 7,
						},
						LastHealthCheck: newNode.LastHealthCheck,
					},
				}
				if err = eventDb.Store.Get().Create(&minerDb).Error; err != nil {
					log.Fatal(err)
				}
			} else {
				sharderDb := event.Sharder{
					PublicKey: newNode.PublicKey,
					Provider: event.Provider{
						LastHealthCheck: newNode.LastHealthCheck,
						ID:              newNode.ID,
						ServiceCharge:   newNode.Settings.ServiceChargeRatio,
						NumDelegates:    newNode.Settings.MaxNumDelegates,
						Rewards: event.ProviderRewards{
							ProviderID:                    newNode.ID,
							RoundServiceChargeLastUpdated: 11,
						},
					},
				}
				if err := eventDb.Store.Get().Create(&sharderDb).Error; err != nil {
					log.Fatal(err)
				}
			}
			orderedPoolIds := newNode.OrderedPoolIds()
			for _, id := range orderedPoolIds {
				pool := newNode.Pools[id]
				dps = append(dps, event.DelegatePool{
					PoolID:               id,
					ProviderType:         nodeType,
					ProviderID:           newNode.ID,
					DelegateID:           pool.DelegateID,
					Balance:              pool.Balance,
					Reward:               pool.Reward,
					TotalReward:          pool.Reward,
					Status:               pool.Status,
					RoundCreated:         pool.RoundCreated,
					RoundPoolLastUpdated: viper.GetInt64(benchmark.NumBlocks),
					StakedAt:             pool.StakedAt,
				})
			}
		}
	}
	if eventDb.Debug() {
		if err := eventDb.Store.Get().Create(&dRewards).Error; err != nil {
			log.Fatal(err)
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
		allIDs := allNodes[1:]
		_, err = balances.InsertTrieNode(ShardersKeepKey, &allIDs)
		if err != nil {
			panic(err)
		}
	}
	_, err = balances.InsertTrieNode(key, &allNodes)
	if err != nil {
		panic(err)
	}

	if viper.GetBool(benchmark.EventDbEnabled) {
		if err := eventDb.Store.Get().Create(&dps).Error; err != nil {
			log.Fatal(err)
		}
	}

	return nodes, publickKeys
}

func SetUpNodes(
	miners, sharders, sharderKeys []string,
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
		nextSharder.PublicKey = sharderKeys[i]
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
	_, _ []string,
	balances cstate.StateContextI,
) {
	var magicBlock block.MagicBlock
	_, _ = balances.InsertTrieNode(MagicBlockKey, &magicBlock)

	var gsos = block.NewGroupSharesOrSigns()
	_, _ = balances.InsertTrieNode(GroupShareOrSignsKey, gsos)
}

func AddMockProviderRewards(
	miners, sharders []string,
	eventDb *event.EventDb,
) {
	if eventDb.Debug() {
		return
	}
	var pRewards []event.RewardProvider
	for _, miner := range miners {
		for bk := int64(1); bk <= viper.GetInt64(benchmark.NumBlocks); bk++ {
			pRewards = append(pRewards, event.RewardProvider{
				Amount:      mockRewardAmount,
				BlockNumber: bk,
				ProviderId:  miner,
				RewardType:  mockRewardType,
			})
		}
	}
	for _, sharder := range sharders {
		for bk := int64(1); bk <= viper.GetInt64(benchmark.NumBlocks); bk++ {
			pRewards = append(pRewards, event.RewardProvider{
				Amount:      mockRewardAmount,
				BlockNumber: bk,
				ProviderId:  sharder,
				RewardType:  mockRewardType,
			})
		}
	}
	if err := eventDb.Store.Get().Create(&pRewards).Error; err != nil {
		log.Fatal(err)
	}
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
