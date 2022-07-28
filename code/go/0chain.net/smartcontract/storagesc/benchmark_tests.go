package storagesc

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"0chain.net/chaincore/currency"

	sc "0chain.net/smartcontract"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/chaincore/smartcontract"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/encryption"
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

func (bt BenchTest) Run(balances cstate.TimedQueryStateContext, b *testing.B) error {

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
	creationTimeRaw := viper.GetInt64(bk.MptCreationTime)
	creationTime := common.Now()
	if creationTimeRaw != 0 {
		creationTime = common.Timestamp(creationTimeRaw)
	}
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
			name:     "commit_connection",
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
			name:     "storage.new_allocation_request",
			endpoint: newAllocationRequestF,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID:     data.Clients[0],
				CreationDate: creationTime,
				Value: func() currency.Coin {
					v, err := currency.ParseZCN(100 * viper.GetFloat64(bk.StorageMaxWritePrice))
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
					Expiration:      common.Timestamp(viper.GetDuration(bk.StorageMinAllocDuration).Seconds()) + creationTime,
					Owner:           data.Clients[0],
					OwnerPublicKey:  data.PublicKeys[0],
					Blobbers:        blobbers,
					ReadPriceRange:  PriceRange{0, currency.Coin(viper.GetInt64(bk.StorageMaxReadPrice) * 1e10)},
					WritePriceRange: PriceRange{0, currency.Coin(viper.GetInt64(bk.StorageMaxWritePrice) * 1e10)},
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
				CreationDate: creationTime - 1,
				Value:        updateAllocVal,
			},
			input: func() []byte {
				uar := updateAllocationRequest{
					ID:              getMockAllocationId(0),
					OwnerID:         data.Clients[0],
					Size:            10000000,
					Expiration:      common.Timestamp(50 * 60 * 60),
					SetImmutable:    true,
					RemoveBlobberId: getMockBlobberId(0),
					AddBlobberId:    getMockBlobberId(viper.GetInt(bk.NumBlobbers) - 1),
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
				//CreationDate: common.Timestamp(viper.GetDuration(bk.StorageMinAllocDuration).Seconds()) + now,
				CreationDate: creationTime + benchAllocationExpire(creationTime) + 1,
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
				ClientID:     data.Clients[1],
				ToClientID:   ADDRESS,
				CreationDate: creationTime,
				Value:        maxIndividualFreeAlloc,
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
					Blobbers:           freeBlobbers,
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
				CreationDate: creationTime,
				Value:        maxIndividualFreeAlloc,
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
				CreationDate: creationTime + 1,
				ClientID:     "d46458063f43eb4aeb4adf1946d123908ef63143858abb24376d42b5761bf577",
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
				CreationDate: creationTime + 1,
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
				CreationDate: creationTime + 1,
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
				CreationDate: creationTime + 1,
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
		{
			name:     "storage.update_validator_settings",
			endpoint: ssc.updateValidatorSettings,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: creationTime + 1,
				ClientID:     getMockValidatorId(0),
				ToClientID:   ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&ValidationNode{
					ID:                getMockValidatorId(0),
					BaseURL:           getMockValidatorUrl(0),
					StakePoolSettings: getMockStakePoolSettings(getMockValidatorId(0)),
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
				ClientID:     data.Clients[0],
				CreationDate: creationTime,
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
				ClientID:     data.Clients[0],
				CreationDate: creationTime,
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
				ClientID:     data.Clients[0],
				CreationDate: creationTime,
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
			txn:      &transaction.Transaction{CreationDate: creationTime},
			input:    []byte{},
		},
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
		{
			name:     "storage.write_pool_unlock",
			endpoint: ssc.writePoolUnlock,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				Value: rpMinLock,
				ClientID: data.Clients[getMockOwnerFromAllocationIndex(
					viper.GetInt(bk.NumAllocations)-1, viper.GetInt(bk.NumActiveClients))],
				ToClientID: ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&unlockRequest{
					AllocationID: getMockAllocationId(viper.GetInt(bk.NumAllocations) - 1),
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
				ClientID:     data.Clients[0],
				ToClientID:   ADDRESS,
				CreationDate: creationTime,
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
			name:     "storage.collect_reward",
			endpoint: ssc.collectReward,
			txn: &transaction.Transaction{
				ClientID:     data.Clients[0],
				ToClientID:   ADDRESS,
				CreationDate: creationTime,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&stakepool.CollectRewardRequest{
					PoolId:       getMockBlobberStakePoolId(0, 0),
					ProviderType: spenum.Blobber,
				})
				return bytes
			}(),
		},
		{
			name: "storage.blobber_block_rewards",
			endpoint: func(
				_ *transaction.Transaction,
				_ []byte,
				balances cstate.StateContextI,
			) (string, error) {
				err := ssc.blobberBlockRewards(balances)
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
					//startBlobbers := getMockBlobberBlockFromAllocationIndex(i)

					vt := &ValidationTicket{
						ChallengeID:  getMockChallengeId(encryption.Hash("0"), getMockAllocationId(0)),
						BlobberID:    getMockBlobberId(0),
						ValidatorID:  getMockValidatorId(i),
						ValidatorKey: data.PublicKeys[0],
						Result:       true,
						Message:      "mock message",
						MessageCode:  "mock message code",
						Timestamp:    creationTime,
						Signature:    "",
					}
					hash := encryption.Hash(fmt.Sprintf("%v:%v:%v:%v:%v:%v", vt.ChallengeID, vt.BlobberID,
						vt.ValidatorID, vt.ValidatorKey, vt.Result, vt.Timestamp))
					_ = sigScheme.SetPublicKey(data.PublicKeys[0])
					sigScheme.SetPrivateKey(data.PrivateKeys[0])
					vt.Signature, _ = sigScheme.Sign(hash)
					validationTickets = append(validationTickets, vt)
				}
				bytes, _ := json.Marshal(&ChallengeResponse{
					ID:                getMockChallengeId(encryption.Hash("0"), getMockAllocationId(0)),
					ValidationTickets: validationTickets,
				})
				return bytes
			}(),
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
					"max_mint":                      "1500000.02",
					"time_unit":                     "720h",
					"min_alloc_size":                "1024",
					"min_alloc_duration":            "5m",
					"max_challenge_completion_time": "3m",
					"min_offer_duration":            "10h",
					"min_blobber_capacity":          "1024",

					"readpool.min_lock": "10",

					"writepool.min_lock":        "10",
					"writepool.min_lock_period": "2m",
					"writepool.max_lock_period": "8760h",

					"stakepool.min_lock": "10",

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
					"validators_per_challenge":             "2",
					"max_delegates":                        "100",

					"block_reward.block_reward":     "1000",
					"block_reward.qualifying_stake": "1",
					"block_reward.sharder_ratio":    "80.0",
					"block_reward.miner_ratio":      "20.0",
					"block_reward.gamma.alpha":      "0.2",
					"block_reward.gamma.a":          "10",
					"block_reward.gamma.b":          "9",
					"block_reward.zeta.i":           "1",
					"block_reward.zeta.k":           "0.9",
					"block_reward.zeta.mu":          "0.2",

					"expose_mpt": "false",
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
				challengesEnabled := viper.GetBool(bk.StorageChallengeEnabled)
				if challengesEnabled {
					err := ssc.generateChallenge(txn, balances.GetBlock(), nil, balances)
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
