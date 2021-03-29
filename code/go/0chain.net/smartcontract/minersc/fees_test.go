package minersc

import (
	"math/rand"
	"testing"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type miner struct {
	miner    *Client
	delegate *Client
	stakers  []*Client
}

// create and add miner, create stake holders, don't stake
func newMiner(t *testing.T, msc *MinerSmartContract, now, ns int64,
	val state.Balance, balances cstate.StateContextI) (mn *miner) {

	mn = new(miner)
	mn.miner, mn.delegate = addMiner(t, msc, now, balances)
	for i := int64(0); i < ns; i++ {
		mn.stakers = append(mn.stakers, newClient(val, balances))
	}
	return
}

// create and add sharder, create stake holders, don't stake
func newSharder(t *testing.T, msc *MinerSmartContract, now, ns int64,
	val state.Balance, balances cstate.StateContextI) (sh *sharder) {

	sh = new(sharder)
	sh.sharder, sh.delegate = addSharder(t, msc, now, balances)
	for i := int64(0); i < ns; i++ {
		sh.stakers = append(sh.stakers, newClient(val, balances))
	}
	return
}

type sharder struct {
	sharder  *Client
	delegate *Client
	stakers  []*Client
}

func extractMiners(miners []*miner) (list []*Client) {
	list = make([]*Client, 0, len(miners))
	for _, mn := range miners {
		list = append(list, mn.miner)
	}
	return
}

func extractSharders(sharders []*sharder) (list []*Client) {
	list = make([]*Client, 0, len(sharders))
	for _, mn := range sharders {
		list = append(list, mn.sharder)
	}
	return
}

// just few random sharders
func extractBlockSharders(sharders []*sharder, n int) (bs []string) {
	if n > len(sharders) {
		panic("to many sharders for block wanted")
	}
	bs = make([]string, 0, n)
	var perms = rand.Perm(len(sharders))
	for i := 0; i < n; i++ {
		bs = append(bs, sharders[perms[i]].sharder.id)
	}
	return
}

func getBlockSharders(sharders []*sharder, bs []string) (got []*sharder) {
	got = make([]*sharder, 0, len(bs))
	for _, sh := range sharders {
		for _, id := range bs {
			if sh.sharder.id == id {
				got = append(got, sh)
			}
		}
	}
	return
}

func createPreviousMagicBlock(miners []*miner, sharders []*sharder) (
	b *block.Block) {

	b = new(block.Block)

	b.MagicBlock = block.NewMagicBlock()
	b.MagicBlock.Miners = node.NewPool(node.NodeTypeMiner)
	b.MagicBlock.Sharders = node.NewPool(node.NodeTypeSharder)

	for _, mn := range miners {
		b.MagicBlock.Miners.NodesMap[mn.miner.id] = new(node.Node)
	}
	for _, sh := range sharders {
		b.MagicBlock.Sharders.NodesMap[sh.sharder.id] = new(node.Node)
	}
	return
}

func (msc *MinerSmartContract) setDKGMinersTestHelper(t *testing.T,
	miners []*miner, balances *testBalances) {

	t.Helper()

	var gn, err = msc.getGlobalNode(balances)
	require.NoError(t, err)

	var dmn *DKGMinerNodes
	dmn, err = msc.getMinersDKGList(balances)
	require.NoError(t, err)

	dmn.setConfigs(gn)
	for _, mn := range miners {
		dmn.SimpleNodes[mn.miner.id] = &SimpleNode{ID: mn.miner.id}
		dmn.Waited[mn.miner.id] = true
	}

	_, err = balances.InsertTrieNode(DKGMinersKey, dmn)
	require.NoError(t, err)
}

func Test_payFees(t *testing.T) {
	t.Skip("needs fixing")
	const stakeVal, stakeHolders = 10e10, 5

	var (
		balances = newTestBalances()
		msc      = newTestMinerSC()
		now      int64
		err      error

		miners   []*miner
		sharders []*sharder
	)

	setConfig(t, balances)

	t.Run("add miners", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			miners = append(miners, newMiner(t, msc, now, stakeHolders,
				stakeVal, balances))
			now += 10
		}
	})

	t.Run("add sharders", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			sharders = append(sharders, newSharder(t, msc, now, stakeHolders,
				stakeVal, balances))
			now += 10
		}
	})

	// add all the miners to DKG miners list
	// add all the miners and the sharders to latest finalized magic block

	msc.setDKGMinersTestHelper(t, miners, balances)
	balances.setLFMB(createPreviousMagicBlock(miners, sharders))

	t.Run("stake miners", func(t *testing.T) {
		for _, mn := range miners {
			for _, st := range mn.stakers {
				_, err = st.callAddToDelegatePool(t, msc, now, stakeVal,
					mn.miner.id, balances)
				require.NoError(t, err, "staking miner")
				now += 10
			}
		}

		for _, mn := range miners {
			assert.Zero(t, balances.balances[mn.miner.id], "balance?")
			assert.Zero(t, balances.balances[mn.delegate.id], "balance?")
			for _, st := range mn.stakers {
				assert.Zero(t, balances.balances[st.id], "balance?")
			}
		}
	})

	t.Run("stake sharders", func(t *testing.T) {
		for _, sh := range sharders {
			for _, st := range sh.stakers {
				_, err = st.callAddToDelegatePool(t, msc, now, stakeVal,
					sh.sharder.id, balances)
				require.NoError(t, err, "staking sharder")
				now += 10
			}
		}

		for _, sh := range sharders {
			assert.Zero(t, balances.balances[sh.sharder.id], "balance?")
			assert.Zero(t, balances.balances[sh.delegate.id], "balance?")
			for _, st := range sh.stakers {
				assert.Zero(t, balances.balances[st.id], "balance?")
			}
		}
	})

	// add all the miners to DKG miners list
	msc.setDKGMinersTestHelper(t, miners, balances)

	t.Run("pay fees -> view change", func(t *testing.T) {

		for id, bal := range balances.balances {
			if id == ADDRESS {
				continue
			}
			assert.Zerof(t, bal, "unexpected balance: %s", id)
		}

		setRounds(t, msc, 250, 251, balances)
		setMagicBlock(t, extractMiners(miners), extractSharders(sharders),
			balances)
		var (
			b         = block.Provider().(*block.Block)
			generator = miners[0]
		)
		b.Round = 251                                 // VC round
		b.MinerID = generator.miner.id                // block generator
		b.PrevBlock = block.Provider().(*block.Block) // stub
		// payFees transaction
		now += 10
		var tx = newTransaction(generator.miner.id, ADDRESS, 0, now)
		balances.txn = tx
		balances.block = b
		balances.blockSharders = extractBlockSharders(sharders, 3)
		var gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "getting global node")
		_, err = msc.payFees(tx, nil, gn, balances)
		require.NoError(t, err, "pay_fees error")

		// pools becomes active, nothing should be payed

		for _, mn := range miners {
			assert.Zero(t, balances.balances[mn.miner.id],
				"miner balance")
			assert.Zero(t, balances.balances[mn.delegate.id],
				"miner delegate balance?")
			for _, st := range mn.stakers {
				assert.Zero(t, balances.balances[st.id], "stake balance?")
			}
		}
		for _, sh := range sharders {
			assert.Zero(t, balances.balances[sh.sharder.id],
				"sharder balance")
			assert.Zero(t, balances.balances[sh.delegate.id],
				"sharder delegate balance?")
			for _, st := range sh.stakers {
				assert.Zero(t, balances.balances[st.id], "stake balance?")
			}
		}

		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "can't get global node")
		assert.EqualValues(t, 251, gn.LastRound)
		assert.EqualValues(t, 0, gn.Minted)
	})

	// add all the miners to DKG miners list
	msc.setDKGMinersTestHelper(t, miners, balances)

	t.Run("pay fees -> no fees", func(t *testing.T) {

		for id, bal := range balances.balances {
			if id == ADDRESS {
				continue
			}
			assert.Zerof(t, bal, "unexpected balance: %s", id)
		}

		setRounds(t, msc, 251, 501, balances)
		var (
			b         = block.Provider().(*block.Block)
			generator = miners[1]
		)
		b.Round = 252                                 // VC round
		b.MinerID = generator.miner.id                // block generator
		b.PrevBlock = block.Provider().(*block.Block) // stub
		// payFees transaction
		now += 10
		var tx = newTransaction(generator.miner.id, ADDRESS, 0, now)
		balances.txn = tx
		balances.block = b
		balances.blockSharders = extractBlockSharders(sharders, 3)
		var gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "getting global node")
		_, err = msc.payFees(tx, nil, gn, balances)
		require.NoError(t, err, "pay_fees error")

		// pools active, no fees, rewards should be payed for
		// generator's and block sharders' stake holders

		var (
			expected = make(map[string]state.Balance)
			got      = make(map[string]state.Balance)
		)

		for _, mn := range miners {
			assert.Zero(t, balances.balances[mn.miner.id])
			assert.Zero(t, balances.balances[mn.delegate.id])
			for _, st := range mn.stakers {
				if mn == generator {
					expected[st.id] += 77e7
				} else {
					expected[st.id] = 0
				}
				got[st.id] = balances.balances[st.id]
			}
		}

		for _, sh := range sharders {
			assert.Zero(t, balances.balances[sh.sharder.id])
			assert.Zero(t, balances.balances[sh.delegate.id])
			for _, st := range sh.stakers {
				expected[st.id] += 0
				got[st.id] = balances.balances[st.id]
			}
		}

		for _, sh := range getBlockSharders(sharders, balances.blockSharders) {
			for _, st := range sh.stakers {
				expected[st.id] += 21e7
			}
		}

		assert.Equal(t, expected, got, "balances")

	})

	// don't set DKG miners list, because no VC is expected

	// reset all balances
	balances.balances = make(map[string]state.Balance)

	t.Run("pay fees -> with fees", func(t *testing.T) {

		setRounds(t, msc, 252, 501, balances)
		var (
			b         = block.Provider().(*block.Block)
			generator = miners[1]
		)
		b.Round = 253                                 // VC round
		b.MinerID = generator.miner.id                // block generator
		b.PrevBlock = block.Provider().(*block.Block) // stub
		// payFees transaction
		now += 10
		var tx = newTransaction(generator.miner.id, ADDRESS, 0, now)
		balances.txn = tx
		balances.block = b
		balances.blockSharders = extractBlockSharders(sharders, 3)
		// add fees
		tx.Fee = 100e10
		b.Txns = append(b.Txns, tx)
		var gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "getting global node")
		_, err = msc.payFees(tx, nil, gn, balances)
		require.NoError(t, err, "pay_fees error")

		// pools are active, rewards as above and +fees

		var (
			expected = make(map[string]state.Balance)
			got      = make(map[string]state.Balance)
		)

		for _, mn := range miners {
			assert.Zero(t, balances.balances[mn.miner.id])
			assert.Zero(t, balances.balances[mn.delegate.id])
			for _, st := range mn.stakers {
				if mn == generator {
					expected[st.id] += 77e7 + 11e10 // + generator fees
				} else {
					expected[st.id] += 0
				}
				got[st.id] = balances.balances[st.id]
			}
		}

		for _, sh := range sharders {
			assert.Zero(t, balances.balances[sh.sharder.id])
			assert.Zero(t, balances.balances[sh.delegate.id])
			for _, st := range sh.stakers {
				expected[st.id] += 0
				got[st.id] = balances.balances[st.id]
			}
		}

		for _, sh := range getBlockSharders(sharders, balances.blockSharders) {
			for _, st := range sh.stakers {
				expected[st.id] += 21e7 + 3e10 // + block sharders fees
			}
		}

		assert.Equal(t, expected, got, "balances")

	})

	// don't set DKG miners list, because no VC is expected

	// reset all balances
	balances.balances = make(map[string]state.Balance)

	t.Run("pay fees -> view change interests", func(t *testing.T) {

		setRounds(t, msc, 500, 501, balances)
		var (
			b         = block.Provider().(*block.Block)
			generator = miners[1]
		)
		b.Round = 501                                 // VC round
		b.MinerID = generator.miner.id                // block generator
		b.PrevBlock = block.Provider().(*block.Block) // stub
		// payFees transaction
		now += 10
		var tx = newTransaction(generator.miner.id, ADDRESS, 0, now)
		balances.txn = tx
		balances.block = b
		balances.blockSharders = extractBlockSharders(sharders, 3)
		// add fees
		var gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "getting global node")
		_, err = msc.payFees(tx, nil, gn, balances)
		require.NoError(t, err, "pay_fees error")

		// pools are active, rewards as above and +fees

		var (
			expected = make(map[string]state.Balance)
			got      = make(map[string]state.Balance)
		)

		for _, mn := range miners {
			assert.Zero(t, balances.balances[mn.miner.id])
			assert.Zero(t, balances.balances[mn.delegate.id])
			for _, st := range mn.stakers {
				if mn == generator {
					expected[st.id] += 77e7 + 1e10
				} else {
					expected[st.id] += 1e10
				}
				got[st.id] = balances.balances[st.id]
			}
		}

		for _, sh := range sharders {
			assert.Zero(t, balances.balances[sh.sharder.id])
			assert.Zero(t, balances.balances[sh.delegate.id])
			for _, st := range sh.stakers {
				expected[st.id] += 1e10
				got[st.id] = balances.balances[st.id]
			}
		}

		for _, sh := range getBlockSharders(sharders, balances.blockSharders) {
			for _, st := range sh.stakers {
				expected[st.id] += 21e7
			}
		}

		assert.Equal(t, expected, got, "balances")

	})

	t.Run("epoch", func(t *testing.T) {
		var gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		var ir, rr = gn.InterestRate, gn.RewardRate
		gn.epochDecline()
		assert.True(t, gn.InterestRate < ir)
		assert.True(t, gn.RewardRate < rr)
	})

}
