package storagesc

import (
	"encoding/hex"
	"strconv"
	"testing"
	"time"

	"0chain.net/core/config"
	"github.com/0chain/common/core/currency"

	"0chain.net/chaincore/block"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/chain/state/mocks"
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
		TargetConfig          Config
	}

	setExpectations := func(t *testing.T, p parameters) args {
		var balances = &mocks.StateContextI{}
		var ssc = &StorageSmartContract{

			SmartContract: sci.NewSC(ADDRESS),
		}
		var txn = &transaction.Transaction{
			ClientID: p.client,
		}

		var oldChanges config.StringMap
		oldChanges.Fields = p.previousMap
		balances.On("GetTrieNode", settingChangesKey,
			mock.MatchedBy(func(c *config.StringMap) bool {
				*c = oldChanges
				return true
			})).Return(nil).Once()

		for key, value := range p.inputMap {
			oldChanges.Fields[key] = value
		}

		var expected = config.NewStringMap()
		for key, value := range p.previousMap {
			expected.Fields[key] = value
		}
		for key, value := range p.inputMap {
			expected.Fields[key] = value
		}

		balances.On(
			"InsertTrieNode",
			settingChangesKey,
			mock.MatchedBy(func(actual *config.StringMap) bool {
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

		var conf = &Config{
			OwnerId: owner,
		}
		balances.On("GetTrieNode", scConfigKey(ADDRESS),
			mock.MatchedBy(func(c *Config) bool {
				*c = *conf
				return true
			})).Return(nil).Once()

		return args{
			ssc:      ssc,
			txn:      txn,
			input:    (&config.StringMap{Fields: p.inputMap}).Encode(),
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
					"time_unit":                       "720h",
					"min_alloc_size":                  "1024",
					"max_challenge_completion_rounds": "720",
					"min_blobber_capacity":            "1024",

					"readpool.min_lock":  "10",
					"writepool.min_lock": "10",

					"max_total_free_allocation":      "10000",
					"max_individual_free_allocation": "100",
					"cancellation_charge":            "0.2",

					"free_allocation_settings.data_shards":           "10",
					"free_allocation_settings.parity_shards":         "5",
					"free_allocation_settings.size":                  "10000000000",
					"free_allocation_settings.read_price_range.min":  "0.0",
					"free_allocation_settings.read_price_range.max":  "0.04",
					"free_allocation_settings.write_price_range.min": "0.0",
					"free_allocation_settings.write_price_range.max": "0.1",
					"free_allocation_settings.read_pool_fraction":    "0.2",

					"validator_reward":                 "0.025",
					"blobber_slash":                    "0.1",
					"max_read_price":                   "100",
					"max_write_price":                  "100",
					"max_file_size":                    "40000000000000",
					"challenge_enabled":                "true",
					"challenge_generation_gap":         "1",
					"validators_per_challenge":         "2",
					"num_validators_rewarded":          "10",
					"max_blobber_select_for_challenge": "5",
					"max_delegates":                    "100",
					"owner_id":                         "f769ccdf8587b8cab6a0f6a8a5a0a91d3405392768f283c80a45d6023a1bfa1f",
					"block_reward.block_reward":        "1000",
					"block_reward.qualifying_stake":    "1",
					"block_reward.gamma.alpha":         "0.2",
					"block_reward.gamma.a":             "10",
					"block_reward.gamma.b":             "9",
					"block_reward.zeta.i":              "1",
					"block_reward.zeta.k":              "0.9",
					"block_reward.zeta.mu":             "0.2",
					"cost.update_settings":             "105",
					"cost.read_redeem":                 "105",
					"cost.commit_connection":           "105",
					"cost.new_allocation_request":      "105",
					"cost.update_allocation_request":   "105",
					"cost.finalize_allocation":         "105",
					"cost.cancel_allocation":           "105",
					"cost.add_free_storage_assigner":   "105",
					"cost.free_allocation_request":     "105",
					"cost.blobber_health_check":        "105",
					"cost.update_blobber_settings":     "105",
					"cost.pay_blobber_block_rewards":   "105",
					"cost.challenge_response":          "105",
					"cost.generate_challenge":          "105",
					"cost.add_validator":               "105",
					"cost.update_validator_settings":   "105",
					"cost.add_blobber":                 "105",
					"cost.read_pool_lock":              "105",
					"cost.read_pool_unlock":            "105",
					"cost.write_pool_lock":             "105",
					"cost.stake_pool_lock":             "105",
					"cost.stake_pool_unlock":           "105",
					"cost.commit_settings_changes":     "105",
					"cost.collect_reward":              "105",
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
		TargetConfig Config
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
		conf := newConfig()
		conf.OwnerId = owner
		balances.On("GetTrieNode", scConfigKey(ADDRESS),
			mockSetValue(conf)).Return(nil).Once()
		balances.On("GetTrieNode", settingChangesKey,
			mockSetValue(&config.StringMap{
				Fields: p.inputMap,
			})).Return(nil).Once()

		balances.On(
			"InsertTrieNode",
			scConfigKey(ADDRESS),
			mock.MatchedBy(func(conf *Config) bool {
				for key, value := range p.inputMap {
					setting := getConfField(*conf, key)
					switch Settings[key].configType {
					case config.Int:
						{
							expected, err := strconv.Atoi(value)
							require.NoError(t, err)
							actual, ok := setting.(int)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case config.Int64:
						{
							expected, err := strconv.ParseInt(value, 10, 64)
							require.NoError(t, err)
							actual, ok := setting.(int64)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case config.Float64:
						{
							expected, err := strconv.ParseFloat(value, 64)
							require.NoError(t, err)
							actual, ok := setting.(float64)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case config.Boolean:
						{
							expected, err := strconv.ParseBool(value)
							require.NoError(t, err)
							actual, ok := setting.(bool)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case config.Duration:
						{
							expected, err := time.ParseDuration(value)
							require.NoError(t, err)
							actual, ok := setting.(time.Duration)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case config.CurrencyCoin:
						{
							expected, err := strconv.ParseFloat(value, 64)
							expected = x10 * expected
							require.NoError(t, err)
							actual, ok := setting.(currency.Coin)
							require.True(t, ok)
							if currency.Coin(expected) != actual {
								return false
							}
						}
					case config.Cost:
						{
							expected, err := strconv.Atoi(value)
							require.NoError(t, err)
							actual, ok := setting.(int)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case config.Key:
						{
							_, err := hex.DecodeString(value)
							require.NoError(t, err)
							actual, ok := setting.(string)
							require.True(t, ok)
							if value != actual {
								return false
							}
						}
					default:
						return false
					}
				}
				return true
			}),
		).Return("", nil).Once()

		return args{
			ssc:      ssc,
			txn:      txn,
			input:    (&config.StringMap{Fields: p.inputMap}).Encode(),
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
			title: "all_settings",
			parameters: parameters{
				client: mockMinerId,
				inputMap: map[string]string{
					"time_unit":                       "720h",
					"min_alloc_size":                  "1024",
					"max_challenge_completion_rounds": "720",
					"min_blobber_capacity":            "1024",

					"readpool.min_lock":  "10",
					"writepool.min_lock": "10",

					"max_total_free_allocation":      "10000",
					"max_individual_free_allocation": "100",
					"cancellation_charge":            "0.2",

					"free_allocation_settings.data_shards":           "10",
					"free_allocation_settings.parity_shards":         "5",
					"free_allocation_settings.size":                  "10000000000",
					"free_allocation_settings.read_price_range.min":  "0.0",
					"free_allocation_settings.read_price_range.max":  "0.04",
					"free_allocation_settings.write_price_range.min": "0.0",
					"free_allocation_settings.write_price_range.max": "0.1",
					"free_allocation_settings.read_pool_fraction":    "0.2",
					"max_blobbers_per_allocation":                    "40",
					"health_check_period":                            "40h",
					"validator_reward":                               "0.025",
					"blobber_slash":                                  "0.1",
					"max_read_price":                                 "100",
					"max_write_price":                                "100",
					"max_file_size":                                  "40000000000000",
					"challenge_enabled":                              "true",
					"challenge_generation_gap":                       "1",
					"validators_per_challenge":                       "2",
					"num_validators_rewarded":                        "10",
					"max_blobber_select_for_challenge":               "5",
					"max_delegates":                                  "100",
					"owner_id":                                       "f769ccdf8587b8cab6a0f6a8a5a0a91d3405392768f283c80a45d6023a1bfa1f",
					"block_reward.block_reward":                      "1000",
					"block_reward.qualifying_stake":                  "1",
					"block_reward.gamma.alpha":                       "0.2",
					"block_reward.gamma.a":                           "10",
					"block_reward.gamma.b":                           "9",
					"block_reward.zeta.i":                            "1",
					"block_reward.zeta.k":                            "0.9",
					"block_reward.zeta.mu":                           "0.2",
					"cost.update_settings":                           "105",
					"cost.read_redeem":                               "105",
					"cost.commit_connection":                         "105",
					"cost.new_allocation_request":                    "105",
					"cost.update_allocation_request":                 "105",
					"cost.finalize_allocation":                       "105",
					"cost.cancel_allocation":                         "105",
					"cost.add_free_storage_assigner":                 "105",
					"cost.free_allocation_request":                   "105",
					"cost.blobber_health_check":                      "105",
					"cost.update_blobber_settings":                   "105",
					"cost.pay_blobber_block_rewards":                 "105",
					"cost.challenge_response":                        "105",
					"cost.generate_challenge":                        "105",
					"cost.add_validator":                             "105",
					"cost.update_validator_settings":                 "105",
					"cost.add_blobber":                               "105",
					"cost.read_pool_lock":                            "105",
					"cost.read_pool_unlock":                          "105",
					"cost.write_pool_lock":                           "105",
					"cost.stake_pool_lock":                           "105",
					"cost.stake_pool_unlock":                         "105",
					"cost.commit_settings_changes":                   "105",
					"cost.collect_reward":                            "105",
				},
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.title, func(t *testing.T) {
			test := test
			args := setExpectations(t, test.parameters)
			_, err := args.ssc.commitSettingChanges(args.txn, args.input, args.balances)
			if err != nil {
				t.Fatal("commitSettingChanges err: ", err.Error())
				return
			}
		})
	}
}

func getConfField(conf Config, field string) interface{} {
	if isCost(field) {
		value, _ := conf.getCost(field)
		return value
	}

	switch Settings[field].setting {
	case MaxStake:
		return conf.MaxStake
	case MinStake:
		return conf.MinStake
	case MinStakePerDelegate:
		return conf.MinStakePerDelegate
	case TimeUnit:
		return conf.TimeUnit
	case MinAllocSize:
		return conf.MinAllocSize
	case MaxChallengeCompletionRounds:
		return conf.MaxChallengeCompletionRounds
	case MinBlobberCapacity:
		return conf.MinBlobberCapacity

	case ReadPoolMinLock:
		return conf.ReadPool.MinLock
	case WritePoolMinLock:
		return conf.WritePool.MinLock

	case HealthCheckPeriod:
		return conf.HealthCheckPeriod
	case MaxTotalFreeAllocation:
		return conf.MaxTotalFreeAllocation
	case MaxIndividualFreeAllocation:
		return conf.MaxIndividualFreeAllocation
	case CancellationCharge:
		return conf.CancellationCharge
	case FreeAllocationDataShards:
		return conf.FreeAllocationSettings.DataShards
	case FreeAllocationParityShards:
		return conf.FreeAllocationSettings.ParityShards
	case FreeAllocationSize:
		return conf.FreeAllocationSettings.Size
	case FreeAllocationReadPriceRangeMin:
		return conf.FreeAllocationSettings.ReadPriceRange.Min
	case FreeAllocationReadPriceRangeMax:
		return conf.FreeAllocationSettings.ReadPriceRange.Max
	case FreeAllocationWritePriceRangeMin:
		return conf.FreeAllocationSettings.WritePriceRange.Min
	case FreeAllocationWritePriceRangeMax:
		return conf.FreeAllocationSettings.WritePriceRange.Max
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
	case MaxFileSize:
		return conf.MaxFileSize
	case ChallengeEnabled:
		return conf.ChallengeEnabled
	case ChallengeGenerationGap:
		return conf.ChallengeGenerationGap
	case ValidatorsPerChallenge:
		return conf.ValidatorsPerChallenge
	case NumValidatorsRewarded:
		return conf.NumValidatorsRewarded
	case MaxBlobberSelectForChallenge:
		return conf.MaxBlobberSelectForChallenge
	case MaxDelegates:
		return conf.MaxDelegates
	case BlockRewardBlockReward:
		return conf.BlockReward.BlockReward
	case BlockRewardQualifyingStake:
		return conf.BlockReward.QualifyingStake
	case MaxBlobbersPerAllocation:
		return conf.MaxBlobbersPerAllocation
	case BlockRewardGammaAlpha:
		return conf.BlockReward.Gamma.Alpha
	case BlockRewardGammaA:
		return conf.BlockReward.Gamma.A
	case BlockRewardGammaB:
		return conf.BlockReward.Gamma.B
	case BlockRewardZetaI:
		return conf.BlockReward.Zeta.I
	case BlockRewardZetaK:
		return conf.BlockReward.Zeta.K
	case BlockRewardZetaMu:
		return conf.BlockReward.Zeta.Mu
	case OwnerId:
		return conf.OwnerId
	default:
		panic("unknown field: " + field)
	}
}
