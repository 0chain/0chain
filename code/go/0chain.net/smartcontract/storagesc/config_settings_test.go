package storagesc

import (
	"testing"
	"time"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSettings(t *testing.T) {
	require.Len(t, SettingName, int(NumberOfSettings))
	require.Len(t, Settings, int(NumberOfSettings))
	require.Len(t, ConfitTypeName, int(NumberOfTypes))

	for _, name := range SettingName {
		require.EqualValues(t, name, SettingName[Settings[name].setting])
	}
}

func TestUpdateConfig(t *testing.T) {
	type args struct {
		ssc      *StorageSmartContract
		txn      *transaction.Transaction
		input    []byte
		balances chainstate.StateContextI
	}

	type parameters struct {
		client       string
		inputMap     map[string]interface{}
		TargetConfig scConfig
	}

	setExpectations := func(t *testing.T, p parameters) args {
		var balances = &mocks.StateContextI{}
		var ssc = &StorageSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}
		var txn = &transaction.Transaction{
			ClientID: p.client,
		}

		balances.On("GetTrieNode", scConfigKey(ssc.ID)).Return(&scConfig{}, nil).Once()
		balances.On(
			"InsertTrieNode",
			scConfigKey(ssc.ID),
			mock.MatchedBy(func(conf *scConfig) bool {
				for key, value := range p.inputMap {
					if getConfField(*conf, key) != value {
						return false
					}
				}
				return true
			}),
		).Return("", nil).Once()

		return args{
			ssc:      ssc,
			txn:      txn,
			input:    (&inputMap{p.inputMap}).Encode(),
			balances: balances,
		}
	}

	type want struct {
		error bool
		msg   string
	}

	testCases := []struct {
		title      string
		parameters parameters
		want       want
	}{
		{
			title: "all_settigns",
			parameters: parameters{
				client: owner,
				inputMap: map[string]interface{}{
					"max_mint":                      zcnToBalance(1500000.0),
					"time_unit":                     720 * time.Hour,
					"min_alloc_size":                int64(1024),
					"min_alloc_duration":            5 * time.Minute,
					"max_challenge_completion_time": 30 * time.Minute,
					"min_offer_duration":            10 * time.Hour,
					"min_blobber_capacity":          int64(1024),

					"readpool.min_lock":        int64(10),
					"readpool.min_lock_period": 1 * time.Hour,
					"readpool.max_lock_period": 8760 * time.Hour,

					"writepool.min_lock":        int64(10),
					"writepool.min_lock_period": 2 * time.Minute,
					"writepool.max_lock_period": 8760 * time.Hour,

					"stakepool.min_lock":          int64(10),
					"stakepool.interest_rate":     float64(0.0),
					"stakepool.interest_interval": 1 * time.Minute,

					"max_total_free_allocation":      zcnToBalance(10000),
					"max_individual_free_allocation": zcnToBalance(100),

					"free_allocation_settings.data_shards":                   int(10),
					"free_allocation_settings.parity_shards":                 int(5),
					"free_allocation_settings.size":                          int64(10000000000),
					"free_allocation_settings.duration":                      5000 * time.Hour,
					"free_allocation_settings.read_price_range.min":          zcnToBalance(0.0),
					"free_allocation_settings.read_price_range.max":          zcnToBalance(0.04),
					"free_allocation_settings.write_price_range.min":         zcnToBalance(0.0),
					"free_allocation_settings.write_price_range.max":         zcnToBalance(0.1),
					"free_allocation_settings.max_challenge_completion_time": 1 * time.Minute,
					"free_allocation_settings.read_pool_fraction":            float64(0.2),

					"validator_reward":                     float64(0.025),
					"blobber_slash":                        float64(0.1),
					"max_read_price":                       zcnToBalance(100),
					"max_write_price":                      zcnToBalance(100),
					"failed_challenges_to_cancel":          int(20),
					"failed_challenges_to_revoke_min_lock": int(10),
					"challenge_enabled":                    true,
					"challenge_rate_per_mb_min":            float64(1.0),
					"max_challenges_per_generation":        int(100),
					"max_delegates":                        int(100),

					"block_reward.block_reward":           zcnToBalance(1000),
					"block_reward.qualifying_stake":       zcnToBalance(1),
					"block_reward.sharder_ratio":          float64(80),
					"block_reward.miner_ratio":            float64(20),
					"block_reward.blobber_capacity_ratio": float64(20),
					"block_reward.blobber_usage_ratio":    float64(80),

					"expose_mpt": false,
				},
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.title, func(t *testing.T) {
			test := test
			t.Parallel()
			args := setExpectations(t, test.parameters)

			_, err := args.ssc.updateSettings(args.txn, args.input, args.balances)
			require.EqualValues(t, test.want.error, err != nil)
			if err != nil {
				require.EqualValues(t, test.want.msg, err.Error())
				return
			}
			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
		})
	}
}

func getConfField(conf scConfig, field string) interface{} {
	switch Settings[field].setting {
	case MaxMint:
		return conf.MaxMint
	case TimeUnit:
		return conf.TimeUnit
	case MinAllocSize:
		return conf.MinAllocSize
	case MinAllocDuration:
		return conf.MinAllocDuration
	case MaxChallengeCompletionTime:
		return conf.MaxChallengeCompletionTime
	case MinOfferDuration:
		return conf.MinOfferDuration
	case MinBlobberCapacity:
		return conf.MinBlobberCapacity

	case ReadPoolMinLock:
		return conf.ReadPool.MinLock
	case ReadPoolMinLockPeriod:
		return conf.ReadPool.MinLockPeriod
	case ReadPoolMaxLockPeriod:
		return conf.ReadPool.MaxLockPeriod

	case WritePoolMinLock:
		return conf.WritePool.MinLock
	case WritePoolMinLockPeriod:
		return conf.WritePool.MinLockPeriod
	case WritePoolMaxLockPeriod:
		return conf.WritePool.MaxLockPeriod

	case StakePoolMinLock:
		return conf.StakePool.MinLock
	case StakePoolInterestRate:
		return conf.StakePool.InterestRate
	case StakePoolInterestInterval:
		return conf.StakePool.InterestInterval

	case MaxTotalFreeAllocation:
		return conf.MaxTotalFreeAllocation
	case MaxIndividualFreeAllocation:
		return conf.MaxIndividualFreeAllocation

	case FreeAllocationDataShards:
		return conf.FreeAllocationSettings.DataShards
	case FreeAllocationParityShards:
		return conf.FreeAllocationSettings.ParityShards
	case FreeAllocationSize:
		return conf.FreeAllocationSettings.Size
	case FreeAllocationDuration:
		return conf.FreeAllocationSettings.Duration
	case FreeAllocationReadPriceRangeMin:
		return conf.FreeAllocationSettings.ReadPriceRange.Min
	case FreeAllocationReadPriceRangeMax:
		return conf.FreeAllocationSettings.ReadPriceRange.Max
	case FreeAllocationWritePriceRangeMin:
		return conf.FreeAllocationSettings.WritePriceRange.Min
	case FreeAllocationWritePriceRangeMax:
		return conf.FreeAllocationSettings.WritePriceRange.Max
	case FreeAllocationMaxChallengeCompletionTime:
		return conf.FreeAllocationSettings.MaxChallengeCompletionTime
	case FreeAllocationReadPoolFraction:
		return conf.FreeAllocationSettings.ReadPoolFraction

	case ValidatorReward:
		return conf.ValidatorReward
	case BlobberSlash:
		return conf.BlobberSlash
	case MaxReadPrice:
		return conf.MaxReadPrice
	case MaxWritePrice:
		return conf.MaxWritePrice
	case FailedChallengesToCancel:
		return conf.FailedChallengesToCancel
	case FailedChallengesToRevokeMinLock:
		return conf.FailedChallengesToRevokeMinLock
	case ChallengeEnabled:
		return conf.ChallengeEnabled
	case ChallengeGenerationRate:
		return conf.ChallengeGenerationRate
	case MaxChallengesPerGeneration:
		return conf.MaxChallengesPerGeneration
	case MaxDelegates:
		return conf.MaxDelegates
	case BlockRewardBlockReward:
		return conf.BlockReward.BlockReward
	case BlockRewardQualifyingStake:
		return conf.BlockReward.QualifyingStake
	case BlockRewardSharderWeight:
		return conf.BlockReward.SharderWeight
	case BlockRewardMinerWeight:
		return conf.BlockReward.MinerWeight
	case BlockRewardBlobberCapacityWeight:
		return conf.BlockReward.BlobberCapacityWeight
	case BlockRewardBlobberUsageWeight:
		return conf.BlockReward.BlobberUsageWeight

	case ExposeMpt:
		return conf.ExposeMpt
	default:
		panic("unknown field: " + field)
	}
}
