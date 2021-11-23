package storagesc

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"0chain.net/chaincore/smartcontract"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/encryption"
	sc "0chain.net/smartcontract"
	bk "0chain.net/smartcontract/benchmark"

	"github.com/spf13/viper"

	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
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
	txn   *transaction.Transaction
	input []byte
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

func (bt BenchTest) Run(balances cstate.StateContextI, b *testing.B) error {
	_, err := bt.endpoint(bt.Transaction(), bt.input, balances)
	return err
}

func BenchmarkTests(
	data bk.BenchData, sigScheme bk.SignatureScheme,
) bk.TestSuite {
	var now = common.Now()
	var ssc = StorageSmartContract{

		SmartContract: sci.NewSC(ADDRESS),
	}
	ssc.setSC(ssc.SmartContract, &smartcontract.BCContext{})
	var tests = []BenchTest{
		// read/write markers
		{
			name:     "storage.read_redeem",
			endpoint: ssc.commitBlobberRead,
			txn: &transaction.Transaction{
				ClientID:   data.Clients[0],
				ToClientID: ADDRESS,
			},
			input: func() []byte {
				rm := ReadMarker{
					ClientID:        data.Clients[0],
					ClientPublicKey: data.PublicKeys[0],
					BlobberID:       getMockBlobberId(0),
					AllocationID:    getMockAllocationId(0),
					OwnerID:         data.Clients[0],
					Timestamp:       now,
					ReadCounter:     viper.GetInt64(bk.NumWriteRedeemAllocation) + 1,
					PayerID:         data.Clients[0],
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
			name:     "commit_connection",
			endpoint: ssc.commitBlobberConnection,
			txn: &transaction.Transaction{
				ClientID:   getMockBlobberId(0),
				ToClientID: ADDRESS,
			},
			input: func() []byte {
				wm := WriteMarker{
					AllocationRoot:         encryption.Hash("allocation root"),
					PreviousAllocationRoot: encryption.Hash("allocation root"),
					AllocationID:           getMockAllocationId(0),
					Size:                   1024,
					BlobberID:              getMockBlobberId(0),
					Timestamp:              1,
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
			name:     "storage.new_allocation_request_random",
			endpoint: ssc.newAllocationRequest,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID:     data.Clients[0],
				CreationDate: now,
				Value:        100 * viper.GetInt64(bk.StorageMinAllocSize),
			},
			input: func() []byte {
				bytes, _ := (&newAllocationRequest{
					DataShards:                 viper.GetInt(bk.NumBlobbersPerAllocation) / 2,
					ParityShards:               viper.GetInt(bk.NumBlobbersPerAllocation) / 2,
					Size:                       100 * viper.GetInt64(bk.StorageMinAllocSize),
					Expiration:                 common.Timestamp(viper.GetDuration(bk.StorageMinAllocDuration).Seconds()) + now,
					Owner:                      data.Clients[0],
					OwnerPublicKey:             data.PublicKeys[0],
					PreferredBlobbers:          []string{},
					ReadPriceRange:             PriceRange{0, state.Balance(viper.GetInt64(bk.StorageMaxReadPrice) * 1e10)},
					WritePriceRange:            PriceRange{0, state.Balance(viper.GetInt64(bk.StorageMaxWritePrice) * 1e10)},
					MaxChallengeCompletionTime: viper.GetDuration(bk.StorageMaxChallengeCompletionTime),
					DiversifyBlobbers:          false,
				}).encode()
				return bytes
			}(),
		},
		{
			name:     "storage.new_allocation_request_preferred",
			endpoint: ssc.newAllocationRequest,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID:     data.Clients[0],
				CreationDate: now,
				Value:        100 * viper.GetInt64(bk.StorageMinAllocSize),
			},
			input: func() []byte {
				var blobberUrls []string
				for i := 0; i < viper.GetInt(bk.AvailableKeys); i++ {
					blobberUrls = append(blobberUrls, getMockBlobberId(0)+".com")
				}
				bytes, _ := (&newAllocationRequest{
					DataShards:                 viper.GetInt(bk.NumBlobbersPerAllocation) / 2,
					ParityShards:               viper.GetInt(bk.NumBlobbersPerAllocation) / 2,
					Size:                       100 * viper.GetInt64(bk.StorageMinAllocSize),
					Expiration:                 common.Timestamp(viper.GetDuration(bk.StorageMinAllocDuration).Seconds()) + now,
					Owner:                      data.Clients[0],
					OwnerPublicKey:             data.PublicKeys[0],
					PreferredBlobbers:          blobberUrls[:8],
					ReadPriceRange:             PriceRange{0, state.Balance(viper.GetInt64(bk.StorageMaxReadPrice) * 1e10)},
					WritePriceRange:            PriceRange{0, state.Balance(viper.GetInt64(bk.StorageMaxWritePrice) * 1e10)},
					MaxChallengeCompletionTime: viper.GetDuration(bk.StorageMaxChallengeCompletionTime),
					DiversifyBlobbers:          false,
				}).encode()
				return bytes
			}(),
		},
		// diversified blobbers panics if blobbers are more than around 30-50
		/*
			{
				name:     "storage.new_allocation_request_diversify",
				endpoint: ssc.newAllocationRequest,
				txn: &transaction.Transaction{
					HashIDField: datastore.HashIDField{
						Hash: encryption.Hash("mock transaction hash"),
					},
					ClientID:     data.Clients[0],
					CreationDate: now,
					Value:        100 * viper.GetInt64(bk.StorageMinAllocSize),
				},
				input: func() []byte {
					bytes, _ := (&newAllocationRequest{
						DataShards:                 viper.GetInt(bk.NumBlobbersPerAllocation) / 2,
						ParityShards:               viper.GetInt(bk.NumBlobbersPerAllocation) / 2,
						Size:                       100 * viper.GetInt64(bk.StorageMinAllocSize),
						Expiration:                 common.Timestamp(viper.GetDuration(bk.StorageMinAllocDuration).Seconds()) + now,
						Owner:                      data.Clients[0],
						OwnerPublicKey:             data.PublicKeys[0],
						PreferredBlobbers:          []string{},
						ReadPriceRange:             PriceRange{0, state.Balance(viper.GetInt64(bk.StorageMaxReadPrice) * 1e10)},
						WritePriceRange:            PriceRange{0, state.Balance(viper.GetInt64(bk.StorageMaxWritePrice) * 1e10)},
						MaxChallengeCompletionTime: viper.GetDuration(bk.StorageMaxChallengeCompletionTime),
						DiversifyBlobbers:          true,
					}).encode()
					return bytes
				}(),
			},
		*/
		{
			name:     "storage.update_allocation_request",
			endpoint: ssc.updateAllocationRequest,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID: data.Clients[0],
				Value:    viper.GetInt64(bk.StorageMinAllocSize) * 1e10,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&updateAllocationRequest{
					ID:           getMockAllocationId(0),
					OwnerID:      data.Clients[0],
					Size:         10000000,
					Expiration:   common.Timestamp(viper.GetDuration(bk.StorageMinAllocDuration).Seconds()),
					SetImmutable: true,
				})
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
				CreationDate: common.Timestamp((time.Hour * 1000).Seconds()) + now,
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
		{
			name:     "storage.cancel_allocation",
			endpoint: ssc.cancelAllocationRequest,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: now + 1,
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
				ClientID: owner,
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
				ClientID:     data.Clients[1],
				ToClientID:   ADDRESS,
				CreationDate: common.Timestamp(viper.GetInt64(bk.Now)),
				Value:        int64(viper.GetFloat64(bk.StorageMaxIndividualFreeAllocation) * 1e10),
			},
			input: func() []byte {
				var request = struct {
					Recipient  string           `json:"recipient"`
					FreeTokens float64          `json:"free_tokens"`
					Timestamp  common.Timestamp `json:"timestamp"`
				}{
					data.Clients[0],
					viper.GetFloat64(bk.StorageMaxIndividualFreeAllocation),
					1,
				}
				responseBytes, err := json.Marshal(&request)
				if err != nil {
					panic(err)
				}
				err = sigScheme.SetPublicKey(data.PublicKeys[0])
				if err != nil {
					panic(err)
				}
				sigScheme.SetPrivateKey(data.PrivateKeys[0])
				signature, err := sigScheme.Sign(hex.EncodeToString(responseBytes))
				if err != nil {
					panic(err)
				}
				fsmBytes, _ := json.Marshal(&freeStorageMarker{
					Assigner:   data.Clients[0],
					Recipient:  request.Recipient,
					FreeTokens: request.FreeTokens,
					Timestamp:  request.Timestamp,
					Signature:  signature,
				})
				bytes, _ := json.Marshal(&freeStorageAllocationInput{
					RecipientPublicKey: data.PublicKeys[1],
					Marker:             string(fsmBytes),
				})
				return bytes
			}(),
		},
		{
			name:     "storage.free_update_allocation",
			endpoint: ssc.updateFreeStorageRequest,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID:     data.Clients[1],
				ToClientID:   ADDRESS,
				CreationDate: common.Timestamp(viper.GetInt64(bk.Now)),
				Value:        int64(viper.GetFloat64(bk.StorageMaxIndividualFreeAllocation) * 1e10),
			},
			input: func() []byte {
				var request = struct {
					Recipient  string           `json:"recipient"`
					FreeTokens float64          `json:"free_tokens"`
					Timestamp  common.Timestamp `json:"timestamp"`
				}{
					data.Clients[0],
					viper.GetFloat64(bk.StorageMaxIndividualFreeAllocation),
					1,
				}
				responseBytes, _ := json.Marshal(&request)
				_ = sigScheme.SetPublicKey(data.PublicKeys[0])
				sigScheme.SetPrivateKey(data.PrivateKeys[0])
				signature, _ := sigScheme.Sign(hex.EncodeToString(responseBytes))
				fsmBytes, _ := json.Marshal(&freeStorageMarker{
					Assigner:   data.Clients[0],
					Recipient:  request.Recipient,
					FreeTokens: request.FreeTokens,
					Timestamp:  request.Timestamp,
					Signature:  signature,
				})
				bytes, _ := json.Marshal(&freeStorageUpgradeInput{
					AllocationId: getMockAllocationId(0),
					Marker:       string(fsmBytes),
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
				CreationDate: now + 1,
				ClientID:     data.Clients[0],
				ToClientID:   ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&StorageNode{
					ID:                encryption.Hash("my_new_blobber"),
					BaseURL:           "my_new_blobber.com",
					Terms:             getMockBlobberTerms(),
					Capacity:          viper.GetInt64(bk.StorageMinBlobberCapacity) * 1000,
					StakePoolSettings: getMockStakePoolSettings(encryption.Hash("my_new_blobber")),
				})
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
				CreationDate: now + 1,
				ClientID:     data.Clients[0],
				ToClientID:   ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&ValidationNode{
					ID:                encryption.Hash("my_new_validator"),
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
				CreationDate: now + 1,
				ClientID:     getMockBlobberId(0),
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
				CreationDate: now + 1,
				ClientID:     getMockBlobberId(0),
				ToClientID:   ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&StorageNode{
					ID:                getMockBlobberId(0),
					Terms:             getMockBlobberTerms(),
					Capacity:          viper.GetInt64(bk.StorageMinBlobberCapacity) * 1000,
					StakePoolSettings: getMockStakePoolSettings(getMockBlobberId(0)),
				})
				return bytes
			}(),
		},
		// add_curator
		{
			name:     "storage.curator_transfer_allocation",
			endpoint: ssc.curatorTransferAllocation,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID: data.Clients[0],
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&transferAllocationInput{
					AllocationId:      getMockAllocationId(0),
					NewOwnerId:        data.Clients[1],
					NewOwnerPublicKey: data.PublicKeys[1],
				})
				return bytes
			}(),
		},
		{
			name:     "storage.add_curator",
			endpoint: ssc.addCurator,
			txn: &transaction.Transaction{
				ClientID: data.Clients[0],
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&curatorInput{
					CuratorId:    data.Clients[viper.GetInt(bk.NumCurators)],
					AllocationId: getMockAllocationId(0),
				})
				return bytes
			}(),
		},
		{
			name:     "storage.remove_curator",
			endpoint: ssc.removeCurator,
			txn: &transaction.Transaction{
				ClientID: data.Clients[0],
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&curatorInput{
					CuratorId:    data.Clients[0],
					AllocationId: getMockAllocationId(0),
				})
				return bytes
			}(),
		},
		// read_pool
		{
			name:     "storage.new_read_pool",
			endpoint: ssc.newReadPool,
			txn:      &transaction.Transaction{},
			input:    []byte{},
		},
		{
			name:     "storage.read_pool_lock",
			endpoint: ssc.readPoolLock,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				Value:      int64(viper.GetFloat64(bk.StorageReadPoolMinLock) * 1e10),
				ClientID:   data.Clients[0],
				ToClientID: ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&lockRequest{
					Duration:     viper.GetDuration(bk.StorageReadPoolMinLockPeriod),
					AllocationID: getMockAllocationId(0),
				})
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
				Value:        int64(viper.GetFloat64(bk.StorageReadPoolMinLock) * 1e10),
				ClientID:     data.Clients[0],
				ToClientID:   ADDRESS,
				CreationDate: now + common.Timestamp(viper.GetDuration(bk.StorageWritePoolMinLockPeriod))*10,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&unlockRequest{
					PoolID: getMockReadPoolId(0, 0, 0),
				})
				return bytes
			}(),
		},
		// write pool
		{
			name:     "storage.write_pool_lock",
			endpoint: ssc.writePoolLock,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				Value:      int64(viper.GetFloat64(bk.StorageWritePoolMinLock) * 1e10),
				ClientID:   data.Clients[0],
				ToClientID: ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&lockRequest{
					Duration:     viper.GetDuration(bk.StorageWritePoolMinLockPeriod),
					AllocationID: getMockAllocationId(0),
				})
				return bytes
			}(),
		},
		{
			name:     "storage.write_pool_unlock",
			endpoint: ssc.writePoolUnlock,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				Value:        int64(viper.GetFloat64(bk.StorageReadPoolMinLock) * 1e10),
				ClientID:     data.Clients[0],
				ToClientID:   ADDRESS,
				CreationDate: now + common.Timestamp(viper.GetDuration(bk.StorageWritePoolMinLockPeriod))*10,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&unlockRequest{
					PoolID: getMockWritePoolId(0, 0, 0),
				})
				return bytes
			}(),
		},

		// stake pool
		{
			name:     "storage.stake_pool_lock",
			endpoint: ssc.stakePoolLock,
			txn: &transaction.Transaction{
				ClientID: data.Clients[0],
				Value:    int64(viper.GetFloat64(bk.StorageStakePoolMinLock) * 1e10),
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&stakePoolRequest{
					BlobberID: getMockBlobberId(0),
					//PoolID:    getMockStakePoolId(0, 0, data.Clients),
					PoolID: getMockBlobberStakePoolId(0, 0),
				})
				return bytes
			}(),
		},
		{
			name:     "storage.stake_pool_unlock",
			endpoint: ssc.stakePoolUnlock,
			txn: &transaction.Transaction{
				ClientID:   data.Clients[0],
				ToClientID: ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&stakePoolRequest{
					BlobberID: getMockBlobberId(0),
					PoolID:    getMockBlobberStakePoolId(0, 0),
				})
				return bytes
			}(),
		},
		{
			name:     "storage.stake_pool_pay_interests",
			endpoint: ssc.stakePoolPayInterests,
			txn:      &transaction.Transaction{},
			input: func() []byte {
				bytes, _ := json.Marshal(&stakePoolRequest{
					BlobberID: getMockBlobberId(0),
					PoolID:    getMockBlobberStakePoolId(0, 0),
				})
				return bytes
			}(),
		},
		{
			name:     "storage.challenge_response",
			endpoint: ssc.verifyChallenge,
			txn: &transaction.Transaction{
				ClientID: getMockBlobberId(0),
			},
			input: func() []byte {
				var validationTickets []*ValidationTicket
				vt := &ValidationTicket{
					ChallengeID:  getMockChallengeId(0, 0),
					BlobberID:    getMockBlobberId(0),
					ValidatorID:  getMockValidatorId(0),
					ValidatorKey: data.PublicKeys[0],
					Result:       true,
					Message:      "mock message",
					MessageCode:  "mock message code",
					Timestamp:    now,
					Signature:    "",
				}
				validationTickets = append(validationTickets, vt)
				hash := encryption.Hash(fmt.Sprintf("%v:%v:%v:%v:%v:%v", vt.ChallengeID, vt.BlobberID,
					vt.ValidatorID, vt.ValidatorKey, vt.Result, vt.Timestamp))
				_ = sigScheme.SetPublicKey(data.PublicKeys[0])
				sigScheme.SetPrivateKey(data.PrivateKeys[0])
				vt.Signature, _ = sigScheme.Sign(hash)
				bytes, _ := json.Marshal(&ChallengeResponse{
					ID:                getMockChallengeId(0, 0),
					ValidationTickets: validationTickets,
				})
				return bytes
			}(),
		},

		{
			name:     "storage.update_settings",
			endpoint: ssc.updateSettings,
			txn: &transaction.Transaction{
				ClientID: owner,
			},
			input: (&sc.StringMap{
				Fields: map[string]string{
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
			}).Encode(),
		},
		{
			name: "storage.generate_challenges",
			endpoint: func(
				txn *transaction.Transaction,
				_ []byte,
				balances cstate.StateContextI,
			) (string, error) {
				challengesEnabled := viper.GetBool(bk.StorageChallengeEnabled)
				if challengesEnabled {
					err := ssc.generateChallenges(txn, balances.GetBlock(), nil, balances)
					if err != nil {
						return "", nil
					}
				} else {
					return "Challenges disabled in the config", nil
				}
				return "Challenges generated", nil
			},
			txn: &transaction.Transaction{
				CreationDate: common.Timestamp(viper.GetInt64(bk.Now)),
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
