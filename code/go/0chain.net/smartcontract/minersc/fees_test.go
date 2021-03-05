package minersc

import (
	"0chain.net/chaincore/config"
	"fmt"
	"math/rand"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"

	"github.com/stretchr/testify/require"
)

type TestClient struct {
	client   *Client
	delegate *Client
	stakers  []*Client
}

func createLFMB(miners []*TestClient, sharders []*TestClient) (
	b *block.Block) {

	b = new(block.Block)

	b.MagicBlock = block.NewMagicBlock()
	b.MagicBlock.Miners = node.NewPool(node.NodeTypeMiner)
	b.MagicBlock.Sharders = node.NewPool(node.NodeTypeSharder)

	for _, miner := range miners {
		b.MagicBlock.Miners.NodesMap[miner.client.id] = new(node.Node)
	}
	for _, sharder := range sharders {
		b.MagicBlock.Sharders.NodesMap[sharder.client.id] = new(node.Node)
	}
	return
}

func (msc *MinerSmartContract) setDKGMiners(t *testing.T,
	miners []*TestClient, balances *testBalances) {

	t.Helper()

	var global, err = msc.getGlobalNode(balances)
	require.NoError(t, err)

	var nodes *DKGMinerNodes
	nodes, err = msc.getMinersDKGList(balances)
	require.NoError(t, err)

	nodes.setConfigs(global)
	for _, miner := range miners {
		nodes.SimpleNodes[miner.client.id] = &SimpleNode{ID: miner.client.id}
		nodes.Waited[miner.client.id] = true
	}

	_, err = balances.InsertTrieNode(DKGMinersKey, nodes)
	require.NoError(t, err)
}

