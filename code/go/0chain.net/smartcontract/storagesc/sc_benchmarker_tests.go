package storagesc

import (
	"encoding/json"
	"strconv"

	sc "0chain.net/smartcontract"

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
		{
			Name:     "new_allocation_request",
			Endpoint: ssc.newAllocationRequest,
			Txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("mock transaction hash"),
				},
				ClientID:     clients[0],
				CreationDate: now,
				Value:        vi.GetInt64(sc.StorageMinAllocSize),
			},
			Input: func() []byte {
				bytes, _ := (&newAllocationRequest{
					DataShards:                 4,
					ParityShards:               4,
					Size:                       vi.GetInt64(sc.StorageMinAllocSize),
					Expiration:                 common.Timestamp(vi.GetDuration(sc.StorageMinAllocDuration).Seconds()) + now,
					Owner:                      clients[0],
					OwnerPublicKey:             keys[0],
					PreferredBlobbers:          []string{},
					ReadPriceRange:             PriceRange{0, state.Balance(vi.GetInt64(sc.StorageMaxReadPrice))},
					WritePriceRange:            PriceRange{0, state.Balance(vi.GetInt64(sc.StorageMaxWritePrice))},
					MaxChallengeCompletionTime: vi.GetDuration("max_challenge_completionTime"),
					DiversifyBlobbers:          false,
				}).encode()
				return bytes
			}(),
		},
		{
			Name:     "new_read_pool",
			Endpoint: ssc.newReadPool,
			Txn:      transaction.Transaction{},
			Input:    []byte{},
		},
		{
			Name:     "stake_pool_pay_interests",
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
