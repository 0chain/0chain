package storagesc

import (
	"testing"
	"time"

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

func Test_getConfig(t *testing.T) {
	var (
		ssc        = newTestStorageSC()
		balances   = newTestBalances(t, false)
		configured = setConfig(t, balances)
		gn, _      = ssc.getConfig(balances, false)
	)
	assert.EqualValues(t, configured, gn)
}

func TestMinerSmartContractUpdate(t *testing.T) {
	var (
		ssc            = newTestStorageSC()
		balances       = newTestBalances(t, false)
		tx             = newTransaction(owner, ssc.ID, 0, 0)
		originalConfig = setConfig(t, balances)
		err            error
		config         = new(scConfig)
		update         = new(scConfig)
	)

	balances.txn = tx

	// 1. Malformed update
	t.Run("malformed update", func(t *testing.T) {
		_, err = ssc.updateConfig(tx, []byte("} malformed {"), balances)
		assertErrMsg(t, err, "update_config: invalid character '}' looking for beginning of value")
	})

	// 2. Non owner account tries to update
	t.Run("non owner account", func(t *testing.T) {
		tx.ClientID = randString(32)
		_, err = ssc.updateConfig(tx, []byte("} malformed {"), balances)
		assertErrMsg(t, err, "update_config: unauthorized access - only the owner can update the variables")
	})

	// 3. All variables requested shall be denied
	t.Run("all variables denied", func(t *testing.T) {
		update.ChallengeEnabled = true
		tx.ClientID = owner
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config, originalConfig)
	})

	// 4. Time unit will update
	t.Run("time unit update", func(t *testing.T) {
		update.TimeUnit = time.Second * 5
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.TimeUnit, update.TimeUnit)
	})

	// 5. Max mint will update
	t.Run("max mint update", func(t *testing.T) {
		update.MaxMint = 123456789
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.MaxMint, update.MaxMint)
	})

	// 6. Min allocation size will update
	t.Run("min allocation size update", func(t *testing.T) {
		update.MinAllocSize = 1024
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.MinAllocSize, update.MinAllocSize)
	})

	// 7. Min allocation duration will update
	t.Run("min allocation duration update", func(t *testing.T) {
		update.MinAllocDuration = time.Hour * 7
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.MinAllocDuration, update.MinAllocDuration)
	})

	// 8. Max challenge completion time will update
	t.Run("max challenge completion time update", func(t *testing.T) {
		update.MaxChallengeCompletionTime = time.Minute * 33
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.MaxChallengeCompletionTime, update.MaxChallengeCompletionTime)
	})

	// 9. Min offer duration will update
	t.Run("min offer duration update", func(t *testing.T) {
		update.MinOfferDuration = time.Hour * 72
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.MinOfferDuration, update.MinOfferDuration)
	})

	// 10. Min blobber capacity will update
	t.Run("min blobber capacity update", func(t *testing.T) {
		update.MinBlobberCapacity = 2048
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.MinBlobberCapacity, update.MinBlobberCapacity)
	})

	// 11. Validator reward will update
	t.Run("validator reward update", func(t *testing.T) {
		update.ValidatorReward = .19
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.ValidatorReward, update.ValidatorReward)
	})

	// 12. Blobber slash will update
	t.Run("blobber slash update", func(t *testing.T) {
		update.BlobberSlash = .81
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.BlobberSlash, update.BlobberSlash)
	})

	// 13. Max read price will update
	t.Run("max read price update", func(t *testing.T) {
		update.MaxReadPrice = 13579
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.MaxReadPrice, update.MaxReadPrice)
	})

	// 14. Max write price will update
	t.Run("max write price update", func(t *testing.T) {
		update.MaxWritePrice = 35791
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.MaxWritePrice, update.MaxWritePrice)
	})

	// 15. Failed challenges to cancel will update
	t.Run("failed challenges to cancel update", func(t *testing.T) {
		update.FailedChallengesToCancel = 17
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.FailedChallengesToCancel, update.FailedChallengesToCancel)
	})

	// 16. Failed challenges to revoke min lock will update
	t.Run("failed challenges to revoke min lock update", func(t *testing.T) {
		update.FailedChallengesToRevokeMinLock = 21
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.FailedChallengesToRevokeMinLock, update.FailedChallengesToRevokeMinLock)
	})

	// 17. Challenge enabled will update
	t.Run("challenge enabled update", func(t *testing.T) {
		update.ChallengeEnabled = false
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.ChallengeEnabled, update.ChallengeEnabled)
	})

	// 18. Max challenges per generation will update
	t.Run("max challenges per generation update", func(t *testing.T) {
		update.MaxChallengesPerGeneration = 51
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.MaxChallengesPerGeneration, update.MaxChallengesPerGeneration)
	})

	// 19. Challenge generation rate will update
	t.Run("challenge generation update", func(t *testing.T) {
		update.ChallengeGenerationRate = 0.77
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.ChallengeGenerationRate, update.ChallengeGenerationRate)
	})

	// 20. Min stake will update
	t.Run("min stake update", func(t *testing.T) {
		update.MinStake = 3000000000
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.MinStake, update.MinStake)
	})

	// 21. Max stake will update
	t.Run("max stake update", func(t *testing.T) {
		update.MaxStake = 3000000000000
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.MaxStake, update.MaxStake)
	})

	// 22. Max delegates will update
	t.Run("max delegates update", func(t *testing.T) {
		update.MaxDelegates = 99
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.MaxDelegates, update.MaxDelegates)
	})

	// 23. Max charge will update
	t.Run("max charge update", func(t *testing.T) {
		update.MaxCharge = .87
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.MaxCharge, update.MaxCharge)
	})

	// 24. Read pool config won't update
	t.Run("read pool config fail", func(t *testing.T) {
		update.ReadPool = &readPoolConfig{}
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.ReadPool, originalConfig.ReadPool)
	})

	// 25. Read pool config min lock will update
	t.Run("read pool config min lock update", func(t *testing.T) {
		update.ReadPool.MinLock = 7000000000
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.ReadPool.MinLock, update.ReadPool.MinLock)
	})

	// 26. Read pool config min lock period will update
	t.Run("read pool config min lock period update", func(t *testing.T) {
		update.ReadPool.MinLockPeriod = time.Hour * 24
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.ReadPool.MinLockPeriod, update.ReadPool.MinLockPeriod)
	})

	// 27. Read pool config max lock period will update
	t.Run("read pool config max lock period update", func(t *testing.T) {
		update.ReadPool.MaxLockPeriod = time.Hour * 240
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.ReadPool.MaxLockPeriod, update.ReadPool.MaxLockPeriod)
	})

	// 28. Write pool config won't update
	t.Run("write pool config fail", func(t *testing.T) {
		update.WritePool = &writePoolConfig{}
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.WritePool, originalConfig.WritePool)
	})

	// 29. Write pool config min lock will update
	t.Run("write pool config min lock update", func(t *testing.T) {
		update.WritePool.MinLock = 9100000000
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.WritePool.MinLock, update.WritePool.MinLock)
	})

	// 30. Write pool config min lock period will update
	t.Run("write pool config min lock period update", func(t *testing.T) {
		update.WritePool.MinLockPeriod = time.Hour * 36
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.WritePool.MinLockPeriod, update.WritePool.MinLockPeriod)
	})

	// 31. Write pool config max lock period will update
	t.Run("write pool config max lock period update", func(t *testing.T) {
		update.WritePool.MaxLockPeriod = time.Hour * 360
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.WritePool.MaxLockPeriod, update.WritePool.MaxLockPeriod)
	})

	// 32. Stake pool config won't update
	t.Run("read pool config fail", func(t *testing.T) {
		update.StakePool = &stakePoolConfig{}
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.StakePool, originalConfig.StakePool)
	})

	// 33. Stake pool config min lock will update
	t.Run("write pool config min lock update", func(t *testing.T) {
		update.StakePool.MinLock = 17700000000
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.StakePool.MinLock, update.StakePool.MinLock)
	})

	// 34. Stake pool config interest rate will update
	t.Run("write pool config interest rate update", func(t *testing.T) {
		update.StakePool.InterestRate = 0.09
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.StakePool.InterestRate, update.StakePool.InterestRate)
	})

	// 35. Stake pool config interest interval will update
	t.Run("write pool config interest interval update", func(t *testing.T) {
		update.StakePool.InterestInterval = time.Hour * 8760
		_, err = ssc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		config, err = ssc.getConfig(balances, false)
		require.NoError(t, err)
		assert.EqualValues(t, config.StakePool.InterestInterval, update.StakePool.InterestInterval)
	})
}