func Test_payFees(t *testing.T) {
	const sharderStakeValue, minerStakeValue, generatorStakeValue = 5, 3, 2
	const sharderStakersAmount, minerStakersAmount, generatorStakersAmount = 13, 11, 7
    const minersAmount, shardersAmount = 17, 19
	const generatorIdx = 0

    const timeDelta = 10

	var (
		balances = newTestBalances()
		msc      = newTestMinerSC()
		now      int64
		err      error

		miners   []*TestClient
		sharders []*TestClient
	)

	setConfig(t, balances)

	config.DevConfiguration.IsDkgEnabled = true
	config.DevConfiguration.IsFeeEnabled = true

	t.Run("add miners", func(t *testing.T) {
		var generator = newClientWithStakers(true, t, msc, now,
			generatorStakersAmount, generatorStakeValue, balances)

		for idx := 0; idx < minersAmount; idx++ {
			if idx == generatorIdx {
				miners = append(miners, generator)
			} else {
				miners = append(miners, newClientWithStakers(true, t, msc, now,
					minerStakersAmount, minerStakeValue, balances))
			}
			now += timeDelta
		}
	})

	t.Run("add sharders", func(t *testing.T) {
		for idx := 0; idx < shardersAmount; idx++ {
			sharders = append(sharders, newClientWithStakers(false, t, msc, now,
				sharderStakersAmount, sharderStakeValue, balances))
			now += timeDelta
		}
	})

	//todo: advanced test case: create pool of N stakers and assign them to different nodes randomly,
	//      this way 1 staker might be stake holder of several different miners/sharders at the same time
	//      and more complicated computation is required in order to test such case

	msc.setDKGMiners(t, miners, balances)
	balances.setLFMB(createLFMB(miners, sharders))

	t.Run("stake miners", func(t *testing.T) {
		for idx, miner := range miners {
			var stakeValue int64
			if idx == generatorIdx {
				stakeValue = generatorStakeValue
			} else {
				stakeValue = minerStakeValue
			}

			for _, staker := range miner.stakers {
				_, err = staker.callAddToDelegatePool(t, msc, now,
					stakeValue, miner.client.id, balances)

				require.NoError(t, err, "staking miner")
				now += timeDelta
			}
		}

		msc.assertZeroNodesBalances(t, balances, miners, "miners' balances must be unchanged so far")
		msc.assertZeroStakersBalances(t, balances, miners, "stakers' balances must be unchanged so far")
	})

	t.Run("stake sharders", func(t *testing.T) {
		for _, sharder := range sharders {
			for _, staker := range sharder.stakers {
				_, err = staker.callAddToDelegatePool(t, msc, now,
					sharderStakeValue, sharder.client.id, balances)

				require.NoError(t, err, "staking sharder")
				now += timeDelta
			}
		}

		msc.assertZeroNodesBalances(t, balances, sharders, "sharders' balances must be unchanged so far")
		msc.assertZeroStakersBalances(t, balances, sharders, "stakers' balances must be unchanged so far")
    })

	msc.setDKGMiners(t, miners, balances)

	t.Run("pay fees -> view change", func(t *testing.T) {
		config.DevConfiguration.ViewChange = true

		zeroizeBalances(balances)
		setRounds(t, msc, 250, 251, balances)

		assertPendingPoolsNotEmpty(t, msc, balances)
		assertActivePoolsAreEmpty(t, msc, balances)

		setMagicBlock(t,
			unwrapClients(miners),
			unwrapClients(sharders),
			balances)

		var generator, blck = prepareGeneratorAndBlock(miners, 0, 251)

		// payFees transaction
		now += timeDelta
		var tx = newTransaction(generator.client.id, ADDRESS, 0, now)
		balances.txn = tx
		balances.block = blck
		balances.blockSharders = selectRandom(sharders, 3)

		var global, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "getting global node")

		_, err = msc.payFees(tx, nil, global, balances)
		require.NoError(t, err, "pay_fees error")

		// pools become active, nothing should be paid
		assertActivePoolsNotEmpty(t, msc, balances)

		msc.assertZeroNodesBalances(t, balances, miners, "miners' balances must be unchanged so far")
		msc.assertZeroNodesBalances(t, balances, sharders, "sharders' balances must be unchanged so far")
		msc.assertZeroStakersBalances(t, balances, miners, "stakers' balances must be unchanged so far")
		msc.assertZeroStakersBalances(t, balances, sharders, "stakers' balances must be unchanged so far")

		global, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "can't get global node")
		require.EqualValues(t, 251, global.LastRound)
		require.EqualValues(t, 0, global.Minted)
	})

	msc.setDKGMiners(t, miners, balances)

	t.Run("pay fees -> no fees", func(t *testing.T) {
		zeroizeBalances(balances)
		setRounds(t, msc, 251, 501, balances)

		var generator, blck = prepareGeneratorAndBlock(miners, 0, 252)

		// payFees transaction
		now += timeDelta
		var tx = newTransaction(generator.client.id, ADDRESS, 0, now)
		balances.txn = tx
		balances.block = blck
		balances.blockSharders = selectRandom(sharders, 3)

		var global, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "getting global node")

		_, err = msc.payFees(tx, nil, global, balances)
		require.NoError(t, err, "pay_fees error")

		// pools active, no fees, rewards should be payed for
		// generator's and block sharders' stake holders

		var (
			expected = make(map[string]state.Balance)
			actual   = make(map[string]state.Balance)
		)

		msc.assertZeroNodesBalances(t, balances, miners, "miners' balances must be zero")
		msc.assertZeroNodesBalances(t, balances, sharders, "sharders' balances must be zero")

		for idx, miner := range miners {
			var stakeValue state.Balance = 0;
			if idx == generatorIdx {
				stakeValue = generatorStakeValue;
			} else {
				stakeValue = minerStakeValue;
			}

			for _, staker := range miner.stakers {
				expected[staker.id] = stakeValue;
				actual[staker.id] = balances.balances[staker.id]
			}
		}

		require.Equal(t, len(expected), len(actual), "sizes of balance maps")
		require.Equal(t, expected, actual, "balances")

		for _, sharder := range sharders {
			for _, staker := range sharder.stakers {
				expected[staker.id] = 0 //only block sharders get paid
				actual[staker.id] = balances.balances[staker.id]
			}
		}

		for _, sharder := range filterClientsById(sharders, balances.blockSharders) {
			for _, staker := range sharder.stakers {
				expected[staker.id] += sharderStakeValue
			}
		}

		require.Equal(t, len(expected), len(actual), "sizes of balance maps")
		require.Equal(t, expected, actual, "balances")
	})

	// don't set DKG miners list, because no VC is expected

	t.Run("pay fees -> with fees", func(t *testing.T) {
	 zeroizeBalances(balances)
		setRounds(t, msc, 252, 501, balances)

		var generator, blck = prepareGeneratorAndBlock(miners, 0, 253)

		// payFees transaction
		now += timeDelta
		var tx = newTransaction(generator.client.id, ADDRESS, 0, now)
		balances.txn = tx
		balances.block = blck
		balances.blockSharders = selectRandom(sharders, 3)

		// add fees
		tx.Fee = 100e10
		blck.Txns = append(blck.Txns, tx)

		var global, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "getting global node")

		_, err = msc.payFees(tx, nil, global, balances)
		require.NoError(t, err, "pay_fees error")

		// pools are active, rewards as above and +fees

		var (
			expected = make(map[string]state.Balance)
			actual   = make(map[string]state.Balance)
		)

		msc.assertZeroNodesBalances(t, balances, miners, "miners' balances must be zero")
		msc.assertZeroNodesBalances(t, balances, sharders, "sharders' balances must be zero")

		for _, miner := range miners {
			for _, staker:= range miner.stakers {
				if miner == generator {
					expected[staker.id] += 77e7 + 11e10 // + generator fees
				} else {
					expected[staker.id] += 0
				}
				actual[staker.id] = balances.balances[staker.id]
			}
		}

		for _, sharder := range sharders {
			for _, staker := range sharder.stakers {
				expected[staker.id] += 0
				actual[staker.id] = balances.balances[staker.id]
			}
		}

		for _, sharder := range filterClientsById(sharders, balances.blockSharders) {
			for _, staker := range sharder.stakers {
				expected[staker.id] += 21e7 + 3e10 // + block sharders fees
			}
		}

		require.Equal(t, len(expected), len(actual), "sizes of balance maps")
		require.Equal(t, expected, actual, "balances")
	})

	// don't set DKG miners list, because no VC is expected

	t.Run("pay fees -> view change interests", func(t *testing.T) {
	 zeroizeBalances(balances)
		setRounds(t, msc, 500, 501, balances)

		var generator, blck = prepareGeneratorAndBlock(miners, 0, 501)

		// payFees transaction
		now += timeDelta
		var tx = newTransaction(generator.client.id, ADDRESS, 0, now)
		balances.txn = tx
		balances.block = blck
		balances.blockSharders = selectRandom(sharders, 3)

		// add fees
		var gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "getting global node")

		_, err = msc.payFees(tx, nil, gn, balances)
		require.NoError(t, err, "pay_fees error")

		// pools are active, rewards as above and +fees

		var (
			expected = make(map[string]state.Balance)
			actual   = make(map[string]state.Balance)
		)

		msc.assertZeroNodesBalances(t, balances, miners, "miners' balances must be zero")
		msc.assertZeroNodesBalances(t, balances, sharders, "sharders' balances must be zero")

		for _, miner := range miners {
			for _, staker := range miner.stakers {
				if miner == generator {
					expected[staker.id] += 77e7 + 1e10
				} else {
					expected[staker.id] += 1e10
				}
				actual[staker.id] = balances.balances[staker.id]
			}
		}

		for _, sharder := range sharders {
			for _, staker := range sharder.stakers {
				expected[staker.id] += 1e10
				actual[staker.id] = balances.balances[staker.id]
			}
		}

		for _, sharder := range filterClientsById(sharders, balances.blockSharders) {
			for _, staker := range sharder.stakers {
				expected[staker.id] += 21e7
			}
		}

		require.Equal(t, len(expected), len(actual), "sizes of balance maps")
		require.Equal(t, expected, actual, "balances")
	})

	t.Run("epoch", func(t *testing.T) {
		var global, err = msc.getGlobalNode(balances)
		require.NoError(t, err)

		var interestRate, rewardRate = global.InterestRate, global.RewardRate
		global.epochDecline()

		require.True(t, global.InterestRate < interestRate)
		require.True(t, global.RewardRate < rewardRate)
	})
}

