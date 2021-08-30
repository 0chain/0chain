package storagesc

import (
	"encoding/json"
	"time"

	sc "0chain.net/smartcontract/benchmark"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
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
	bt.endpoint(&bt.txn, bt.input, balances)
}

func BenchmarkTests(
	vi *viper.Viper,
	data sc.BenchData,
) []BenchTest {
	var now = common.Timestamp(vi.GetInt64(sc.Now))
	var ssc = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	return []BenchTest{
		/*
			{
				name:     "storage_read_redeem",
				endpoint: ssc.commitBlobberRead,
				txn:      transaction.Transaction{},
				input: func() []byte {
					bytes := (&ReadConnection{
						ReadMarker: &ReadMarker{
							ClientID:        data.Clients[0],
							ClientPublicKey: data.PublicKeys[0],
							BlobberID:       data.Blobbers[0],
							AllocationID:    data.Allocations[0],
							OwnerID:         data.Clients[0],
							Timestamp:       now,
							ReadCounter:     1,
							Signature: "", // todo work out how to sign
							PayerID:         data.Clients[0],
						},
					}).Encode()
					return bytes
				}(),
			},
		*/
		// data.Allocations
		{
			name:     "storage_new_allocation_request",
			endpoint: ssc.newAllocationRequest,
			txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID:     data.Clients[0],
				CreationDate: now,
				Value:        100 * vi.GetInt64(sc.StorageMinAllocSize),
			},
			input: func() []byte {
				bytes, _ := (&newAllocationRequest{
					DataShards:                 4,
					ParityShards:               4,
					Size:                       100 * vi.GetInt64(sc.StorageMinAllocSize),
					Expiration:                 common.Timestamp(vi.GetDuration(sc.StorageMinAllocDuration).Seconds()) + now,
					Owner:                      data.Clients[0],
					OwnerPublicKey:             data.PublicKeys[0],
					PreferredBlobbers:          []string{},
					ReadPriceRange:             PriceRange{0, state.Balance(vi.GetInt64(sc.StorageMaxReadPrice) * 1e10)},
					WritePriceRange:            PriceRange{0, state.Balance(vi.GetInt64(sc.StorageMaxWritePrice) * 1e10)},
					MaxChallengeCompletionTime: vi.GetDuration(sc.StorageMaxChallengeCompletionTime),
					DiversifyBlobbers:          false,
				}).encode()
				return bytes
			}(),
		},
		{
			name:     "storage_update_allocation_request",
			endpoint: ssc.updateAllocationRequest,
			txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID: data.Clients[0],
				Value:    100 * vi.GetInt64(sc.StorageMinAllocSize),
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&updateAllocationRequest{
					ID:           data.Allocations[0],
					OwnerID:      data.Clients[0],
					Size:         100 * vi.GetInt64(sc.StorageMinAllocSize),
					Expiration:   common.Timestamp(vi.GetDuration(sc.StorageMinAllocDuration).Seconds()),
					SetImmutable: true,
				})
				return bytes
			}(),
		},
		{
			name:     "storage_finalize_allocation",
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
			name:     "storage_cancel_allocation",
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
			name:     "add_free_storage_assigner",
			endpoint: ssc.addFreeStorageAssigner,
			txn: transaction.Transaction{
				ClientID: owner,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&newFreeStorageAssignerInfo{
					Name:            "mock name",
					PublicKey:       encryption.Hash("mock public key"),
					IndividualLimit: vi.GetFloat64(sc.StorageMaxIndividualFreeAllocation),
					TotalLimit:      vi.GetFloat64(sc.StorageMaxTotalFreeAllocation),
				})
				return bytes
			}(),
		},
		{
			name:     "storage_free_allocation_request",
			endpoint: ssc.freeAllocationRequest,
			txn: transaction.Transaction{
				ClientID: data.Clients[0],
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&freeStorageMarker{
					Assigner:   data.Clients[0],
					Recipient:  data.Clients[1],
					FreeTokens: vi.GetFloat64(sc.StorageMaxIndividualFreeAllocation),
					Timestamp:  1,
					Signature:  "",
				})
				return bytes
			}(),
		},
		// data.Blobbers
		{
			name:     "storage_add_blobber",
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
					Terms:             getMockBlobberTerms(vi),
					Capacity:          vi.GetInt64(sc.StorageMinBlobberCapacity) * 1000,
					StakePoolSettings: getStakePoolSettings(vi, encryption.Hash("my_new_blobber")),
				})
				return bytes
			}(),
		},
		{
			name:     "storage_add_validator",
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
					StakePoolSettings: getStakePoolSettings(vi, encryption.Hash("my_new_validator")),
				})
				return bytes
			}(),
		},
		{
			name:     "storage_blobber_health_check",
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
			name:     "storage_update_blobber_settings",
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
					Terms:             getMockBlobberTerms(vi),
					Capacity:          vi.GetInt64(sc.StorageMinBlobberCapacity) * 1000,
					StakePoolSettings: getStakePoolSettings(vi, data.Blobbers[0]),
				})
				return bytes
			}(),
		},
		// add_curator
		{
			name:     "storage_add_curator",
			endpoint: ssc.addCurator,
			txn: transaction.Transaction{
				ClientID: data.Clients[0],
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&curatorInput{
					CuratorId:    data.Clients[vi.GetInt(sc.NumCurators)],
					AllocationId: data.Allocations[0],
				})
				return bytes
			}(),
		},
		{
			name:     "storage_remove_curator",
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
			name:     "storage_new_read_pool",
			endpoint: ssc.newReadPool,
			txn:      transaction.Transaction{},
			input:    []byte{},
		},
		{
			name:     "storage_read_pool_unlock",
			endpoint: ssc.readPoolUnlock,
			txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				Value:      int64(vi.GetFloat64(sc.StorageReadPoolMinLock) * 1e10),
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
		// todo read_pool_lock, seems to be bugged, needs to be fixed before can test
		// write pool
		{
			name:     "storage_write_pool_unlock",
			endpoint: ssc.readPoolUnlock,
			txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				Value:      int64(vi.GetFloat64(sc.StorageWritePoolMinLock) * 1e10),
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
		// todo write_pool_lock, seems to be bugged, needs to be fixed before can test

		// stake pool
		{
			name:     "storage_stake_pool_lock",
			endpoint: ssc.stakePoolLock,
			txn: transaction.Transaction{
				ClientID: data.Clients[0],
				Value:    int64(vi.GetFloat64(sc.StorageStakePoolMinLock) * 1e10),
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
			name:     "storage_stake_pool_unlock",
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
			name:     "storage_stake_pool_pay_interests",
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
			name: "storage_generate_challenges",
			endpoint: func(
				txn *transaction.Transaction,
				_ []byte,
				balances cstate.StateContextI,
			) (string, error) {
				challengesEnabled := vi.GetBool(sc.StorageChallengeEnabled)
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
				CreationDate: common.Timestamp(vi.GetInt64(sc.Now)),
			},
			input: nil,
		},
		// todo "update_config" waiting for PR489
	}
}
