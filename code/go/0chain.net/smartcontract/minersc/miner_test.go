package minersc

import (
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/state"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const stakeVal, stakeHolders = 1e10, 3

func Test_DeleteNode(t *testing.T) {

	var (
		balances *testBalances
		msc      = newTestMinerSC()
		now      int64

		miners   []*miner
		sharders []*sharder
	)

	updateTestCases := []struct {
		title     string
		testFuncs func(t *testing.T)
	}{
		{
			"delete miner pending pool",
			func(t *testing.T) {
				payFees(t, balances, msc, miners, sharders, 251, 252, now)
				// check that the balance for the stakers of miner 0 are 0
				checkStakersBalances(t, miners[0].stakers, balances, 0, "checking after staking")
				// delete miner at index 0
				deleteMiner(t, balances, msc, miners, sharders, "", 251, 252, now, 0)
				// check that the balance for the stakers of miner 0 are back to the stake value
				checkStakersBalances(t, miners[0].stakers, balances, stakeVal, "checking after delete")
			},
		},

		{
			"delete miner active pool",
			func(t *testing.T) {
				payFees(t, balances, msc, miners, sharders, 251, 251, now)
				// check that the balance for the stakers of miner 0 are 0
				checkStakersBalances(t, miners[0].stakers, balances, 0, "checking after staking")
				// delete miner at index 0
				deleteMiner(t, balances, msc, miners, sharders, "", 252, 501, now, 0)
				checkStakersBalances(t, miners[0].stakers, balances, 0, "checking after delete started")
				msc.setDKGMinersTestHelper(t, miners, balances)
				payFees(t, balances, msc, miners, sharders, 501, 501, now)
				// check that the balance for the stakers of miner 0 are back to the stake value along with the interest and rewards
				interest := getMinerStakerInterest(t, balances, msc, miners[0])
				reward := getMinerStakerReward(t, balances, msc, miners[0], 1)
				checkStakersBalances(t, miners[0].stakers, balances, interest+reward+stakeVal, "checking after delete completed")
			},
		},
		{
			"delete miner in wait phase",
			func(t *testing.T) {
				payFees(t, balances, msc, miners, sharders, 251, 251, now)
				// check that the balance for the stakers of miner 0 are 0
				checkStakersBalances(t, miners[0].stakers, balances, 0, "checking after staking")
				// set phasers to stun
				pn := &PhaseNode{Phase: Wait, StartRound: 251, CurrentRound: 252, Restarts: 0}
				_, err := balances.InsertTrieNode(pn.GetKey(), pn)
				require.NoError(t, err, "setting_phasers error")
				// delete miner at index 0
				deleteMiner(t, balances, msc, miners, sharders,
					"failed to delete from view change: magic block has already been created for next view change",
					252, 501, now, 0)
			},
		},
		{
			"delete sharder pending pool",
			func(t *testing.T) {
				payFees(t, balances, msc, miners, sharders, 251, 252, now)
				// check that the balance for the stakers of miner 0 are 0
				checkStakersBalances(t, sharders[0].stakers, balances, 0, "checking after staking")
				// delete miner at index 0
				deleteSharder(t, balances, msc, miners, sharders, "", 251, 252, now, 0)
				// check that the balance for the stakers of miner 0 are back to the stake value
				checkStakersBalances(t, sharders[0].stakers, balances, stakeVal, "checking after delete")
			},
		},
		{
			"delete sharder active pool",
			func(t *testing.T) {
				payFees(t, balances, msc, miners, sharders, 251, 251, now)
				// check that the balance for the stakers of miner 0 are 0
				checkStakersBalances(t, sharders[0].stakers, balances, 0, "checking after staking")
				// delete miner at index 0
				deleteSharder(t, balances, msc, miners, sharders, "", 252, 501, now, 0)
				checkStakersBalances(t, sharders[0].stakers, balances, 0, "checking after delete started")
				msc.setDKGMinersTestHelper(t, miners, balances)
				payFees(t, balances, msc, miners, sharders, 501, 501, now)
				// check that the balance for the stakers of miner 0 are back to the stake value along with the interest and rewards
				interest := getSharderStakerInterest(t, balances, msc, sharders[0])
				reward := getSharderStakerReward(t, balances, msc, sharders[0], 1)
				checkStakersBalances(t, sharders[0].stakers, balances, interest+reward+stakeVal, "checking after delete completed")
			},
		},
		{
			"delete sharder in wait phase",
			func(t *testing.T) {
				payFees(t, balances, msc, miners, sharders, 251, 251, now)
				// check that the balance for the stakers of miner 0 are 0
				checkStakersBalances(t, sharders[0].stakers, balances, 0, "checking after staking")
				// set phasers to stun
				pn := &PhaseNode{Phase: Wait, StartRound: 251, CurrentRound: 252, Restarts: 0}
				_, err := balances.InsertTrieNode(pn.GetKey(), pn)
				require.NoError(t, err, "setting_phasers error")
				// delete miner at index 0
				deleteSharder(t, balances, msc, miners, sharders,
					"failed to delete from view change: magic block has already been created for next view change",
					252, 501, now, 0)
			},
		},
	}
	for _, tc := range updateTestCases {
		t.Run(tc.title, func(t *testing.T) {
			balances = newTestBalances()
			setConfig(t, balances)
			miners, sharders = setupChain(t, miners, sharders, balances, msc, now)
			setMagicBlock(t, extractMiners(miners), extractSharders(sharders), balances)

			tc.testFuncs(t)
		})
	}
}

func payFees(t *testing.T, balances *testBalances, msc *MinerSmartContract, miners []*miner, sharders []*sharder, round, vc, now int64) {
	t.Run("pay fees", func(t *testing.T) {
		setRounds(t, msc, round-1, vc, balances)
		setupBlock(balances, miners, sharders, now, round)
		var gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "getting global node")
		_, err = msc.payFees(balances.txn, nil, gn, balances)
		require.NoError(t, err, "pay_fees error")
	})
}

