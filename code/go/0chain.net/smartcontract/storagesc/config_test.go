package storagesc

import (
	"testing"
	"time"

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
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)
		tx       = newTransaction(owner, ssc.ID, 0, 0)

		ownerTxn    = newTransaction(owner, ssc.ID, 0, 0)
		nonOwnerTxn = newTransaction(randString(32), ssc.ID, 0, 0)

		originalConfig = setConfig(t, balances)
		err            error
		config         = new(scConfig)
	)

	balances.txn = tx

	//test cases that produce errors
	errorTestCases := []struct {
		title string
		txn   *transaction.Transaction
		bytes []byte
		err   string
	}{
		{"malformed update", ownerTxn, []byte("} malformed {"), "update_config: invalid character '}' looking for beginning of value"},
		{"non owner account", nonOwnerTxn, []byte("} malformed {"), "update_config: unauthorized access - only the owner can update the variables"},
	}
	for _, tc := range errorTestCases {
		t.Run(tc.title, func(t *testing.T) {
			balances.txn = tc.txn
			_, err = ssc.updateConfig(tc.txn, tc.bytes, balances)
			require.Error(t, err)
			require.EqualError(t, err, tc.err)
		})
	}

	balances.txn = ownerTxn

	//test cases that will be denied
	deniedTestCases := []struct {
		title       string
		request     *scConfig
		requireFunc func(config, request *scConfig)
	}{
		{"all variables denied",
			&scConfig{
				ChallengeEnabled: true,
			},
			func(config, request *scConfig) {
				require.Equal(t, config, request)
			},
		},
		{"read pool config fail",
			&scConfig{
				ReadPool: &readPoolConfig{},
			},
			func(config, request *scConfig) {
				require.Equal(t, config.ReadPool, request.ReadPool)
			},
		},
		{"write pool config fail",
			&scConfig{
				WritePool: &writePoolConfig{},
			},
			func(config, request *scConfig) {
				require.Equal(t, config.WritePool, request.WritePool)
			},
		},
		{"stake pool config fail",
			&scConfig{
				StakePool: &stakePoolConfig{},
			},
			func(config, request *scConfig) {
				require.Equal(t, config.StakePool, request.StakePool)
			},
		},
	}
	for _, tc := range deniedTestCases {
		t.Run(tc.title, func(t *testing.T) {
			_, err = ssc.updateConfig(ownerTxn, tc.request.Encode(), balances)
			require.NoError(t, err)
			config, err = ssc.getConfig(balances, false)
			require.NoError(t, err)
			tc.requireFunc(config, originalConfig)
		})
	}

	updateTestCases := []struct {
		title       string
		request     *scConfig
		requireFunc func(config, request *scConfig)
	}{
		{
			"time unit update",
			&scConfig{
				TimeUnit: time.Second * 5,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.TimeUnit, request.TimeUnit)
			},
		},
		{
			"max mint update",
			&scConfig{
				MaxMint: 123456789,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.MaxMint, request.MaxMint)
			},
		},
		{
			"min allocation size update",
			&scConfig{
				MinAllocSize: 1024,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.MinAllocSize, request.MinAllocSize)
			},
		},
		{
			"min allocation duration update",
			&scConfig{
				MinAllocDuration: time.Hour * 7,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.MinAllocDuration, request.MinAllocDuration)
			},
		},
		{
			"max challenge completion time update",
			&scConfig{
				MaxChallengeCompletionTime: time.Minute * 33,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.MaxChallengeCompletionTime, request.MaxChallengeCompletionTime)
			},
		},
		{
			"min offer duration update",
			&scConfig{
				MinOfferDuration: time.Hour * 72,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.MinOfferDuration, request.MinOfferDuration)
			},
		},
		{
			"min blobber capacity update",
			&scConfig{
				MinBlobberCapacity: 2048,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.MinBlobberCapacity, request.MinBlobberCapacity)
			},
		},
		{
			"validator reward update",
			&scConfig{
				ValidatorReward: 0.19,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.ValidatorReward, request.ValidatorReward)
			},
		},
		{
			"blobber slash update",
			&scConfig{
				BlobberSlash: 0.81,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.BlobberSlash, request.BlobberSlash)
			},
		},
		{
			"max read price update",
			&scConfig{
				MaxReadPrice: 13579,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.MaxReadPrice, request.MaxReadPrice)
			},
		},
		{
			"max write price update",
			&scConfig{
				MaxWritePrice: 35791,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.MaxWritePrice, request.MaxWritePrice)
			},
		},
		{
			"failed challenges to cancel update",
			&scConfig{
				FailedChallengesToCancel: 17,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.FailedChallengesToCancel, request.FailedChallengesToCancel)
			},
		},
		{
			"failed challenges to revoke min lock update",
			&scConfig{
				FailedChallengesToRevokeMinLock: 21,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.FailedChallengesToRevokeMinLock, request.FailedChallengesToRevokeMinLock)
			},
		},
		{
			"challenge enabled update",
			&scConfig{
				ChallengeEnabled: false,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.ChallengeEnabled, request.ChallengeEnabled)
			},
		},
		{
			"max challenges per generation update",
			&scConfig{
				MaxChallengesPerGeneration: 51,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.MaxChallengesPerGeneration, request.MaxChallengesPerGeneration)
			},
		},
		{
			"challenge generation update",
			&scConfig{
				ChallengeGenerationRate: 0.77,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.ChallengeGenerationRate, request.ChallengeGenerationRate)
			},
		},
		{
			"min stake update",
			&scConfig{
				MinStake: 3000000000,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.MinStake, request.MinStake)
			},
		},
		{
			"max stake update",
			&scConfig{
				MaxStake: 3000000000000,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.MaxStake, request.MaxStake)
			},
		},
		{
			"max delegates update",
			&scConfig{
				MaxDelegates: 99,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.MaxDelegates, request.MaxDelegates)
			},
		},
		{
			"max charge update",
			&scConfig{
				MaxCharge: 0.87,
			},
			func(config, request *scConfig) {
				require.Equal(t, config.MaxCharge, request.MaxCharge)
			},
		},
		{
			"read pool config min lock update",
			&scConfig{
				ReadPool: &readPoolConfig{
					MinLock: 7000000000,
				},
			},
			func(config, request *scConfig) {
				require.Equal(t, config.ReadPool.MinLock, request.ReadPool.MinLock)
			},
		},
		{
			"read pool config min lock period update",
			&scConfig{
				ReadPool: &readPoolConfig{
					MinLockPeriod: time.Hour * 24,
				},
			},
			func(config, request *scConfig) {
				require.Equal(t, config.ReadPool.MinLockPeriod, request.ReadPool.MinLockPeriod)
			},
		},
		{
			"read pool config max lock period update",
			&scConfig{
				ReadPool: &readPoolConfig{
					MaxLockPeriod: time.Hour * 240,
				},
			},
			func(config, request *scConfig) {
				require.Equal(t, config.ReadPool.MaxLockPeriod, request.ReadPool.MaxLockPeriod)
			},
		},
		{
			"write pool config min lock update",
			&scConfig{
				WritePool: &writePoolConfig{
					MinLock: 9100000000,
				},
			},
			func(config, request *scConfig) {
				require.Equal(t, config.WritePool.MinLock, request.WritePool.MinLock)
			},
		},
		{
			"write pool config min lock period update",
			&scConfig{
				WritePool: &writePoolConfig{
					MinLockPeriod: time.Hour * 36,
				},
			},
			func(config, request *scConfig) {
				require.Equal(t, config.WritePool.MinLockPeriod, request.WritePool.MinLockPeriod)
			},
		},
		{
			"write pool config max lock period update",
			&scConfig{
				WritePool: &writePoolConfig{
					MaxLockPeriod: time.Hour * 360,
				},
			},
			func(config, request *scConfig) {
				require.Equal(t, config.WritePool.MaxLockPeriod, request.WritePool.MaxLockPeriod)
			},
		},
		{
			"stake pool config min lock update",
			&scConfig{
				StakePool: &stakePoolConfig{
					MinLock: 17700000000,
				},
			},
			func(config, request *scConfig) {
				require.Equal(t, config.StakePool.MinLock, request.StakePool.MinLock)
			},
		},
		{
			"stake pool config interest rate update",
			&scConfig{
				StakePool: &stakePoolConfig{
					InterestRate: 0.09,
				},
			},
			func(config, request *scConfig) {
				require.Equal(t, config.StakePool.InterestRate, request.StakePool.InterestRate)
			},
		},
		{
			"stake pool config interest interval update",
			&scConfig{
				StakePool: &stakePoolConfig{
					InterestInterval: time.Hour * 8760,
				},
			},
			func(config, request *scConfig) {
				require.Equal(t, config.StakePool.InterestInterval, request.StakePool.InterestInterval)
			},
		},
	}
	for _, tc := range updateTestCases {
		t.Run(tc.title, func(t *testing.T) {
			balances.txn = ownerTxn
			_, err = ssc.updateConfig(ownerTxn, tc.request.Encode(), balances)
			require.NoError(t, err)
			config, err = ssc.getConfig(balances, false)
			require.NoError(t, err)
			tc.requireFunc(config, tc.request)
		})
	}
}