func prepareGeneratorAndBlock(miners []*TestClient, idx int, round int64) (
	generator *TestClient, blck *block.Block) {

	//todo: that's weird
	generator = miners[idx]

	blck = block.Provider().(*block.Block)
	blck.Round = round                                // VC round
	blck.MinerID = generator.client.id                // block generator
	blck.PrevBlock = block.Provider().(*block.Block)  // stub

	return generator, blck
}

func selectRandom(clients []*TestClient, n int) (selection []string) {
	if n > len(clients) {
		panic("too many elements requested")
	}

	selection = make([]string, 0, n)

	var permutations = rand.Perm(len(clients))
	for i := 0; i < n; i++ {
		selection = append(selection, clients[permutations[i]].client.id)
	}
	return
}

func filterClientsById(clients []*TestClient, ids []string) (
	selection []*TestClient) {

	selection = make([]*TestClient, 0, len(ids))

	for _, client := range clients {
		for _, id := range ids {
			if client.client.id == id {
				selection = append(selection, client)
			}
		}
	}
	return
}

func zeroizeBalances(balances *testBalances) {
	balances.balances = make(map[string]state.Balance)
}

func assertBalancesAreZeros(t *testing.T, balances *testBalances) {
	for id, value := range balances.balances {
		if id == ADDRESS {
			continue
		}
		require.Zerof(t, value, "%s has non-zero balance: %d", id, value)
	}
}
//todo: "assert" -> "require"
func assertActivePoolsAreEmpty(t *testing.T, msc *MinerSmartContract,
	balances *testBalances) {
		assertPools(t, msc, balances, true, true)
}

