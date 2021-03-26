package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/core/datastore"
	"math/rand"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/state"

	"github.com/stretchr/testify/require"
)

//prime numbers
const BlockReward, TransactionFee = 7723, 7919
const GeneratorStakeValue, MinerStakeValue, SharderStakeValue = 2, 3, 5
const GeneratorStakersAmount, MinerStakersAmount, SharderStakersAmount = 11, 13, 17
const MinersAmount, ShardersAmount = 19, 23
const BlockShardersAmount = 7

const timeDelta = 10

func Test_payFees(t *testing.T) {
	var (
		balances = newTestBalances()
		msc      = newTestMinerSC()
		now      int64

		miners   []*TestClient
		sharders []*TestClient
		generator  *TestClient
	)

	{
		var global GlobalNode

		global.ViewChange = 0
		global.MaxN = 100
		global.MinN = 3
		global.MaxS = 30
		global.MinS = 1
		global.MaxDelegates = 100
		global.TPercent = 0.51
		global.KPercent = 0.75
		global.LastRound = 0
		global.MaxStake = state.Balance(100.0e10)
		global.MinStake = state.Balance(1)
		global.InterestRate = 0.1
		global.RewardRate = 1.0
		global.ShareRatio = 0.10
		global.BlockReward = BlockReward
		global.MaxCharge = 0.5 // %
		global.Epoch = 15e6    // 15M
		global.RewardDeclineRate = 0.1
		global.RewardRoundPeriod = 250
		global.InterestDeclineRate = 0.1
		global.MaxMint = state.Balance(4e6 * 1e10)
		global.Minted = 0

		mustSave(t, GlobalNodeKey, &global, balances)
	}

	config.DevConfiguration.ViewChange = true
	config.DevConfiguration.IsDkgEnabled = true
	config.DevConfiguration.IsFeeEnabled = true

	t.Run("create miners", func(t *testing.T) {
		generator = newClientWithStakers(true, t, msc, now,
			GeneratorStakersAmount, GeneratorStakeValue, balances)

		var generatorIdx = rand.Intn(MinersAmount)

		for i := 0; i < MinersAmount; i++ {
			if i == generatorIdx {
				miners = append(miners, generator)
			} else {
				miners = append(miners, newClientWithStakers(true, t, msc, now,
					MinerStakersAmount, MinerStakeValue, balances))
			}
			now += timeDelta
		}
	})

	t.Run("create sharders", func(t *testing.T) {
		for i := 0; i < ShardersAmount; i++ {
			sharders = append(sharders, newClientWithStakers(false, t,
				msc, now, SharderStakersAmount, SharderStakeValue, balances))
			now += timeDelta
		}
	})

	//todo: advanced test case: create pool of N stakers and assign them to
	//		different nodes randomly, this way 1 staker might be stake holder
	//		of several different miners/sharders at the same time and more
	//		complicated computation is required in order to test such case

	msc.setDKGMiners(t, miners, balances)
	balances.setLFMB(createLFMB(miners, sharders))


	t.Run("stake miners", func(t *testing.T) {
		for _, miner := range miners {
			var stakeValue int64
			if miner == generator {
				stakeValue = GeneratorStakeValue
			} else {
				stakeValue = MinerStakeValue
			}

			for _, staker := range miner.stakers {
				var _, err = staker.callAddToDelegatePool(t, msc, now,
					stakeValue, miner.client.id, balances)

				require.NoError(t, err, "staking miner")
				now += timeDelta
			}
		}

		balances.requireNodesHaveZeros(t, miners,
			"miners' balances must be unchanged so far")
		balances.requireStakersHaveZeros(t, miners,
			"stakers' balances must be unchanged so far")
	})

	t.Run("stake sharders", func(t *testing.T) {
		for _, sharder := range sharders {
			for _, staker := range sharder.stakers {
				var _, err = staker.callAddToDelegatePool(t, msc, now,
					SharderStakeValue, sharder.client.id, balances)

				require.NoError(t, err, "staking sharder")
				now += timeDelta
			}
		}

		balances.requireNodesHaveZeros(t, sharders,
			"sharders' balances must be unchanged so far")
		balances.requireStakersHaveZeros(t, sharders,
			"stakers' balances must be unchanged so far")
    })


	msc.setDKGMiners(t, miners, balances)


	t.Run("pay fees [vc, no fees]", func(t *testing.T) {
		balances.requireAllBeZeros(t)
		msc.setRounds(t, 250, 251, balances)

		msc.requirePendingPoolsBeNotEmpty(t, balances)
		msc.requireActivePoolsBeEmpty(t, balances)

		setMagicBlock(t,
			unwrapClients(miners),
			unwrapClients(sharders),
			balances)

		now += timeDelta
		// all rewards go the nodes
		// nothing must be paid to stakers, but pools become active
		msc.callPayFees(t, balances, miners, sharders,
			generator.client.id, 0, 251, now)

		msc.requireActivePoolsBeNotEmpty(t, balances)

		balances.requireTotalAmountBeEqual(t, BlockReward)
		balances.requireStakersHaveZeros(t, miners, "stakers' balances must be unchanged so far")
		balances.requireStakersHaveZeros(t, sharders, "stakers' balances must be unchanged so far")

		var global = msc.retrieveGlobalNode(t, balances)
		require.EqualValues(t, 251, global.LastRound)

		msc.verifyRewardsWithoutFees(t, false,
			balances, generator, miners, sharders)

		balances.requireTotalAmountBeEqual(t, BlockReward)
	})

	msc.setDKGMiners(t, miners, balances)

	t.Run("pay fees [no vc, no fees]", func(t *testing.T) {
		msc.resetBalancesAndRewards(balances)
		msc.setRounds(t, 251, 501, balances)

		msc.requirePendingPoolsBeEmpty(t, balances)
		msc.requireActivePoolsBeNotEmpty(t, balances)

		now += timeDelta
		// pools active, no fees, part of rewards must be payed
		// to generator's and block sharders' stake holders
		msc.callPayFees(t, balances, miners, sharders,
			generator.client.id, 0, 252, now)

		balances.requireTotalAmountBeEqual(t, BlockReward)

		msc.verifyRewardsWithoutFees(t, true,
			balances, generator, miners, sharders)
	})

	// don't set DKG miners list, because no VC is expected

	t.Run("pay fees [no vc, fees]", func(t *testing.T) {
		msc.resetBalancesAndRewards(balances)
		msc.setRounds(t, 252, 501, balances)

		now += timeDelta
		// pools are active, rewards as above and +fees
		msc.callPayFees(t, balances, miners, sharders,
			generator.client.id, TransactionFee, 253, now)

		var (
			expected = make(map[string]state.Balance)
			actual   = make(map[string]state.Balance)
		)

		balances.requireNodesHaveZeros(t, miners, "miners' balances must be zero")
		balances.requireNodesHaveZeros(t, sharders, "sharders' balances must be zero")

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

		for _, sharder := range filterClientsByIds(sharders, balances.blockSharders) {
			for _, staker := range sharder.stakers {
				expected[staker.id] += 21e7 + 3e10 // + block sharders fees
			}
		}

		require.Equal(t, len(expected), len(actual), "sizes of balance maps")
		require.Equal(t, expected, actual, "balances")

		//balances.requireTotalAmountBeEqual(t, BlockReward)
	})

	// don't set DKG miners list, because no VC is expected

	t.Run("pay fees [vc, interests]", func(t *testing.T) {
		msc.resetBalancesAndRewards(balances)
		msc.setRounds(t, 500, 501, balances)

		now += timeDelta
		// pools are active, rewards as above and +fees
		msc.callPayFees(t, balances, miners, sharders,
			generator.client.id, 0, 501, now)

		var (
			expected = make(map[string]state.Balance)
			actual   = make(map[string]state.Balance)
		)

		balances.requireNodesHaveZeros(t, miners, "miners' balances must be zero")
		balances.requireNodesHaveZeros(t, sharders, "sharders' balances must be zero")

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

		for _, sharder := range filterClientsByIds(sharders, balances.blockSharders) {
			for _, staker := range sharder.stakers {
				expected[staker.id] += 21e7
			}
		}

		require.Equal(t, len(expected), len(actual), "sizes of balance maps")
		require.Equal(t, expected, actual, "balances")

		//balances.requireTotalAmountBeEqual(t, BlockReward)
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

func (msc *MinerSmartContract) callPayFees(t *testing.T,
	balances    *testBalances,
	miners      []*TestClient,
	sharders    []*TestClient,
	generatorId datastore.Key,
	fee, round, now int64) {

	var blck = block.Provider().(*block.Block)

	blck.Round = round
	blck.MinerID = generatorId

	blck.PrevBlock = block.Provider().(*block.Block)  // stub

	var tx = newTransaction(generatorId, ADDRESS, 0, now)

	if fee > 0 {
		tx.Fee = fee
	}
	blck.Txns = append(blck.Txns, tx)

	//todo: check (initially tx.Fee and blck.Txns setting were after this:)
	balances.txn = tx
	balances.block = blck
	balances.blockSharders = selectRandom(sharders, BlockShardersAmount)

	var global, err = msc.getGlobalNode(balances)
	require.NoError(t, err, "getting global node")

	_, err = msc.payFees(tx, nil, global, balances)
	require.NoError(t, err, "pay_fees error")

	balances.requireClientWalletsBeEmpty(t, append(miners,sharders...))
}

func (msc *MinerSmartContract) verifyRewardsWithoutFees(t *testing.T,
	withStakers bool, balances *testBalances,
	generator *TestClient, miners []*TestClient,
	sharders []*TestClient) {

	var global = msc.retrieveGlobalNode(t, balances)
	require.EqualValues(t, BlockReward, global.Minted)

	var blockSharders = filterClientsByIds(sharders, balances.blockSharders)

	var generatorShare, shardersShare = global.splitByShareRatio(BlockReward)
	var singleSharderShare = shardersShare / BlockShardersAmount
	generatorShare += shardersShare % BlockShardersAmount

	if !withStakers {
		require.EqualValues(t, generatorShare,
			balances.balances[generator.delegate.id])

		balances.requireSpecifiedBeEqual(t, unwrapDelegates(blockSharders),
			singleSharderShare, "sharders must have equal shares")

		balances.requireStakersHaveZeros(t, miners,
			"stakers' balances must be unchanged so far")
		balances.requireStakersHaveZeros(t, sharders,
			"stakers' balances must be unchanged so far")
	}

	balances.requireNodeAndStakersSumUpTo(t, generator.delegate,
		generator.stakers, generatorShare)
	for _, sharder := range blockSharders {
		balances.requireNodeAndStakersSumUpTo(t, sharder.delegate,
			sharder.stakers, singleSharderShare)
	}
}

func (msc *MinerSmartContract) setRounds(t *testing.T, last, vc int64,
	balances cstate.StateContextI) {

	var global, err = msc.getGlobalNode(balances)
	require.NoError(t, err, "getting global node")
	global.LastRound = last
	global.ViewChange = vc
	require.NoError(t, global.save(balances), "saving global node")
}

func (msc *MinerSmartContract) retrieveGlobalNode(t *testing.T,
	balances *testBalances) *GlobalNode {

	var global, err = msc.getGlobalNode(balances)
	require.NoError(t, err, "can't get global node")

	return global
}

func (msc *MinerSmartContract) requireActivePoolsBeEmpty(t *testing.T,
	balances *testBalances) {
		msc.requirePools(t, balances, true, true)
}

func (msc *MinerSmartContract) requireActivePoolsBeNotEmpty(t *testing.T,
	balances *testBalances) {
		msc.requirePools(t, balances, true, false)
}

func (msc *MinerSmartContract) requirePendingPoolsBeEmpty(t *testing.T,
	balances *testBalances) {
		msc.requirePools(t, balances, false, true)
}

func (msc *MinerSmartContract) requirePendingPoolsBeNotEmpty(t *testing.T,
	balances *testBalances) {
		msc.requirePools(t, balances, false, false)
}

func (msc *MinerSmartContract) requirePools(t *testing.T,
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

// this method is used for easier balances verification
func (msc *MinerSmartContract) resetBalancesAndRewards(balances *testBalances) {
	var global, _ = msc.getGlobalNode(balances)
	global.Minted = 0
	balances.zeroize()
	global.save(balances)
}

func (tb *testBalances) requireClientWalletsBeEmpty(t *testing.T,
	nodes []*TestClient) {

	tb.requireSpecifiedBeEqual(t, unwrapClients(nodes),
		0, "client wallets required to be empty")
}

func (tb *testBalances) requireNodesHaveZeros(t *testing.T,
	nodes []*TestClient, message string) {

	tb.requireSpecifiedBeEqual(t, unwrapClients(nodes), 0, message + " (client wallets)")
	tb.requireSpecifiedBeEqual(t, unwrapDelegates(nodes), 0, message + " (delegate wallets)")
}

func (tb *testBalances) requireStakersHaveZeros(t *testing.T,
	nodes []*TestClient, message string) {

	for _, node := range nodes {
		tb.requireSpecifiedBeEqual(t, node.stakers, 0, message)
	}
}

func selectRandom(clients []*TestClient, n int) (selection []datastore.Key) {
	if n > len(clients) {
		panic("too many elements requested")
	}

	selection = make([]datastore.Key, 0, n)

	var permutations = rand.Perm(len(clients))
	for i := 0; i < n; i++ {
		selection = append(selection, clients[permutations[i]].client.id)
	}
	return
}

func filterClientsByIds(clients []*TestClient, ids []datastore.Key) (
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

func unwrapClients(clients []*TestClient) (list []*Client) {
	for _, miner := range clients {
		list = append(list, miner.client)
	}
	return
}

func unwrapDelegates(clients []*TestClient) (list []*Client) {
	for _, node := range clients {
		list = append(list, node.delegate)
	}
	return list
}