func deleteMiner(t *testing.T, balances *testBalances, msc *MinerSmartContract, miners []*miner, sharders []*sharder, errMessage string, round, vc, now int64, minerIndex int) {
	t.Run("delete miner", func(t *testing.T) {
		setRounds(t, msc, round-1, vc, balances)
		setupBlock(balances, miners, sharders, now, round)
		var gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "getting global node")
		if minerIndex > len(miners) {
			t.Errorf("minerIndex(%v) is greater than length of miners(%v)", minerIndex, len(miners))
		}
		mn := &MinerNode{SimpleNode: &SimpleNode{ID: miners[minerIndex].miner.id}}
		_, err = msc.DeleteMiner(balances.txn, mn.Encode(), gn, balances)
		if errMessage == "" {
			require.NoError(t, err, "delete_miner error")
		} else {
			assertErrMsg(t, err, errMessage)
		}

	})
}

func deleteSharder(t *testing.T, balances *testBalances, msc *MinerSmartContract, miners []*miner, sharders []*sharder, errMessage string, round, vc, now int64, sharderIndex int) {
	t.Run("delete sharder", func(t *testing.T) {
		setRounds(t, msc, round-1, vc, balances)
		setupBlock(balances, miners, sharders, now, round)
		var gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "getting global node")
		if sharderIndex > len(sharders) {
			t.Errorf("sharderIndex(%v) is greater than length of sharders(%v)", sharderIndex, len(sharders))
		}
		mn := &MinerNode{SimpleNode: &SimpleNode{ID: sharders[sharderIndex].sharder.id}}
		_, err = msc.DeleteSharder(balances.txn, mn.Encode(), gn, balances)
		if errMessage == "" {
			require.NoError(t, err, "delete_sharder error")
		} else {
			assertErrMsg(t, err, errMessage)
		}
	})
}

func getSharderStakerReward(t *testing.T, balances *testBalances, msc *MinerSmartContract, sharder *sharder, roundsPaid int) state.Balance {
	var rewards state.Balance
	t.Run("get sharder staker reward", func(t *testing.T) {
		var gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "getting global node")
		var sn *MinerNode
		sn, err = msc.getSharderNode(sharder.sharder.id, balances)
		require.NoError(t, err, "getting sharder node")
		blockReward := state.Balance(float64(gn.BlockReward) * gn.RewardRate)
		sharderSplit := blockReward - state.Balance(float64(blockReward)*(gn.ShareRatio))
		stakersSplit := sharderSplit - state.Balance(float64(sharderSplit)*sn.ServiceCharge)
		rewards = state.Balance(float64(stakersSplit)/float64(len(sharder.stakers))/float64(len(balances.blockSharders))) * state.Balance(roundsPaid)
	})
	return rewards
}

