package storagesc

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	sc "0chain.net/core/config"
	"0chain.net/smartcontract/provider"

	"github.com/0chain/common/core/currency"

	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/chaincore/smartcontract"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	bk "0chain.net/smartcontract/benchmark"

	"github.com/spf13/viper"

	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
)

const owner = "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

type BenchTest struct {
	name     string
	endpoint func(
		*transaction.Transaction,
		[]byte,
		cstate.StateContextI,
	) (string, error)
	txn     *transaction.Transaction
	input   []byte
	timings map[string]time.Duration
}

func (bt BenchTest) Timings() map[string]time.Duration {
	return bt.timings
}

func (bt BenchTest) Name() string {
	return bt.name
}

func (bt BenchTest) Transaction() *transaction.Transaction {
	return &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: bt.txn.Hash,
		},
		ClientID:     bt.txn.ClientID,
		ToClientID:   bt.txn.ToClientID,
		Value:        bt.txn.Value,
		CreationDate: bt.txn.CreationDate,
	}
}

func (bt BenchTest) Run(balances cstate.TimedQueryStateContext, _ *testing.B) error {
	_, err := bt.endpoint(bt.Transaction(), bt.input, balances)
	return err
}

func BenchmarkTests(
	data bk.BenchData, sigScheme bk.SignatureScheme,
) bk.TestSuite {
	updateAllocVal, err := currency.ParseZCN(viper.GetFloat64(bk.StorageMinAllocSize))
	if err != nil {
		panic(err)
	}
	maxIndividualFreeAlloc, err := currency.ParseZCN(viper.GetFloat64(bk.StorageMaxIndividualFreeAllocation))
	if err != nil {
		panic(err)
	}

	rpMinLock, err := currency.ParseZCN(viper.GetFloat64(bk.StorageReadPoolMinLock))
	if err != nil {
		panic(err)
	}

	wpMinLock, err := currency.ParseZCN(viper.GetFloat64(bk.StorageWritePoolMinLock))
	if err != nil {
		panic(err)
	}

	spMinLock, err := currency.ParseZCN(viper.GetFloat64(bk.StorageStakePoolMinLock))
	if err != nil {
		panic(err)
	}
	var blobbers []string
	for i := 0; i < viper.GetInt(bk.NumBlobbersPerAllocation); i++ {
		blobbers = append(blobbers, getMockBlobberId(i))
	}
	var freeBlobbers []string
	for i := 0; i < viper.GetInt(bk.StorageFasDataShards)+viper.GetInt(bk.StorageFasParityShards); i++ {
		freeBlobbers = append(freeBlobbers, getMockBlobberId(i))
	}

	var ssc = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	ssc.setSC(ssc.SmartContract, &smartcontract.BCContext{})

	creationTime := common.Now()
	timings := make(map[string]time.Duration)
	newAllocationRequestF := func(
		t *transaction.Transaction,
		r []byte,
		b cstate.StateContextI,
	) (string, error) {
		return ssc.newAllocationRequest(t, r, b, timings)
	}

	var tests = []BenchTest{
		// read/write markers
		{
			name:     "storage.read_redeem",
			endpoint: ssc.commitBlobberRead,
			txn: &transaction.Transaction{
				ClientID:     data.Clients[0],
				ToClientID:   ADDRESS,
				CreationDate: creationTime,
			},
			input: func() []byte {
				rm := ReadMarker{
					ClientID:        data.Clients[0],
					ClientPublicKey: data.PublicKeys[0],
					BlobberID:       getMockBlobberId(0),
					AllocationID:    getMockAllocationId(0),
					OwnerID:         data.Clients[0],
					Timestamp:       creationTime,
					ReadCounter:     viper.GetInt64(bk.NumWriteRedeemAllocation) + 1,
				}
				_ = sigScheme.SetPublicKey(data.PublicKeys[0])
				sigScheme.SetPrivateKey(data.PrivateKeys[0])
				rm.Signature, _ = sigScheme.Sign(encryption.Hash(rm.GetHashData()))
				return (&ReadConnection{
					ReadMarker: &rm,
				}).Encode()
			}(),
		},
		{
			name:     "storage.commit_connection",
			endpoint: ssc.commitBlobberConnection,
			txn: &transaction.Transaction{
				ClientID:     getMockBlobberId(0),
				ToClientID:   ADDRESS,
				CreationDate: creationTime,
			},
			input: func() []byte {
				wm := WriteMarker{
					AllocationRoot:         encryption.Hash("allocation root"),
					PreviousAllocationRoot: encryption.Hash("allocation root"),
					AllocationID:           getMockAllocationId(0),
					Size:                   256,
					BlobberID:              getMockBlobberId(0),
					Timestamp:              creationTime,
					ClientID:               data.Clients[0],
				}
				_ = sigScheme.SetPublicKey(data.PublicKeys[0])
				sigScheme.SetPrivateKey(data.PrivateKeys[0])
				wm.Signature, _ = sigScheme.Sign(encryption.Hash(wm.GetHashData()))
				bytes, _ := json.Marshal(&BlobberCloseConnection{
					AllocationRoot:     encryption.Hash("allocation root"),
					PrevAllocationRoot: encryption.Hash("allocation root"),
					WriteMarker:        &wm,
				})
				return bytes
			}(),
		},

		// data.Allocations
		{
			name:     "storage.new_allocation_request",
			endpoint: newAllocationRequestF,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID:     data.Clients[0],
				ToClientID:   ADDRESS,
				CreationDate: creationTime,
				Value: func() currency.Coin {
					v, err := currency.ParseZCN(10 * viper.GetFloat64(bk.StorageMaxWritePrice))
					if err != nil {
						panic(err)
					}
					return v
				}(),
			},
			input: func() []byte {
				bytes, _ := (&newAllocationRequest{
					DataShards:      len(blobbers) / 2,
					ParityShards:    len(blobbers) / 2,
					Size:            10 * viper.GetInt64(bk.StorageMinAllocSize),
					Owner:           data.Clients[0],
					OwnerPublicKey:  data.PublicKeys[0],
					Blobbers:        blobbers,
					ReadPriceRange:  PriceRange{0, currency.Coin(viper.GetFloat64(bk.StorageMaxReadPrice) * 1e10)},
					WritePriceRange: PriceRange{0, currency.Coin(viper.GetFloat64(bk.StorageMaxWritePrice) * 1e10)},
				}).encode()
				return bytes
			}(),
			timings: timings,
		},
		{
			name:     "storage.update_allocation_request",
			endpoint: ssc.updateAllocationRequest,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID:     data.Clients[0],
				ToClientID:   ADDRESS,
				CreationDate: creationTime - 1,
				Value:        updateAllocVal,
			},
			input: func() []byte {
				uar := updateAllocationRequest{
					ID:              getMockAllocationId(0),
					OwnerID:         data.Clients[0],
					Size:            10000000,
					RemoveBlobberId: getMockBlobberId(0),
					AddBlobberId:    getMockBlobberId(viper.GetInt(bk.NumBlobbers) - 1),
					FileOptions:     63,
				}
				bytes, _ := json.Marshal(&uar)
				return bytes
			}(),
		},
		{
			name:     "storage.finalize_allocation",
			endpoint: ssc.finalizeAllocation,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: creationTime + benchAllocationExpire(creationTime) + 1,
				ClientID:     data.Clients[getMockOwnerFromAllocationIndex(0, viper.GetInt(bk.NumActiveClients))],
				ToClientID:   ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&lockRequest{
					AllocationID: getMockAllocationId(0),
				})
				return bytes
			}(),
		},
		{
			name:     "storage.cancel_allocation",
			endpoint: ssc.cancelAllocationRequest,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: creationTime - 1,
				ClientID:     data.Clients[0],
				ToClientID:   ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&lockRequest{
					AllocationID: getMockAllocationId(0),
				})
				return bytes
			}(),
		},
		// free data.Allocations
		{
			name:     "storage.add_free_storage_assigner",
			endpoint: ssc.addFreeStorageAssigner,
			txn: &transaction.Transaction{
				ClientID:     viper.GetString(bk.StorageOwner),
				CreationDate: creationTime,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&newFreeStorageAssignerInfo{
					Name:            "mock name",
					PublicKey:       encryption.Hash("mock public key"),
					IndividualLimit: viper.GetFloat64(bk.StorageMaxIndividualFreeAllocation),
					TotalLimit:      viper.GetFloat64(bk.StorageMaxTotalFreeAllocation),
				})
				return bytes
			}(),
		},
		{
			name:     "storage.free_allocation_request",
			endpoint: ssc.freeAllocationRequest,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID:     data.Clients[getMockOwnerFromAllocationIndex(0, viper.GetInt(bk.NumActiveClients))],
				ToClientID:   ADDRESS,
				CreationDate: creationTime,
				Value:        maxIndividualFreeAlloc,
			},
			input: func() []byte {
				var request = struct {
					Recipient  string  `json:"recipient"`
					FreeTokens float64 `json:"free_tokens"`
					Nonce      int64   `json:"nonce"`
				}{
					data.Clients[getMockOwnerFromAllocationIndex(0, viper.GetInt(bk.NumActiveClients))],
					viper.GetFloat64(bk.StorageMaxIndividualFreeAllocation),
					1,
				}
				err = sigScheme.SetPublicKey(data.PublicKeys[0])
				if err != nil {
					panic(err)
				}
				sigScheme.SetPrivateKey(data.PrivateKeys[0])
				marker := fmt.Sprintf("%s:%f:%d:%s",
					request.Recipient,
					request.FreeTokens,
					request.Nonce, freeBlobbers)
				signature, err := sigScheme.Sign(hex.EncodeToString([]byte(marker)))
				if err != nil {
					panic(err)
				}
				fsmBytes, _ := json.Marshal(&freeStorageMarker{
					Assigner:   data.Clients[getMockOwnerFromAllocationIndex(0, viper.GetInt(bk.NumActiveClients))],
					Recipient:  request.Recipient,
					FreeTokens: request.FreeTokens,
					Nonce:      request.Nonce,
					Signature:  signature,
					Blobbers:   freeBlobbers,
				})
				bytes, _ := json.Marshal(&freeStorageAllocationInput{
					RecipientPublicKey: data.PublicKeys[1],
					Marker:             string(fsmBytes),
				})
				return bytes
			}(),
		},

		// data.Blobbers
		{
			name:     "storage.add_blobber",
			endpoint: ssc.addBlobber,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: creationTime + 1,
				//ClientID:     "d46458063f43eb4aeb4adf1946d123908ef63143858abb24376d42b5761bf577",
				ClientID:   encryption.Hash("my_new_blobber"),
				ToClientID: ADDRESS,
			},
			input: func() []byte {
				b := &StorageNode{}
				bv2 := &storageNodeV2{
					Provider: provider.Provider{
						ProviderType: spenum.Blobber,
					},
					BaseURL:           "my_new_blobber.com",
					Terms:             getMockBlobberTerms(),
					Capacity:          viper.GetInt64(bk.StorageMinBlobberCapacity) * 1000,
					StakePoolSettings: getMockStakePoolSettings(encryption.Hash("my_new_blobber")),
				}
				b.SetEntity(bv2)
				bytes, _ := json.Marshal(b)
				return bytes
			}(),
		},
		{
			name:     "storage.add_validator",
			endpoint: ssc.addValidator,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: creationTime + 1,
				ClientID:     encryption.Hash("my_new_validator"),
				ToClientID:   ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&ValidationNode{
					Provider: provider.Provider{
						ID:           encryption.Hash("my_new_validator"),
						ProviderType: spenum.Validator,
					},
					BaseURL:           "my_new_validator.com",
					StakePoolSettings: getMockStakePoolSettings(encryption.Hash("my_new_validator")),
				})

				return bytes
			}(),
		},
		{
			name:     "storage.blobber_health_check",
			endpoint: ssc.blobberHealthCheck,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: creationTime + 1,
				ClientID:     getMockBlobberId(0),
				ToClientID:   ADDRESS,
			},
			input: []byte{},
		},
		{
			name:     "storage.validator_health_check",
			endpoint: ssc.validatorHealthCheck,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: creationTime + 1,
				ClientID:     data.ValidatorIds[0],
				ToClientID:   ADDRESS,
			},
			input: []byte{},
		},
		{
			name:     "storage.update_blobber_settings",
			endpoint: ssc.updateBlobberSettings,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: creationTime + 1,
				ClientID:     getMockBlobberId(0),
				ToClientID:   ADDRESS,
			},
			input: func() []byte {
				stake := currency.Coin(viper.GetInt64(bk.StorageMaxStake) * 1e10)
				totalStake := stake * currency.Coin(viper.GetInt(bk.NumBlobberDelegates))
				b := &StorageNode{}
				b.SetEntity(&storageNodeV2{
					Provider: provider.Provider{
						ID:           getMockBlobberId(0),
						ProviderType: spenum.Blobber,
					},
					Terms:             getMockBlobberTerms(),
					Capacity:          int64(totalStake * GB),
					StakePoolSettings: getMockStakePoolSettings(getMockBlobberId(0)),
				})
				bytes, _ := json.Marshal(b)
				return bytes
			}(),
		},
		{
			name:     "storage.update_validator_settings",
			endpoint: ssc.updateValidatorSettings,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: creationTime + 1,
				ClientID:     data.ValidatorIds[0],
				ToClientID:   ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&ValidationNode{
					Provider: provider.Provider{
						ID:           data.ValidatorIds[0],
						ProviderType: spenum.Validator,
					},
					BaseURL:           getMockValidatorUrl(data.ValidatorIds[0]),
					StakePoolSettings: getMockStakePoolSettings(data.ValidatorIds[0]),
				})
				return bytes
			}(),
		},
		// read_pool
		{
			name:     "storage.read_pool_lock",
			endpoint: ssc.readPoolLock,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				Value:        rpMinLock,
				ClientID:     data.Clients[0],
				ToClientID:   ADDRESS,
				CreationDate: creationTime,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&readPoolLockRequest{})
				return bytes
			}(),
		},
		{
			name:     "storage.read_pool_unlock",
			endpoint: ssc.readPoolUnlock,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				Value:        rpMinLock,
				ClientID:     data.Clients[0],
				ToClientID:   ADDRESS,
				CreationDate: creationTime + 1,
			},
			input: []byte{},
		},
		// write pool
		{
			name:     "storage.write_pool_lock",
			endpoint: ssc.writePoolLock,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				Value:        wpMinLock,
				ClientID:     data.Clients[0],
				ToClientID:   ADDRESS,
				CreationDate: creationTime,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&lockRequest{
					AllocationID: getMockAllocationId(0),
				})
				return bytes
			}(),
		},

		// stake pool
		{
			name:     "storage.stake_pool_lock",
			endpoint: ssc.stakePoolLock,
			txn: &transaction.Transaction{
				ClientID:     data.Clients[0],
				Value:        spMinLock,
				CreationDate: creationTime,
				ToClientID:   ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&stakePoolRequest{
					ProviderType: spenum.Blobber,
					ProviderID:   getMockBlobberId(0),
				})
				return bytes
			}(),
		},
		{
			name:     "storage.stake_pool_unlock",
			endpoint: ssc.stakePoolUnlock,
			txn: &transaction.Transaction{
				ClientID:     getMockBlobberStakePoolId(0, 0, data.Clients),
				ToClientID:   ADDRESS,
				CreationDate: creationTime,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&stakePoolRequest{
					ProviderType: spenum.Blobber,
					ProviderID:   getMockBlobberId(0),
				})
				return bytes
			}(),
		},
		{
			name:     "storage.collect_reward",
			endpoint: ssc.collectReward,
			txn: &transaction.Transaction{
				ClientID:     data.Clients[0],
				ToClientID:   ADDRESS,
				CreationDate: creationTime,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&stakepool.CollectRewardRequest{
					ProviderId:   getMockBlobberId(0),
					ProviderType: spenum.Blobber,
				})
				return bytes
			}(),
		},
		{
			name: "storage.blobber_block_rewards",
			endpoint: func(
				txn *transaction.Transaction,
				_ []byte,
				balances cstate.StateContextI,
			) (string, error) {
				inp := BlobberBlockRewardsInput{Round: balances.GetBlock().Round}
				marshal, err2 := json.Marshal(inp)
				if err2 != nil {
					return "", err2
				}
				err := ssc.blobberBlockRewards(txn, marshal, balances)
				if err != nil {
					return "", err
				} else {
					return "blobber block rewarded", nil
				}
			},
			txn: &transaction.Transaction{CreationDate: creationTime},
		},
		{
			name:     "storage.challenge_response",
			endpoint: ssc.verifyChallenge,
			txn: &transaction.Transaction{
				ClientID:     getMockBlobberId(0),
				CreationDate: creationTime,
			},
			input: func() []byte {
				var validationTickets []*ValidationTicket
				//always use first NumBlobbersPerAllocation/2 validators the same we use for challenge creation.
				//to randomize it we need to load challenge here, not sure if it's needed
				for i := 0; i < viper.GetInt(bk.NumBlobbersPerAllocation)/2; i++ {
					vt := &ValidationTicket{
						ChallengeID:  getMockChallengeId(getMockBlobberId(0), getMockAllocationId(0)),
						BlobberID:    getMockBlobberId(0),
						ValidatorID:  data.ValidatorIds[i],
						ValidatorKey: data.ValidatorPublicKeys[i],
						Result:       true,
						Message:      "mock message",
						MessageCode:  "mock message code",
						Timestamp:    creationTime,
					}
					hash := encryption.Hash(fmt.Sprintf("%v:%v:%v:%v:%v:%v", vt.ChallengeID, vt.BlobberID,
						vt.ValidatorID, vt.ValidatorKey, vt.Result, vt.Timestamp))
					_ = sigScheme.SetPublicKey(data.ValidatorPublicKeys[i])
					sigScheme.SetPrivateKey(data.ValidatorPrivateKeys[i])
					vt.Signature, _ = sigScheme.Sign(hash)
					validationTickets = append(validationTickets, vt)
				}
				bytes, _ := json.Marshal(&ChallengeResponse{
					ID:                getMockChallengeId(getMockBlobberId(0), getMockAllocationId(0)),
					ValidationTickets: validationTickets,
				})
				return bytes
			}(),
		},
		{
			name:     "storage.commit_settings_changes",
			endpoint: ssc.commitSettingChanges,
			txn:      &transaction.Transaction{},
		},
		{
			name: "storage.kill_blobber",
			input: (&provider.ProviderRequest{
				ID: getMockBlobberId(0),
			}).Encode(),
			endpoint: ssc.killBlobber,
			txn: &transaction.Transaction{
				ClientID:     viper.GetString(bk.StorageOwner),
				CreationDate: creationTime,
			},
		},
		{
			name: "storage.kill_validator",
			input: (&provider.ProviderRequest{
				ID: data.ValidatorIds[0],
			}).Encode(),
			endpoint: ssc.killValidator,
			txn: &transaction.Transaction{
				ClientID:     viper.GetString(bk.StorageOwner),
				CreationDate: creationTime,
			},
		},
		{
			name:     "storage.shutdown_blobber",
			endpoint: ssc.shutdownBlobber,
			input: (&provider.ProviderRequest{
				ID: getMockBlobberId(0),
			}).Encode(),
			txn: &transaction.Transaction{
				ClientID: getMockBlobberId(0),
			},
		},
		{
			name:     "storage.shutdown_validator",
			endpoint: ssc.shutdownValidator,
			input: (&provider.ProviderRequest{
				ID: data.ValidatorIds[0],
			}).Encode(),
			txn: &transaction.Transaction{
				ClientID: data.ValidatorIds[0],
			},
		},
		{
			name:     "storage.update_settings",
			endpoint: ssc.updateSettings,
			txn: &transaction.Transaction{
				ClientID:     viper.GetString(bk.StorageOwner),
				CreationDate: creationTime,
			},
			input: (&sc.StringMap{
				Fields: map[string]string{
					"time_unit":                       "720h",
					"min_alloc_size":                  "1024",
					"max_challenge_completion_rounds": "720",
					"min_blobber_capacity":            "1024",

					"readpool.min_lock": "10",

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
					"min_stake_per_delegate":           "1",

					"block_reward.block_reward":     "1000",
					"block_reward.qualifying_stake": "1",
					"block_reward.gamma.alpha":      "0.2",
					"block_reward.gamma.a":          "10",
					"block_reward.gamma.b":          "9",
					"block_reward.zeta.i":           "1",
					"block_reward.zeta.k":           "0.9",
					"block_reward.zeta.mu":          "0.2",

					"cost.update_settings":           "105",
					"cost.read_redeem":               "105",
					"cost.commit_connection":         "105",
					"cost.new_allocation_request":    "105",
					"cost.update_allocation_request": "105",
					"cost.finalize_allocation":       "105",
					"cost.cancel_allocation":         "105",
					"cost.add_free_storage_assigner": "105",
					"cost.free_allocation_request":   "105",
					"cost.blobber_health_check":      "105",
					"cost.update_blobber_settings":   "105",
					"cost.pay_blobber_block_rewards": "105",
					"cost.challenge_response":        "105",
					"cost.generate_challenge":        "105",
					"cost.add_validator":             "105",
					"cost.update_validator_settings": "105",
					"cost.add_blobber":               "105",
					"cost.read_pool_lock":            "105",
					"cost.read_pool_unlock":          "105",
					"cost.write_pool_lock":           "105",
					"cost.stake_pool_lock":           "105",
					"cost.stake_pool_unlock":         "105",
					"cost.commit_settings_changes":   "105",
					"cost.collect_reward":            "105",
				},
			}).Encode(),
		},
		{
			name: "storage.generate_challenge",
			endpoint: func(
				txn *transaction.Transaction,
				_ []byte,
				balances cstate.StateContextI,
			) (string, error) {
				conf, err := getConfig(balances)
				if err != nil {
					return "", err
				}

				input := &GenerateChallengeInput{Round: balances.GetBlock().Round}
				marshal, err := json.Marshal(input)
				if err != nil {
					return "", err
				}

				if conf.ChallengeEnabled {
					err := ssc.generateChallenge(txn, balances.GetBlock(), marshal, conf, balances)
					if err != nil {
						return "", nil
					}
				} else {
					return "OpenChallenges disabled in the config", nil
				}
				return "OpenChallenges generated", nil
			},
			txn: &transaction.Transaction{
				CreationDate: creationTime,
			},
			input: nil,
		},
		// todo "update_config" waiting for PR489
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuite{
		Source:     bk.Storage,
		Benchmarks: testsI,
	}
}
