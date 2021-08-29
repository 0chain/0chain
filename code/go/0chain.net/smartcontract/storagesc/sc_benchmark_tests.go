package storagesc

import (
	"encoding/json"
	"strconv"
	"time"

	sc "0chain.net/smartcontract/benchmark"

	"0chain.net/chaincore/state"
	"0chain.net/core/encryption"

	"github.com/spf13/viper"

	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

func BenchmarkTests(
	vi *viper.Viper,
	clients []string,
	keys []string,
	blobbers []string,
	allocations []string,
) []sc.BenchTest {
	var now = common.Timestamp(vi.GetInt64(sc.Now))
	var ssc = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	return []sc.BenchTest{
		/*
			{
				Name:     "storage_read_redeem",
				Endpoint: ssc.commitBlobberRead,
				Txn:      transaction.Transaction{},
				Input: func() []byte {
					bytes := (&ReadConnection{
						ReadMarker: &ReadMarker{
							ClientID:        clients[0],
							ClientPublicKey: keys[0],
							BlobberID:       blobbers[0],
							AllocationID:    allocations[0],
							OwnerID:         clients[0],
							Timestamp:       now,
							ReadCounter:     1,
							Signature: "", // todo work out how to sign
							PayerID:         clients[0],
						},
					}).Encode()
					return bytes
				}(),
			},
		*/
		// allocations
		{
			Name:     "storage_new_allocation_request",
			Endpoint: ssc.newAllocationRequest,
			Txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID:     clients[0],
				CreationDate: now,
				Value:        100 * vi.GetInt64(sc.StorageMinAllocSize),
			},
			Input: func() []byte {
				bytes, _ := (&newAllocationRequest{
					DataShards:                 4,
					ParityShards:               4,
					Size:                       100 * vi.GetInt64(sc.StorageMinAllocSize),
					Expiration:                 common.Timestamp(vi.GetDuration(sc.StorageMinAllocDuration).Seconds()) + now,
					Owner:                      clients[0],
					OwnerPublicKey:             keys[0],
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
			Name:     "storage_update_allocation_request",
			Endpoint: ssc.updateAllocationRequest,
			Txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID: clients[0],
				Value:    100 * vi.GetInt64(sc.StorageMinAllocSize),
			},
			Input: func() []byte {
				bytes, _ := json.Marshal(&updateAllocationRequest{
					ID:           allocations[0],
					OwnerID:      clients[0],
					Size:         100 * vi.GetInt64(sc.StorageMinAllocSize),
					Expiration:   common.Timestamp(vi.GetDuration(sc.StorageMinAllocDuration).Seconds()),
					SetImmutable: true,
				})
				return bytes
			}(),
		},
		{
			Name:     "storage_finalize_allocation",
			Endpoint: ssc.finalizeAllocation,
			Txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: common.Timestamp((time.Hour * 1000).Seconds()) + now,
				ClientID:     clients[0],
				ToClientID:   ADDRESS,
			},
			Input: func() []byte {
				bytes, _ := json.Marshal(&lockRequest{
					AllocationID: allocations[0],
				})
				return bytes
			}(),
		},
		{
			Name:     "storage_cancel_allocation",
			Endpoint: ssc.cancelAllocationRequest,
			Txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: now + 1,
				ClientID:     clients[0],
				ToClientID:   ADDRESS,
			},
			Input: func() []byte {
				bytes, _ := json.Marshal(&lockRequest{
					AllocationID: allocations[0],
				})
				return bytes
			}(),
		},
		// blobbers
		{
			Name:     "storage_add_blobber",
			Endpoint: ssc.addBlobber,
			Txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: now + 1,
				ClientID:     clients[0],
				ToClientID:   ADDRESS,
			},
			Input: func() []byte {
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
			Name:     "storage_add_validator",
			Endpoint: ssc.addValidator,
			Txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: now + 1,
				ClientID:     clients[0],
				ToClientID:   ADDRESS,
			},
			Input: func() []byte {
				bytes, _ := json.Marshal(&ValidationNode{
					ID:                encryption.Hash("my_new_validator"),
					BaseURL:           "my_new_validator.com",
					StakePoolSettings: getStakePoolSettings(vi, encryption.Hash("my_new_validator")),
				})
				return bytes
			}(),
		},
		{
			Name:     "storage_blobber_health_check",
			Endpoint: ssc.blobberHealthCheck,
			Txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: now + 1,
				ClientID:     blobbers[0],
				ToClientID:   ADDRESS,
			},
			Input: []byte{},
		},
		{
			Name:     "update_blobber_settings",
			Endpoint: ssc.updateBlobberSettings,
			Txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				CreationDate: now + 1,
				ClientID:     blobbers[0],
				ToClientID:   ADDRESS,
			},
			Input: func() []byte {
				bytes, _ := json.Marshal(&StorageNode{
					ID:                blobbers[0],
					Terms:             getMockBlobberTerms(vi),
					Capacity:          vi.GetInt64(sc.StorageMinBlobberCapacity) * 1000,
					StakePoolSettings: getStakePoolSettings(vi, blobbers[0]),
				})
				return bytes
			}(),
		},
		// add_curator
		{
			Name:     "storage_add_curator",
			Endpoint: ssc.addCurator,
			Txn: transaction.Transaction{
				ClientID: clients[0],
			},
			Input: func() []byte {
				bytes, _ := json.Marshal(&curatorInput{
					CuratorId:    clients[vi.GetInt(sc.NumCurators)],
					AllocationId: allocations[0],
				})
				return bytes
			}(),
		},
		{
			Name:     "storage_remove_curator",
			Endpoint: ssc.removeCurator,
			Txn: transaction.Transaction{
				ClientID: clients[0],
			},
			Input: func() []byte {
				bytes, _ := json.Marshal(&curatorInput{
					CuratorId:    clients[0],
					AllocationId: allocations[0],
				})
				return bytes
			}(),
		},
		// read_pool
		{
			Name:     "storage_new_read_pool",
			Endpoint: ssc.newReadPool,
			Txn:      transaction.Transaction{},
			Input:    []byte{},
		}, /*
			{
				Name:     "storage_read_pool_unlock",
				Endpoint: ssc.readPoolUnlock,
				Txn: transaction.Transaction{
					HashIDField: datastore.HashIDField{
						Hash: encryption.Hash("mock transaction hash"),
					},
					Value:      vi.GetInt64(sc.StorageReadPoolMinLock),
					ClientID:   clients[0],
					ToClientID: ADDRESS,
				},
				Input: func() []byte {
					bytes, _ := json.Marshal(&unlockRequest{
						PoolID: allocations[0],
					})
					return bytes
				}(),
			},*/
		{
			Name:     "storage_read_pool_lock",
			Endpoint: ssc.readPoolLock,
			Txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				Value:      vi.GetInt64(sc.StorageReadPoolMinLock),
				ClientID:   clients[0],
				ToClientID: ADDRESS,
			},
			Input: func() []byte {
				bytes, _ := json.Marshal(&lockRequest{
					AllocationID: allocations[0],
					TargetId:     getMockReadPoolId(0, 0, 0),
					Duration:     vi.GetDuration(sc.StorageReadPoolMinLockPeriod),
				})
				return bytes
			}(),
		},
		// write pool

		// stake pool
		{
			Name:     "storage_stake_pool_pay_interests",
			Endpoint: ssc.stakePoolPayInterests,
			Txn:      transaction.Transaction{},
			Input: func() []byte {
				bytes, _ := json.Marshal(&stakePoolRequest{
					BlobberID: blobbers[0],
					PoolID:    blobbers[0] + "Pool" + strconv.Itoa(0),
				})
				return bytes
			}(),
		},
	}
}
