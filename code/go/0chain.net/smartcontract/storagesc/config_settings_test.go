package storagesc

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"0chain.net/chaincore/block"

	"0chain.net/chaincore/state"

	"0chain.net/smartcontract"

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

	for _, name := range SettingName {
		require.EqualValues(t, name, SettingName[Settings[name].setting])
	}
}

func TestUpdateSettings(t *testing.T) {
	type args struct {
		ssc      *StorageSmartContract
		txn      *transaction.Transaction
		input    []byte
		balances chainstate.StateContextI
	}

	type parameters struct {
		client                string
		previousMap, inputMap map[string]string
		TargetConfig          scConfig
	}

	setExpectations := func(t *testing.T, p parameters) args {
		var balances = &mocks.StateContextI{}
		var ssc = &StorageSmartContract{

			SmartContract: sci.NewSC(ADDRESS),
		}
		var txn = &transaction.Transaction{
			ClientID: p.client,
		}

		var oldChanges smartcontract.StringMap
		oldChanges.Fields = p.previousMap
		balances.On("GetTrieNode", settingChangesKey).Return(&oldChanges, nil).Once()

		for key, value := range p.inputMap {
			oldChanges.Fields[key] = value
		}

		var expected = smartcontract.NewStringMap()
		for key, value := range p.previousMap {
			expected.Fields[key] = value
		}
		for key, value := range p.inputMap {
			expected.Fields[key] = value
		}

		balances.On(
			"InsertTrieNode",
			settingChangesKey,
			mock.MatchedBy(func(actual *smartcontract.StringMap) bool {
				if len(expected.Fields) != len(actual.Fields) {
					return false
				}
				for key, value := range expected.Fields {
					if value != actual.Fields[key] {
						return false
					}
				}
				return true
			}),
		).Return("", nil).Once()

		var conf = &scConfig{
			OwnerId: owner,
		}
		balances.On("GetTrieNode", scConfigKey(ssc.ID)).Return(conf, nil).Once()

		return args{
			ssc:      ssc,
			txn:      txn,
			input:    (&smartcontract.StringMap{Fields: p.inputMap}).Encode(),
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
				client:      owner,
				previousMap: map[string]string{},
				inputMap: map[string]string{
					"max_mint":                      "1500000.02",
					"time_unit":                     "720h",
					"min_alloc_size":                "1024",
					"min_alloc_duration":            "5m",
					"max_challenge_completion_time": "30m",
					"min_offer_duration":            "10h",
					"min_blobber_capacity":          "1024",

					"readpool.min_lock":        "10",
					"readpool.min_lock_period": "1h",
					"readpool.max_lock_period": "8760h",

					"writepool.min_lock":        "10",
					"writepool.min_lock_period": "2m",
					"writepool.max_lock_period": "8760h",

					"stakepool.min_lock":          "10",
					"stakepool.interest_rate":     "0.0",
					"stakepool.interest_interval": "1m",

					"max_total_free_allocation":      "10000",
					"max_individual_free_allocation": "100",

					"free_allocation_settings.data_shards":                   "10",
					"free_allocation_settings.parity_shards":                 "5",
					"free_allocation_settings.size":                          "10000000000",
					"free_allocation_settings.duration":                      "5000h",
					"free_allocation_settings.read_price_range.min":          "0.0",
					"free_allocation_settings.read_price_range.max":          "0.04",
					"free_allocation_settings.write_price_range.min":         "0.0",
					"free_allocation_settings.write_price_range.max":         "0.1",
					"free_allocation_settings.max_challenge_completion_time": "1m",
					"free_allocation_settings.read_pool_fraction":            "0.2",

					"validator_reward":                     "0.025",
					"blobber_slash":                        "0.1",
					"max_read_price":                       "100",
					"max_write_price":                      "100",
					"failed_challenges_to_cancel":          "20",
					"failed_challenges_to_revoke_min_lock": "0",
					"challenge_enabled":                    "true",
					"challenge_rate_per_mb_min":            "1.0",
					"max_challenges_per_generation":        "100",
					"max_delegates":                        "100",

					"block_reward.block_reward":           "1000",
					"block_reward.qualifying_stake":       "1",
					"block_reward.sharder_ratio":          "80.0",
					"block_reward.miner_ratio":            "20.0",
					"block_reward.blobber_capacity_ratio": "20.0",
					"block_reward.blobber_usage_ratio":    "80.0",

					"expose_mpt": "false",
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

func TestCommitSettingChanges(t *testing.T) {
	const mockMinerId = "mock miner id"
	type args struct {
		ssc      *StorageSmartContract
		txn      *transaction.Transaction
		input    []byte
		balances chainstate.StateContextI
	}

	type parameters struct {
		client       string
		inputMap     map[string]string
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
		var thisBlock = block.Block{}
		thisBlock.MinerID = mockMinerId

		balances.On("GetTrieNode", scConfigKey(ssc.ID)).Return(&scConfig{OwnerId: owner}, nil).Once()
		balances.On("GetTrieNode", settingChangesKey).Return(&smartcontract.StringMap{
			Fields: p.inputMap,
		}, nil).Once()

		balances.On(
			"InsertTrieNode",
			scConfigKey(ssc.ID),
			mock.MatchedBy(func(conf *scConfig) bool {
				for key, value := range p.inputMap {
					var setting interface{} = getConfField(*conf, key)
					fmt.Println("setting", setting, "value", value)
					switch Settings[key].configType {
					case smartcontract.Int:
						{
							expected, err := strconv.Atoi(value)
							require.NoError(t, err)
							actual, ok := setting.(int)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case smartcontract.Int64:
						{
							expected, err := strconv.ParseInt(value, 10, 64)
							require.NoError(t, err)
							actual, ok := setting.(int64)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case smartcontract.Float64:
						{
							expected, err := strconv.ParseFloat(value, 64)
							require.NoError(t, err)
							actual, ok := setting.(float64)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case smartcontract.Boolean:
						{
							expected, err := strconv.ParseBool(value)
							require.NoError(t, err)
							actual, ok := setting.(bool)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case smartcontract.Duration:
						{
							expected, err := time.ParseDuration(value)
							require.NoError(t, err)
							actual, ok := setting.(time.Duration)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case smartcontract.StateBalance:
						{
							expected, err := strconv.ParseFloat(value, 64)
							expected = x10 * expected
							require.NoError(t, err)
							actual, ok := setting.(state.Balance)
							require.True(t, ok)
							if state.Balance(expected) != actual {
								return false
							}
						}
					}
				}
				return true
			}),
		).Return("", nil).Once()

		return args{
			ssc:      ssc,
			txn:      txn,
			input:    (&smartcontract.StringMap{p.inputMap}).Encode(),
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
				client: mockMinerId,
				inputMap: map[string]string{
					"max_mint":                      "1500000.02",
					"time_unit":                     "720h",
					"min_alloc_size":                "1024",
					"min_alloc_duration":            "5m",
					"max_challenge_completion_time": "30m",
					"min_offer_duration":            "10h",
					"min_blobber_capacity":          "1024",

					"readpool.min_lock":        "10",
					"readpool.min_lock_period": "1h",
					"readpool.max_lock_period": "8760h",

					"writepool.min_lock":        "10",
					"writepool.min_lock_period": "2m",
					"writepool.max_lock_period": "8760h",

					"stakepool.min_lock":          "10",
					"stakepool.interest_rate":     "0.0",
					"stakepool.interest_interval": "1m",

					"max_total_free_allocation":      "10000",
					"max_individual_free_allocation": "100",

					"free_allocation_settings.data_shards":                   "10",
					"free_allocation_settings.parity_shards":                 "5",
					"free_allocation_settings.size":                          "10000000000",
					"free_allocation_settings.duration":                      "5000h",
					"free_allocation_settings.read_price_range.min":          "0.0",
					"free_allocation_settings.read_price_range.max":          "0.04",
					"free_allocation_settings.write_price_range.min":         "0.0",
					"free_allocation_settings.write_price_range.max":         "0.1",
					"free_allocation_settings.max_challenge_completion_time": "1m",
					"free_allocation_settings.read_pool_fraction":            "0.2",

					"validator_reward":                     "0.025",
					"blobber_slash":                        "0.1",
					"max_read_price":                       "100",
					"max_write_price":                      "100",
					"failed_challenges_to_cancel":          "20",
					"failed_challenges_to_revoke_min_lock": "0",
					"challenge_enabled":                    "true",
					"challenge_rate_per_mb_min":            "1.0",
					"max_challenges_per_generation":        "100",
					"max_delegates":                        "100",

					"block_reward.block_reward":           "1000",
					"block_reward.qualifying_stake":       "1",
					"block_reward.sharder_ratio":          "80.0",
					"block_reward.miner_ratio":            "20.0",
					"block_reward.blobber_capacity_ratio": "20.0",
					"block_reward.blobber_usage_ratio":    "80.0",

					"expose_mpt": "false",
				},
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.title, func(t *testing.T) {
			test := test
			t.Parallel()
			args := setExpectations(t, test.parameters)

			_, err := args.ssc.commitSettingChanges(args.txn, args.input, args.balances)
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
	case MinWritePrice:
		return conf.MinWritePrice
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