func getSharderStakerInterest(t *testing.T, balances *testBalances, msc *MinerSmartContract, sharder *sharder) state.Balance {
	var interest state.Balance
	t.Run("get sharder staker interest", func(t *testing.T) {
		var gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "getting global node")
		interest = state.Balance(float64(stakeVal) * gn.InterestRate)

	})
	return interest
}

func getMinerStakerReward(t *testing.T, balances *testBalances, msc *MinerSmartContract, miner *miner, roundsPaid int) state.Balance {
	var rewards state.Balance
	t.Run("get miner staker reward", func(t *testing.T) {
		var gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "getting global node")
		var mn *MinerNode
		mn, err = msc.getMinerNode(miner.miner.id, balances)
		require.NoError(t, err, "getting miner node")
		blockReward := state.Balance(float64(gn.BlockReward) * gn.RewardRate)
		minerSplit := state.Balance(float64(blockReward) * (gn.ShareRatio))
		stakersSplit := minerSplit - state.Balance(float64(minerSplit)*mn.ServiceCharge)
		rewards = state.Balance(float64(stakersSplit)/float64(len(miner.stakers))) * state.Balance(roundsPaid)
	})
	return rewards
}

func getMinerStakerInterest(t *testing.T, balances *testBalances, msc *MinerSmartContract, miner *miner) state.Balance {
	var interest state.Balance
	t.Run("get miner staker interest", func(t *testing.T) {
		var gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "getting global node")
		interest = state.Balance(float64(stakeVal) * gn.InterestRate)
	})
	return interest
}

func setupChain(t *testing.T, miners []*miner, sharders []*sharder, balances *testBalances, msc *MinerSmartContract, now int64) ([]*miner, []*sharder) {
	miners, sharders = nil, nil
	t.Run("add miners", func(t *testing.T) {
		for i := 0; i < 4; i++ {
			miners = append(miners, newMiner(t, msc, now, stakeHolders,
				stakeVal, balances))
			now += 10
		}
	})

	t.Run("add sharders", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			sharders = append(sharders, newSharder(t, msc, now, stakeHolders,
				stakeVal, balances))
			now += 10
		}
	})

	msc.setDKGMinersTestHelper(t, miners, balances)
	balances.setLFMB(createPreviousMagicBlock(miners, sharders))

	t.Run("stake miners", func(t *testing.T) {
		for _, mn := range miners {
			for _, st := range mn.stakers {
				_, err := st.callAddToDelegatePool(t, msc, now, stakeVal,
					mn.miner.id, balances)
				require.NoError(t, err, "staking miner")
				now += 10
			}
		}
	})

	t.Run("stake sharders", func(t *testing.T) {
		for _, sh := range sharders {
			for _, st := range sh.stakers {
				_, err := st.callAddToDelegatePool(t, msc, now, stakeVal,
					sh.sharder.id, balances)
				require.NoError(t, err, "staking sharder")
				now += 10
			}
		}
	})
	return miners, sharders
}

func checkStakersBalances(t *testing.T, stakers []*Client, balances *testBalances, value state.Balance, check string) {
	t.Run(check, func(t *testing.T) {
		for _, staker := range stakers {
			assert.EqualValues(t, value, balances.balances[staker.id])
		}
	})
}

func setupBlock(balances *testBalances, miners []*miner, sharders []*sharder, now, round int64) {
	var (
		b         = block.Provider().(*block.Block)
		generator = miners[0]
	)
	b.Round = round                               // VC round
	b.MinerID = generator.miner.id                // block generator
	b.PrevBlock = block.Provider().(*block.Block) // stub
	now += 10
	var tx = newTransaction(generator.miner.id, ADDRESS, 0, now)
	balances.txn = tx
	balances.block = b
	balances.blockSharders = extractBlockSharders(sharders, 2)
}
