package storagesc

import (
	"encoding/hex"
	"encoding/json"
	"time"

	"0chain.net/chaincore/state"
	sc "0chain.net/smartcontract/benchmark"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/encryption"

	"github.com/spf13/viper"

	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

type BenchTest struct {
	name     string
	endpoint func(
		*transaction.Transaction,
		[]byte,
		cstate.StateContextI,
	) (string, error)
	txn   transaction.Transaction
	input []byte
}

func (bt BenchTest) Name() string {
	return bt.name
}

func (bt BenchTest) Transaction() transaction.Transaction {
	return bt.txn
}

func (bt BenchTest) Run(balances cstate.StateContextI) {
	_, err := bt.endpoint(&bt.txn, bt.input, balances)
	if err != nil {
		panic(err) // todo temporary, remove later
	}
}

func BenchmarkTests(
	data sc.BenchData, sigScheme sc.SignatureScheme,
) sc.TestSuit {
	var now = common.Timestamp(viper.GetInt64(sc.Now))
	var ssc = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	var tests = []BenchTest{
		// read/write markers
		{
			name:     "storage.read_redeem",
			endpoint: ssc.commitBlobberRead,
			txn: transaction.Transaction{
				ClientID:   data.Clients[0],
				ToClientID: ADDRESS,
			},
			input: func() []byte {
				rm := ReadMarker{
					ClientID:        data.Clients[0],
					ClientPublicKey: data.PublicKeys[0],
					BlobberID:       data.Blobbers[0],
					AllocationID:    data.Allocations[0],
					OwnerID:         data.Clients[0],
					Timestamp:       now,
					ReadCounter:     1,
					PayerID:         data.Clients[0],
				}
				sigScheme.SetPublicKey(data.PublicKeys[0])
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
			txn: transaction.Transaction{
				ClientID:   data.Blobbers[0],
				ToClientID: ADDRESS,
			},
			input: func() []byte {
				wm := WriteMarker{
					AllocationRoot:         encryption.Hash("allocation root"),
					PreviousAllocationRoot: encryption.Hash("allocation root"),
					AllocationID:           data.Allocations[0],
					Size:                   1024,
					BlobberID:              data.Blobbers[0],
					Timestamp:              1,
					ClientID:               data.Clients[0],
				}
				sigScheme.SetPublicKey(data.PublicKeys[0])
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
			txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID:     data.Clients[0],
				CreationDate: now,
				Value:        100 * viper.GetInt64(sc.StorageMinAllocSize),
			},
			input: func() []byte {
				bytes, _ := (&newAllocationRequest{
					DataShards:                 4,
					ParityShards:               4,
					Size:                       100 * viper.GetInt64(sc.StorageMinAllocSize),
					Expiration:                 common.Timestamp(viper.GetDuration(sc.StorageMinAllocDuration).Seconds()) + now,
					Owner:                      data.Clients[0],
					OwnerPublicKey:             data.PublicKeys[0],
					PreferredBlobbers:          []string{},
					ReadPriceRange:             PriceRange{0, state.Balance(viper.GetInt64(sc.StorageMaxReadPrice) * 1e10)},
					WritePriceRange:            PriceRange{0, state.Balance(viper.GetInt64(sc.StorageMaxWritePrice) * 1e10)},
					MaxChallengeCompletionTime: viper.GetDuration(sc.StorageMaxChallengeCompletionTime),
					DiversifyBlobbers:          false,
				}).encode()
				return bytes
			}(),
		},
		{
			name:     "storage.new_allocation_request_preferred",
			endpoint: ssc.newAllocationRequest,
			txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID:     data.Clients[0],
				CreationDate: now,
				Value:        100 * viper.GetInt64(sc.StorageMinAllocSize),
			},
			input: func() []byte {
				var blobberUrls []string
				for i := 0; i < 8; i++ {
					blobberUrls = append(blobberUrls, data.Blobbers[i]+".com")
				}
				bytes, _ := (&newAllocationRequest{
					DataShards:                 4,
					ParityShards:               4,
					Size:                       100 * viper.GetInt64(sc.StorageMinAllocSize),
					Expiration:                 common.Timestamp(viper.GetDuration(sc.StorageMinAllocDuration).Seconds()) + now,
					Owner:                      data.Clients[0],
					OwnerPublicKey:             data.PublicKeys[0],
					PreferredBlobbers:          blobberUrls[:8],
					ReadPriceRange:             PriceRange{0, state.Balance(viper.GetInt64(sc.StorageMaxReadPrice) * 1e10)},
					WritePriceRange:            PriceRange{0, state.Balance(viper.GetInt64(sc.StorageMaxWritePrice) * 1e10)},
					MaxChallengeCompletionTime: viper.GetDuration(sc.StorageMaxChallengeCompletionTime),
					DiversifyBlobbers:          false,
				}).encode()
				return bytes
			}(),
		},
		{
			name:     "storage.update_allocation_request",
			endpoint: ssc.updateAllocationRequest,
			txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID: data.Clients[0],
				Value:    100 * viper.GetInt64(sc.StorageMinAllocSize),
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&updateAllocationRequest{
					ID:           data.Allocations[0],
					OwnerID:      data.Clients[0],
					Size:         100 * viper.GetInt64(sc.StorageMinAllocSize),
					Expiration:   common.Timestamp(viper.GetDuration(sc.StorageMinAllocDuration).Seconds()),
					SetImmutable: true,
				})
				return bytes
			}(),
		},
		{
			name:     "storage.finalize_allocation",
			endpoint: ssc.finalizeAllocation,
			txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: common.Timestamp((time.Hour * 1000).Seconds()) + now,
				ClientID:     data.Clients[0],
				ToClientID:   ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&lockRequest{
					AllocationID: data.Allocations[0],
				})
				return bytes
			}(),
		},
		{
			name:     "storage.cancel_allocation",
			endpoint: ssc.cancelAllocationRequest,
			txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: now + 1,
				ClientID:     data.Clients[0],
				ToClientID:   ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&lockRequest{
					AllocationID: data.Allocations[0],
				})
				return bytes
			}(),
		},
		// free data.Allocations
		{
			name:     "storage.add_free_storage_assigner",
			endpoint: ssc.addFreeStorageAssigner,
			txn: transaction.Transaction{
				ClientID: owner,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&newFreeStorageAssignerInfo{
					Name:            "mock name",
					PublicKey:       encryption.Hash("mock public key"),
					IndividualLimit: viper.GetFloat64(sc.StorageMaxIndividualFreeAllocation),
					TotalLimit:      viper.GetFloat64(sc.StorageMaxTotalFreeAllocation),
				})
				return bytes
			}(),
		},
		/* todo needs read_pool_lock fixed
		{
			name:     "storage.free_allocation_request",
			endpoint: ssc.freeAllocationRequest,
			txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID:     data.Clients[1],
				ToClientID:   ADDRESS,
				CreationDate: common.Timestamp(viper.GetInt64(sc.Now)),
				Value:        100 * viper.GetInt64(sc.StorageMinAllocSize),
			},
			input: func() []byte {
				var request = struct {
					Recipient  string           `json:"recipient"`
					FreeTokens float64          `json:"free_tokens"`
					Timestamp  common.Timestamp `json:"timestamp"`
				}{
					data.Clients[0],
					viper.GetFloat64(sc.StorageMaxIndividualFreeAllocation),
					1,
				}
				responseBytes, err := json.Marshal(&request)
				if err != nil {
					panic(err)
				}
				sigScheme.SetPublicKey(data.PublicKeys[0])
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
		*/
		{
			name:     "storage.free_update_allocation",
			endpoint: ssc.updateFreeStorageRequest,
			txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID:     data.Clients[1],
				ToClientID:   ADDRESS,
				CreationDate: common.Timestamp(viper.GetInt64(sc.Now)),
				Value:        100 * viper.GetInt64(sc.StorageMinAllocSize),
			},
			input: func() []byte {
				var request = struct {
					Recipient  string           `json:"recipient"`
					FreeTokens float64          `json:"free_tokens"`
					Timestamp  common.Timestamp `json:"timestamp"`
				}{
					data.Clients[1],
					viper.GetFloat64(sc.StorageMaxIndividualFreeAllocation),
					1,
				}
				responseBytes, _ := json.Marshal(&request)
				sigScheme.SetPublicKey(data.PublicKeys[0])
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
					AllocationId: data.Allocations[1],
					Marker:       string(fsmBytes),
				})
				return bytes
			}(),
		},

		// data.Blobbers
		{
			name:     "storage.add_blobber",
			endpoint: ssc.addBlobber,
			txn: transaction.Transaction{
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
					Capacity:          viper.GetInt64(sc.StorageMinBlobberCapacity) * 1000,
					StakePoolSettings: getStakePoolSettings(encryption.Hash("my_new_blobber")),
				})
				return bytes
			}(),
		},
		{
			name:     "storage.add_validator",
			endpoint: ssc.addValidator,
			txn: transaction.Transaction{
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
					StakePoolSettings: getStakePoolSettings(encryption.Hash("my_new_validator")),
				})
				return bytes
			}(),
		},
		{
			name:     "storage.blobber_health_check",
			endpoint: ssc.blobberHealthCheck,
			txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: now + 1,
				ClientID:     data.Blobbers[0],
				ToClientID:   ADDRESS,
			},
			input: []byte{},
		},
		{
			name:     "storage.update_blobber_settings",
			endpoint: ssc.updateBlobberSettings,
			txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: now + 1,
				ClientID:     data.Blobbers[0],
				ToClientID:   ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&StorageNode{
					ID:                data.Blobbers[0],
					Terms:             getMockBlobberTerms(),
					Capacity:          viper.GetInt64(sc.StorageMinBlobberCapacity) * 1000,
					StakePoolSettings: getStakePoolSettings(data.Blobbers[0]),
				})
				return bytes
			}(),
		},
		// add_curator
		{
			name:     "storage.curator_transfer_allocation",
			endpoint: ssc.curatorTransferAllocation,
			txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID: data.Clients[0],
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&transferAllocationInput{
					AllocationId:      data.Allocations[0],
					NewOwnerId:        data.Clients[1],
					NewOwnerPublicKey: data.PublicKeys[1],
				})
				return bytes
			}(),
		},
		{
			name:     "storage.add_curator",
			endpoint: ssc.addCurator,
			txn: transaction.Transaction{
				ClientID: data.Clients[0],
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&curatorInput{
					CuratorId:    data.Clients[viper.GetInt(sc.NumCurators)],
					AllocationId: data.Allocations[0],
				})
				return bytes
			}(),
		},
		{
			name:     "storage.remove_curator",
			endpoint: ssc.removeCurator,
			txn: transaction.Transaction{
				ClientID: data.Clients[0],
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&curatorInput{
					CuratorId:    data.Clients[0],
					AllocationId: data.Allocations[0],
				})
				return bytes
			}(),
		},
		// read_pool
		{
			name:     "storage.new_read_pool",
			endpoint: ssc.newReadPool,
			txn:      transaction.Transaction{},
			input:    []byte{},
		},
		/* todo read_unlock_lock, seems to be bugged, needs to be fixed before can benchmark
		{
			name:     "storage.read_pool_unlock",
			endpoint: ssc.readPoolUnlock,
			txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				Value:      int64(viper.GetFloat64(sc.StorageReadPoolMinLock) * 1e10),
				ClientID:   data.Clients[0],
				ToClientID: ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&unlockRequest{
					PoolID: data.Allocations[0],
				})
				return bytes
			}(),
		},
		*/
		// todo read_pool_lock, seems to be bugged, needs to be fixed before we can benchmark
		// write pool
		/*
			{
				name:     "storage.write_pool_unlock",
				endpoint: ssc.readPoolUnlock,
				txn: transaction.Transaction{
					HashIDField: datastore.HashIDField{
						Hash: encryption.Hash("mock transaction hash"),
					},
					Value:      int64(viper.GetFloat64(sc.StorageWritePoolMinLock) * 1e10),
					ClientID:   data.Clients[0],
					ToClientID: ADDRESS,
				},
				input: func() []byte {
					bytes, _ := json.Marshal(&unlockRequest{
						PoolID: data.Allocations[0],
					})
					return bytes
				}(),
			},
		*/
		// todo write_pool_unlock, seems to be bugged, needs to be fixed before we can benchmark
		// todo write_pool_lock, seems to be bugged, needs to be fixed before we can benchmark

		// stake pool
		{
			name:     "storage.stake_pool_lock",
			endpoint: ssc.stakePoolLock,
			txn: transaction.Transaction{
				ClientID: data.Clients[0],
				Value:    int64(viper.GetFloat64(sc.StorageStakePoolMinLock) * 1e10),
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&stakePoolRequest{
					BlobberID: data.Blobbers[0],
					//PoolID:    getMockStakePoolId(0, 0, data.Clients),
					PoolID: getMockStakePoolId(0, 0),
				})
				return bytes
			}(),
		},
		{
			name:     "storage.stake_pool_unlock",
			endpoint: ssc.stakePoolUnlock,
			txn: transaction.Transaction{
				ClientID:   data.Clients[0],
				ToClientID: ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&stakePoolRequest{
					BlobberID: data.Blobbers[0],
					PoolID:    getMockStakePoolId(0, 0),
				})
				return bytes
			}(),
		},
		{
			name:     "storage.stake_pool_pay_interests",
			endpoint: ssc.stakePoolPayInterests,
			txn:      transaction.Transaction{},
			input: func() []byte {
				bytes, _ := json.Marshal(&stakePoolRequest{
					BlobberID: data.Blobbers[0],
					PoolID:    getMockStakePoolId(0, 0),
				})
				return bytes
			}(),
		},
		{
			name: "storage.generate_challenges",
			endpoint: func(
				txn *transaction.Transaction,
				_ []byte,
				balances cstate.StateContextI,
			) (string, error) {
				challengesEnabled := viper.GetBool(sc.StorageChallengeEnabled)
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
			txn: transaction.Transaction{
				CreationDate: common.Timestamp(viper.GetInt64(sc.Now)),
			},
			input: nil,
		},
		// todo "update_config" waiting for PR489
	}
	var testsI []sc.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return sc.TestSuit{sc.Storage, testsI}
}