func assertActivePoolsNotEmpty(t *testing.T, msc *MinerSmartContract,
	balances *testBalances) {
		assertPools(t, msc, balances, true, false)
}

func assertPendingPoolsAreEmpty(t *testing.T, msc *MinerSmartContract,
	balances *testBalances) {
		assertPools(t, msc, balances, false, true)
}

func assertPendingPoolsNotEmpty(t *testing.T, msc *MinerSmartContract,
	balances *testBalances) {
	assertPools(t, msc, balances, false, false)
}

func assertPools(t *testing.T, msc *MinerSmartContract,
	balances *testBalances, activeNotPending bool, areEmpty bool) {

	var simple *ConsensusNodes
	var miners, sharders []*ConsensusNode
	simple,   _ = msc.getMinersList(balances)
	miners,   _ = msc.readPools(simple, balances)

	simple,   _ = msc.getShardersList(balances, AllShardersKey)
	sharders, _ = msc.readPools(simple, balances)

	for _, node := range append(miners, sharders...) {
		if activeNotPending {
			if areEmpty {
				require.False(t, len(node.Active) > 0, "active pools must be empty")
			} else {
				require.True(t, len(node.Active) > 0, "active pools must be non-empty")
			}
		} else {
			if areEmpty {
				require.False(t, len(node.Pending) > 0, "pending pools must be empty")
			} else {
				require.True(t, len(node.Pending) > 0, "pending pools must be non-empty")
			}
		}
	}
}

func (msc *MinerSmartContract) assertZeroBalances(t *testing.T,
	balances *testBalances, clients []*Client,
	message string) {

	for _, client := range clients {
		require.Zero(t, balances.balances[client.id], message)
	}
}

func (msc *MinerSmartContract) assertZeroNodesBalances(t *testing.T,
	balances *testBalances, nodes []*TestClient,
	message string) {

	msc.assertZeroBalances(t, balances, unwrapClients(nodes), message + " (client wallets)")
	msc.assertZeroBalances(t, balances, unwrapDelegates(nodes), message + " (delegate wallets)")
}

func (msc *MinerSmartContract) assertZeroStakersBalances(t *testing.T,
	balances *testBalances, nodes []*TestClient,
	message string) {

	for _, node := range nodes {
		msc.assertZeroBalances(t, balances, node.stakers, message)
	}
}

func unwrapClients(clients []*TestClient) (list []*Client) {
	list = make([]*Client, 0, len(clients))
	for _, miner := range clients {
		list = append(list, miner.client)
	}
	return
}

func unwrapDelegates(clients []*TestClient) (list []*Client) {
	list = make([]*Client, 0, len(clients))
	for _, node := range clients {
		list = append(list, node.delegate)
	}
	return list
}

func (msc *MinerSmartContract) debug_pools(state *testBalances) {
	var simple *ConsensusNodes
	var miners, sharders []*ConsensusNode
	var err error

	if simple, err = msc.getMinersList(state); err == nil {
		if miners, err = msc.readPools(simple, state); err == nil {
			for _, miner := range miners {
				fmt.Printf("\t=-- miner %s: %d active pools, %d pending pools\n",
					miner.ID, len(miner.Active), len(miner.Pending))
			}
		} else {
			fmt.Printf("\t--- can't retrieve pools: %v\n", err)
		}
	} else {
		fmt.Printf("\t>-- couldn't retrieve miners: %v\n", err)
	}

	if simple, err = msc.getShardersList(state, AllShardersKey); err == nil {
		if sharders, err = msc.readPools(simple, state); err == nil {
			for _, sharder := range sharders {
				fmt.Printf("\t=-- sharder %s: %d active pools, %d pending pools\n",
					sharder.ID, len(sharder.Active), len(sharder.Pending))
			}
		} else {
			fmt.Printf("\t--- can't retrieve pools: %v\n", err)
		}
	} else {
		fmt.Printf("\t>-- couldn't retrieve sharders: %v\n", err)
	}
}
