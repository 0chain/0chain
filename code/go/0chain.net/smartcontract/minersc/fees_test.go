package minersc

import (
	"math/rand"
	"testing"

	"0chain.net/smartcontract/provider"
	"0chain.net/smartcontract/stakepool/spenum"

	"github.com/0chain/common/core/currency"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type miner struct {
	miner    *Client
	node     *MinerNode
	delegate *Client
	stakers  []*Client
}

func (m *miner) execAddMinerTxn(msc *MinerSmartContract, now int64, balances cstate.StateContextI) (string, error) {
	return m.miner.callAddMiner(msc, now, m.node, balances)
}

// create and add miner, create stake holders, don't stake
func newMinerWithStake(t *testing.T, msc *MinerSmartContract, now, ns int64,
	val currency.Coin, saveToMB bool, balances cstate.StateContextI) (mn *miner, err error) {
	mn, err = addMiner(t, msc, now, saveToMB, balances)
	if err != nil {
		return nil, err
	}

	for i := int64(0); i < ns; i++ {
		mn.stakers = append(mn.stakers, newClient(val, balances))
	}
	return
}

// create and add sharder, create stake holders, don't stake
func newSharderWithStake(t *testing.T, msc *MinerSmartContract, now, ns int64,
	val currency.Coin, saveToMB bool, balances cstate.StateContextI) (sh *sharder, err error) {
	sh, err = addSharder(t, msc, now, saveToMB, balances)
	if err != nil {
		return nil, err
	}

	for i := int64(0); i < ns; i++ {
		sh.stakers = append(sh.stakers, newClient(val, balances))
	}
	return
}

type sharder struct {
	sharder  *Client
	node     *MinerNode
	delegate *Client
	stakers  []*Client
}

func (s *sharder) execAddSharderTxn(msc *MinerSmartContract, now int64, balances cstate.StateContextI) (string, error) {
	return s.sharder.callAddSharder(msc, now, s.node, balances)
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

	var gn, err = getGlobalNode(balances)
	require.NoError(t, err)

	var dmn *DKGMinerNodes
	dmn, err = getDKGMinersList(balances)
	require.NoError(t, err)

	dmn.setConfigs(gn)
	for _, mn := range miners {
		dmn.SimpleNodes[mn.miner.id] = &SimpleNode{
			Provider: provider.Provider{
				ID:           mn.miner.id,
				ProviderType: spenum.Miner,
			},
		}
		dmn.Waited[mn.miner.id] = true
	}

	err = updateDKGMinersList(balances, dmn)
	require.NoError(t, err)
}

func existInDelegatesOfNodes(id string, nodes []*MinerNode) bool {
	for _, n := range nodes {
		if n.Settings.DelegateWallet == id {
			return true
		}
	}
	return false
}

func computeMinerPayments(gn *GlobalNode, msc *MinerSmartContract, b *block.Block) (currency.Coin, error) {
	blockReward := gn.MustBase().BlockReward
	minerR, _, err := gn.splitByShareRatio(blockReward)
	if err != nil {
		return 0, err
	}
	fees, err := msc.sumFee(b, false)
	if err != nil {
		return 0, err
	}
	minerF, _, err := gn.splitByShareRatio(fees)
	if err != nil {
		return 0, err
	}

	return minerR + minerF, nil
}

func computeShardersPayments(gn *GlobalNode, msc *MinerSmartContract, b *block.Block) (currency.Coin, error) {
	blockReward := gn.MustBase().BlockReward
	_, sharderR, err := gn.splitByShareRatio(blockReward)
	if err != nil {
		return 0, err
	}
	fees, err := msc.sumFee(b, false)
	if err != nil {
		return 0, err
	}
	_, sharderF, err := gn.splitByShareRatio(fees)
	if err != nil {
		return 0, err
	}
	return sharderR + sharderF, nil
}

func Test_payFees(t *testing.T) {
	t.Skip("Needs to be reworked. We now no longer pay fees with transfers and mints")
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

	for i := 0; i < 10; i++ {
		mn, err := newMinerWithStake(t, msc, now, stakeHolders, stakeVal, true, balances)
		require.NoError(t, err)
		miners = append(miners, mn)
		now += 10
	}

	for i := 0; i < 10; i++ {
		sn, err := newSharderWithStake(t, msc, now, stakeHolders, stakeVal, true, balances)
		require.NoError(t, err)
		sharders = append(sharders, sn)
		now += 10
	}

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
		var gn, err = getGlobalNode(balances)
		require.NoError(t, err, "getting global node")
		_, err = msc.payFees(tx, nil, gn, balances)
		require.NoError(t, err, "pay_fees error")

		// pools becomes active, nothing should be payed

		for _, mn := range miners {
			if mn == generator {
				mnPayment, err := computeMinerPayments(gn, msc, b)
				require.NoError(t, err)
				assert.Equal(t,
					balances.balances[mn.delegate.id],
					mnPayment,
				)
				balances.balances[mn.delegate.id] = 0
			}

			assert.Zero(t, balances.balances[mn.miner.id],
				"miner balance")
			assert.Zero(t, balances.balances[mn.delegate.id],
				"miner delegate balance?")
			for _, st := range mn.stakers {
				assert.Zero(t, balances.balances[st.id], "stake balance?")
			}
		}

		blockSharders, err := getAllShardersList(balances)
		require.NoError(t, err, "getShardersList error")
		for _, sh := range sharders {
			if existInDelegatesOfNodes(sh.delegate.id, blockSharders.Nodes) {
				shP, err := computeShardersPayments(gn, msc, b)
				require.NoError(t, err)
				shP = shP / currency.Coin(len(blockSharders.Nodes))

				assert.Equal(t,
					balances.balances[sh.delegate.id],
					shP,
				)
				balances.balances[sh.delegate.id] = 0
			}

			assert.Zero(t, balances.balances[sh.sharder.id],
				"sharder balance")
			assert.Zero(t, balances.balances[sh.delegate.id],
				"sharder delegate balance?")
			for _, st := range sh.stakers {
				assert.Zero(t, balances.balances[st.id], "stake balance?")
			}
		}

		gn, err = getGlobalNode(balances)
		require.NoError(t, err, "can't get global node")
		assert.EqualValues(t, 251, gn.MustBase().LastRound)
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
		var gn, err = getGlobalNode(balances)
		require.NoError(t, err, "getting global node")
		_, err = msc.payFees(tx, nil, gn, balances)
		require.NoError(t, err, "pay_fees error")

		// pools active, no fees, rewards should be payed for
		// generator's and block sharders' stake holders

		var (
			expected = make(map[string]currency.Coin)
			got      = make(map[string]currency.Coin)
		)

		for _, mn := range miners {
			mnPayment, err := computeMinerPayments(gn, msc, b)
			require.NoError(t, err)
			if mn == generator {
				assert.Equal(t,
					balances.balances[mn.delegate.id],
					mnPayment,
				)
				balances.balances[mn.delegate.id] = 0
			}

			assert.Zero(t, balances.balances[mn.miner.id])
			assert.Zero(t, balances.balances[mn.delegate.id])
			for _, st := range mn.stakers {
				expected[st.id] = 0
				got[st.id] = balances.balances[st.id]
			}
		}

		blockSharders, err := getAllShardersList(balances)
		require.NoError(t, err)
		sharderPayments, err := computeShardersPayments(gn, msc, b)
		require.NoError(t, err)
		sharderPayments = sharderPayments / currency.Coin(len(blockSharders.Nodes))
		for _, sh := range sharders {
			if existInDelegatesOfNodes(sh.delegate.id, blockSharders.Nodes) {
				assert.Equal(t,
					balances.balances[sh.delegate.id],
					sharderPayments,
				)
				balances.balances[sh.delegate.id] = 0
			}
			assert.Zero(t, balances.balances[sh.sharder.id])
			assert.Zero(t, balances.balances[sh.delegate.id])
			for _, st := range sh.stakers {
				expected[st.id] += 0
				got[st.id] = balances.balances[st.id]
			}
		}

		assert.Equal(t, expected, got, "balances")
	})

	// don't set DKG miners list, because no VC is expected

	// reset all balances
	balances.balances = make(map[string]currency.Coin)

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
		var gn, err = getGlobalNode(balances)
		require.NoError(t, err, "getting global node")
		_, err = msc.payFees(tx, nil, gn, balances)
		require.NoError(t, err, "pay_fees error")

		// pools are active, rewards as above and +fees

		var (
			expected = make(map[string]currency.Coin)
			got      = make(map[string]currency.Coin)
		)

		for _, mn := range miners {
			mnPayment, err := computeMinerPayments(gn, msc, b)
			require.NoError(t, err)
			if mn == generator {
				assert.Equal(t,
					balances.balances[mn.delegate.id],
					mnPayment,
				)
				balances.balances[mn.delegate.id] = 0
			}

			assert.Zero(t, balances.balances[mn.miner.id])
			assert.Zero(t, balances.balances[mn.delegate.id])

			for _, st := range mn.stakers {
				expected[st.id] += 0
				got[st.id] = balances.balances[st.id]
			}
		}

		blockSharders, err := getAllShardersList(balances)
		require.NoError(t, err)
		for _, sh := range sharders {
			if existInDelegatesOfNodes(sh.delegate.id, blockSharders.Nodes) {
				shP, err := computeShardersPayments(gn, msc, b)
				require.NoError(t, err)
				shP = shP / currency.Coin(len(blockSharders.Nodes))
				assert.Equal(t,
					balances.balances[sh.delegate.id],
					shP,
				)
				balances.balances[sh.delegate.id] = 0
			}

			assert.Zero(t, balances.balances[sh.sharder.id])
			assert.Zero(t, balances.balances[sh.delegate.id])

			for _, st := range sh.stakers {
				expected[st.id] += 0
				got[st.id] = balances.balances[st.id]
			}
		}

		assert.Equal(t, expected, got, "balances")
	})

	// don't set DKG miners list, because no VC is expected

	// reset all balances
	balances.balances = make(map[string]currency.Coin)

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
		var gn, err = getGlobalNode(balances)
		require.NoError(t, err, "getting global node")
		_, err = msc.payFees(tx, nil, gn, balances)
		require.NoError(t, err, "pay_fees error")

		// pools are active, rewards as above and +fees

		var (
			expected = make(map[string]currency.Coin)
			got      = make(map[string]currency.Coin)
		)

		for _, mn := range miners {
			mnPayment, err := computeMinerPayments(gn, msc, b)
			require.NoError(t, err)
			if mn == generator {
				assert.Equal(t,
					balances.balances[mn.delegate.id],
					mnPayment,
				)
				balances.balances[mn.delegate.id] = 0
			}

			assert.Zero(t, balances.balances[mn.miner.id])
			assert.Zero(t, balances.balances[mn.delegate.id])
			for _, st := range mn.stakers {
				expected[st.id] += 0
				got[st.id] = balances.balances[st.id]
			}
		}

		blockSharders, err := getAllShardersList(balances)
		require.NoError(t, err)
		for _, sh := range sharders {
			if existInDelegatesOfNodes(sh.delegate.id, blockSharders.Nodes) {
				shP, err := computeShardersPayments(gn, msc, b)
				require.NoError(t, err)
				shP = shP / currency.Coin(len(blockSharders.Nodes))
				assert.Equal(t,
					balances.balances[sh.delegate.id],
					shP,
				)
				balances.balances[sh.delegate.id] = 0
			}

			assert.Zero(t, balances.balances[sh.sharder.id])
			assert.Zero(t, balances.balances[sh.delegate.id])

			for _, st := range sh.stakers {
				expected[st.id] += 0
				got[st.id] = balances.balances[st.id]
			}
		}

		assert.Equal(t, expected, got, "balances")

	})

	t.Run("epoch", func(t *testing.T) {
		var gn, err = getGlobalNode(balances)
		require.NoError(t, err)
		var rr = gn.MustBase().RewardRate
		gn.epochDecline()
		assert.True(t, gn.MustBase().RewardRate < rr)
	})

}
