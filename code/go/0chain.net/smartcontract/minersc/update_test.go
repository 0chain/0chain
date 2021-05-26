package minersc

import (
	"testing"

	configpkg "0chain.net/chaincore/config"
	"0chain.net/chaincore/transaction"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertErrMsg(t *testing.T, err error, msg string) {
	t.Helper()

	if msg == "" {
		assert.Nil(t, err)
		return
	}

	if assert.NotNil(t, err) {
		assert.Equal(t, msg, err.Error())
	}
}

func getGlobalNodeTest() (gn *GlobalNode) {
	const pfx = "smart_contracts.minersc."

	configpkg.SmartContractConfig.Set(pfx+"max_stake", 1000)
	configpkg.SmartContractConfig.Set(pfx+"min_stake", 1)
	configpkg.SmartContractConfig.Set(pfx+"max_n", 50)
	configpkg.SmartContractConfig.Set(pfx+"min_n", 2)
	configpkg.SmartContractConfig.Set(pfx+"t_percent", .51)
	configpkg.SmartContractConfig.Set(pfx+"k_percent", .75)
	configpkg.SmartContractConfig.Set(pfx+"max_s", 30)
	configpkg.SmartContractConfig.Set(pfx+"min_s", 1)
	configpkg.SmartContractConfig.Set(pfx+"max_delegates", 20)
	configpkg.SmartContractConfig.Set(pfx+"reward_round_frequency", 250)
	configpkg.SmartContractConfig.Set(pfx+"interest_rate", .1)
	configpkg.SmartContractConfig.Set(pfx+"reward_rate", .11)
	configpkg.SmartContractConfig.Set(pfx+"share_ratio", .27)
	configpkg.SmartContractConfig.Set(pfx+"block_reward", 3000)
	configpkg.SmartContractConfig.Set(pfx+"max_charge", .44)
	configpkg.SmartContractConfig.Set(pfx+"epoch", 10000)
	configpkg.SmartContractConfig.Set(pfx+"reward_decline_rate", .33)
	configpkg.SmartContractConfig.Set(pfx+"interest_decline_rate", .34)
	configpkg.SmartContractConfig.Set(pfx+"max_mint", 3)

	return &GlobalNode{
		0, 50, 2, 30, 1, 20, .51, .75, 0, 1000 * 1e10, 1e10,
		.1, .11, .27, 3000 * 1e10, .44, 10000, .33, .34, 3 * 1e10,
		nil, 0, 250,
	}
}

func TestMinerSmartContractUpdate(t *testing.T) {
	var (
		msc         = newTestMinerSC()
		balances    = newTestBalances()
		ownerTxn    = newTransaction(owner, msc.ID, 0, 0)
		nonOwnerTxn = newTransaction(randString(32), msc.ID, 0, 0)

		originalGn = getGlobalNodeTest()
		gn, err    = msc.getGlobalNode(balances)
	)

	//test cases that produce errors
	errorTestCases := []struct {
		title string
		txn   *transaction.Transaction
		bytes []byte
		err   string
	}{
		{"malformed update", ownerTxn, []byte("} malformed {"), "failed to update smart contract settings: error decoding input data: invalid character '}' looking for beginning of value"},
		{"non owner account", nonOwnerTxn, []byte("} malformed {"), "failed to update smart contract settings: unauthorized access - only the owner can update the settings"},
	}
	for _, tc := range errorTestCases {
		t.Run(tc.title, func(t *testing.T) {
			balances.txn = tc.txn
			_, err = msc.UpdateSettings(tc.txn, tc.bytes, gn, balances)
			require.Error(t, err)
			require.EqualError(t, err, tc.err)
		})
	}

	balances.txn = ownerTxn

	//test cases that will be denied
	deniedTestCases := []struct {
		title       string
		request     *GlobalNode
		requireFunc func(gn, originalGn *GlobalNode)
	}{
		{"all variables denied",
			&GlobalNode{},
			func(gn, originalGn *GlobalNode) {
				require.Equal(t, gn, originalGn)
			},
		},
	}
	for _, tc := range deniedTestCases {
		t.Run(tc.title, func(t *testing.T) {
			_, err = msc.UpdateSettings(ownerTxn, tc.request.Encode(), gn, balances)
			tc.requireFunc(gn, originalGn)
		})
	}

	updateTestCases := []struct {
		title       string
		request     *GlobalNode
		requireFunc func(gn *GlobalNode, request *GlobalNode)
	}{
		{
			"max n update",
			&GlobalNode{
				MaxN: 99,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.MaxN, request.MaxN)
			},
		},
		{
			"min n update",
			&GlobalNode{
				MinN: 11,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.MinN, request.MinN)
			},
		},
		{
			"max s update",
			&GlobalNode{
				MaxS: 22,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.MaxS, request.MaxS)
			},
		},
		{
			"min s update",
			&GlobalNode{
				MinS: 19,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.MinS, request.MinS)
			},
		},
		{
			"max delegates update",
			&GlobalNode{
				MaxDelegates: 111,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.MaxDelegates, request.MaxDelegates)
			},
		},
		{
			"t percent update",
			&GlobalNode{
				TPercent: 0.57,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.TPercent, request.TPercent)
			},
		},
		{
			"k percent update",
			&GlobalNode{
				KPercent: 0.81,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.KPercent, request.KPercent)
			},
		},
		{
			"max stake update",
			&GlobalNode{
				MaxStake: 987654321,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.MaxStake, request.MaxStake)
			},
		},
		{
			"min stake update",
			&GlobalNode{
				MinStake: 123456,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.MinStake, request.MinStake)
			},
		},
		{
			"interest rate update",
			&GlobalNode{
				InterestRate: 0.09,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.InterestRate, request.InterestRate)
			},
		},
		{
			"reward rate update",
			&GlobalNode{
				RewardRate: 0.55,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.RewardRate, request.RewardRate)
			},
		},
		{
			"share ratio update",
			&GlobalNode{
				ShareRatio: 0.39,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.ShareRatio, request.ShareRatio)
			},
		},
		{
			"block reward update",
			&GlobalNode{
				BlockReward: 757575,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.BlockReward, request.BlockReward)
			},
		},
		{
			"max charge update",
			&GlobalNode{
				MaxCharge: 0.47,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.MaxCharge, request.MaxCharge)
			},
		},
		{
			"epoch update",
			&GlobalNode{
				Epoch: 90000,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.Epoch, request.Epoch)
			},
		},
		{
			"reward decline rate update",
			&GlobalNode{
				RewardDeclineRate: 0.66,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.RewardDeclineRate, request.RewardDeclineRate)
			},
		},
		{
			"interest decline rate update",
			&GlobalNode{
				InterestDeclineRate: 0.47,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.InterestDeclineRate, request.InterestDeclineRate)
			},
		},
		{
			"max mint update",
			&GlobalNode{
				MaxMint: 7531,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.MaxMint, request.MaxMint)
			},
		},
		{
			"reward round frequency update",
			&GlobalNode{
				RewardRoundFrequency: 300,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.RewardRoundFrequency, request.RewardRoundFrequency)
			},
		},
	}
	for _, tc := range updateTestCases {
		t.Run(tc.title, func(t *testing.T) {
			balances.txn = ownerTxn
			_, err = msc.UpdateSettings(ownerTxn, tc.request.Encode(), gn, balances)
			require.NoError(t, err)
			gn, err = msc.getGlobalNode(balances)
			require.NoError(t, err)
			tc.requireFunc(gn, tc.request)
		})
	}
}
