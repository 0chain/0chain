package minersc

import (
	"testing"

	configpkg "0chain.net/chaincore/config"

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

func Test_getConfig(t *testing.T) {
	var (
		msc        = newTestMinerSC()
		balances   = newTestBalances()
		configured = getGlobalNodeTest()
		gn, _      = msc.getGlobalNode(balances)
	)
	assert.EqualValues(t, configured, gn)
}

func TestMinerSmartContractUpdate(t *testing.T) {
	var (
		msc        = newTestMinerSC()
		balances   = newTestBalances()
		tx         = newTransaction(owner, msc.ID, 0, 0)
		originalGn = getGlobalNodeTest()
		gn, err    = msc.getGlobalNode(balances)
		update     = &GlobalNode{}
	)

	balances.txn = tx

	// 1. Malformed update
	t.Run("malformed update", func(t *testing.T) {
		_, err = msc.UpdateSettings(tx, []byte("} malformed {"), gn, balances)
		assertErrMsg(t, err, "failed to update smart contract settings: error decoding input data: invalid character '}' looking for beginning of value")
	})

	// 2. Non owner account tries to update
	t.Run("non owner account", func(t *testing.T) {
		tx.ClientID = randString(32)
		_, err = msc.UpdateSettings(tx, []byte("} malformed {"), gn, balances)
		assertErrMsg(t, err, "failed to update smart contract settings: unauthorized access - only the owner can update the settings")
	})

	// 3. All variables requested shall be denied
	t.Run("all variables denied", func(t *testing.T) {
		tx.ClientID = owner
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn, originalGn)
	})

	// 4. Max N will updated
	t.Run("max n update", func(t *testing.T) {
		update.MaxN = 99
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.MaxN, update.MaxN)
	})

	// 5. Min N will updated
	t.Run("min n update", func(t *testing.T) {
		update.MinN = 11
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.MinN, update.MinN)
	})

	// 6. Max S will updated
	t.Run("max s update", func(t *testing.T) {
		update.MaxS = 22
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.MaxS, update.MaxS)
	})

	// 7. Min S will updated
	t.Run("min s update", func(t *testing.T) {
		update.MinS = 9
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.MinS, update.MinS)
	})

	// 8. Max delegates will updated
	t.Run("max delegates update", func(t *testing.T) {
		update.MaxDelegates = 111
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.MaxDelegates, update.MaxDelegates)
	})

	// 9. T percent will updated
	t.Run("t percent update", func(t *testing.T) {
		update.TPercent = .57
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.TPercent, update.TPercent)
	})

	// 10. K percent will updated
	t.Run("k percent update", func(t *testing.T) {
		update.KPercent = .81
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.KPercent, update.KPercent)
	})

	// 11. Max stake will updated
	t.Run("max stake update", func(t *testing.T) {
		update.MaxStake = 987654321
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.MaxStake, update.MaxStake)
	})

	// 12. Min stake will updated
	t.Run("min stake update", func(t *testing.T) {
		update.MinStake = 123456
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.MinStake, update.MinStake)
	})

	// 13. Interest rate will updated
	t.Run("interest rate update", func(t *testing.T) {
		update.InterestRate = .09
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.InterestRate, update.InterestRate)
	})

	// 14. Reward rate will updated
	t.Run("reward rate update", func(t *testing.T) {
		update.RewardRate = .55
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.RewardRate, update.RewardRate)
	})

	// 15. Share ratio will updated
	t.Run("share ratio update", func(t *testing.T) {
		update.ShareRatio = .39
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.ShareRatio, update.ShareRatio)
	})

	// 16. Block reward will updated
	t.Run("block reward update", func(t *testing.T) {
		update.BlockReward = 757575
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.BlockReward, update.BlockReward)
	})

	// 17. Max charge will updated
	t.Run("max charge update", func(t *testing.T) {
		update.MaxCharge = .47
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.MaxCharge, update.MaxCharge)
	})

	// 18. Epoch will updated
	t.Run("epoch update", func(t *testing.T) {
		update.Epoch = 90000
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.Epoch, update.Epoch)
	})

	// 19. Reward decline rate will updated
	t.Run("reward decline rate update", func(t *testing.T) {
		update.RewardDeclineRate = .66
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.RewardDeclineRate, update.RewardDeclineRate)
	})

	// 20. Interest decline rate will updated
	t.Run("interest decline rate update", func(t *testing.T) {
		update.InterestDeclineRate = .47
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.InterestDeclineRate, update.InterestDeclineRate)
	})

	// 21. Max mint will updated
	t.Run("max mint update", func(t *testing.T) {
		update.MaxMint = 7531
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.MaxMint, update.MaxMint)
	})

	// 22. Reward round frequency will updated
	t.Run("reward round frequency update", func(t *testing.T) {
		update.RewardRoundFrequency = 300
		_, err = msc.UpdateSettings(tx, mustEncode(t, update), gn, balances)
		require.NoError(t, err)
		gn, err = msc.getGlobalNode(balances)
		require.NoError(t, err)
		assert.EqualValues(t, gn.RewardRoundFrequency, update.RewardRoundFrequency)
	})
}
