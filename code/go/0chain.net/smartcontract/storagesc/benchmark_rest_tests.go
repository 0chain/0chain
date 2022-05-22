package storagesc

import (
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	bk "0chain.net/smartcontract/benchmark"
	"encoding/hex"
	"encoding/json"
	"github.com/spf13/viper"
	"log"
	"time"
)

func BenchmarkRestTests(
	data bk.BenchData, sigScheme bk.SignatureScheme,
) bk.TestSuite {
	return bk.GetRestTests(
		[]bk.TestParameters{
			{
				FuncName: "get_blobber_count",
			},
			{
				FuncName: "get_blobber_total_stakes",
			},
			{
				FuncName: "get_blobber_lat_long",
			},
			{
				FuncName: "getConfig",
			},
			{
				FuncName: "transaction",
				Params: map[string]string{
					"transaction_hash": "", // todo add transactions
				},
			},
			{
				FuncName: "transactions",
				Params: map[string]string{
					"client_id":  "", // todo add transactions
					"offset":     "",
					"limit":      "",
					"block_hash": "",
				},
			},
			{
				FuncName: "errors",
				Params: map[string]string{
					"transaction_hash": "", // todo add transactions
				},
			},
			{
				FuncName: "get_block_by_hash",
				Params: map[string]string{
					"block_hash": "", // todo add blocks
				},
			},
			{
				FuncName: "total_saved_data",
			},
			{
				FuncName: "latestreadmarker",
				Params: map[string]string{
					"client":  data.Clients[0],
					"blobber": getMockBlobberId(0),
				},
			},

			{
				FuncName: "readmarkers",
				Params: map[string]string{
					"allocation_id": getMockAllocationId(0),
				},
			},
			{
				FuncName: "count_readmarkers",
				Params: map[string]string{
					"allocation_id": getMockAllocationId(0),
				},
			},
			{
				FuncName: "allocation",
				Params: map[string]string{
					"allocation": getMockAllocationId(0),
				},
			},
			{
				FuncName: "allocations",
				Params: map[string]string{
					"client": data.Clients[0],
				},
			},
			{
				FuncName: "allocation_min_lock",
				Params: map[string]string{
					"allocation_data": func() string {
						now := common.Timestamp(time.Now().Unix())
						nar, _ := (&newAllocationRequest{
							DataShards:                 viper.GetInt(bk.NumBlobbersPerAllocation) / 2,
							ParityShards:               viper.GetInt(bk.NumBlobbersPerAllocation) / 2,
							Size:                       100 * viper.GetInt64(bk.StorageMinAllocSize),
							Expiration:                 2*common.Timestamp(viper.GetDuration(bk.StorageMinAllocDuration).Seconds()) + now,
							Owner:                      data.Clients[0],
							OwnerPublicKey:             data.PublicKeys[0],
							Blobbers:                   []string{},
							ReadPriceRange:             PriceRange{0, state.Balance(viper.GetInt64(bk.StorageMaxReadPrice) * 1e10)},
							WritePriceRange:            PriceRange{0, state.Balance(viper.GetInt64(bk.StorageMaxWritePrice) * 1e10)},
							MaxChallengeCompletionTime: viper.GetDuration(bk.StorageMaxChallengeCompletionTime),
						}).encode()
						return string(nar)
					}(),
				},
			},
			{
				FuncName: "openchallenges",
				Params: map[string]string{
					"blobber": getMockBlobberId(0),
				},
			},
			{
				FuncName: "getchallenge",
				Params: map[string]string{
					"blobber":   getMockBlobberId(0),
					"challenge": getMockChallengeId(0, 0),
				},
			},
			{
				FuncName: "getblobbers",
			},
			{
				FuncName: "getBlobber",
				Params: map[string]string{
					"blobber_id": getMockBlobberId(0),
				},
			},
			{
				FuncName: "getReadPoolStat",
				Params: map[string]string{
					"client_id": data.Clients[0],
				},
			},
			{
				FuncName: "getReadPoolAllocBlobberStat",
				Params: map[string]string{
					"client_id":     data.Clients[0],
					"allocation_id": getMockAllocationId(0),
					"blobber_id":    getMockBlobberId(0),
				},
			},
			{
				FuncName: "writemarkers", // todo
				Params: map[string]string{
					"offset":        "",
					"limit":         "",
					"is_descending": "",
				},
			},
			{
				FuncName: "getWriteMarkers",
				Params: map[string]string{
					"allocation_id": getMockAllocationId(0),
					"filename":      "",
				},
			},
			{
				FuncName: "getWritePoolStat",
				Params: map[string]string{
					"client_id": data.Clients[0],
				},
			},
			{
				FuncName: "getWritePoolAllocBlobberStat",
				Params: map[string]string{
					"client_id":     data.Clients[0],
					"allocation_id": getMockAllocationId(0),
					"blobber_id":    getMockBlobberId(0),
				},
			},
			{
				FuncName: "getStakePoolStat",
				Params: map[string]string{
					"blobber_id": getMockBlobberId(0),
				},
			},
			{
				FuncName: "getUserStakePoolStat",
				Params: map[string]string{
					"client_id": data.Clients[0],
				},
			},
			{
				FuncName: "getChallengePoolStat",
				Params: map[string]string{
					"allocation_id": getMockAllocationId(0),
				},
			},
			{
				FuncName: "get_validator",
				Params: map[string]string{
					"validator_id": getMockValidatorId(0),
				},
			},
			{
				FuncName: "alloc_written_size",
				Params: map[string]string{
					"allocation_id": getMockValidatorId(0),
					"block_number":  getMockValidatorId(0),
				},
			},
			{
				FuncName: "alloc_read_size",
				Params: map[string]string{
					"allocation_id": getMockValidatorId(0),
					"block_number":  getMockValidatorId(0),
				},
			},
			{
				FuncName: "alloc_write_marker_count",
				Params: map[string]string{
					"allocation_id": getMockValidatorId(0),
				},
			},
			{
				FuncName: "collected_reward",
				Params: map[string]string{
					"start_block": getMockValidatorId(0),
					"end_block":   getMockValidatorId(0),
					"client_id":   getMockValidatorId(0),
				},
			},
			{
				FuncName: "alloc_blobbers",
				Params: map[string]string{
					"allocation_data": func() string {
						now := common.Timestamp(time.Now().Unix())
						nar, _ := (&newAllocationRequest{
							DataShards:                 viper.GetInt(bk.NumBlobbersPerAllocation) / 2,
							ParityShards:               viper.GetInt(bk.NumBlobbersPerAllocation) / 2,
							Size:                       100 * viper.GetInt64(bk.StorageMinAllocSize),
							Expiration:                 2*common.Timestamp(viper.GetDuration(bk.StorageMinAllocDuration).Seconds()) + now,
							Owner:                      data.Clients[0],
							OwnerPublicKey:             data.PublicKeys[0],
							Blobbers:                   []string{},
							ReadPriceRange:             PriceRange{0, state.Balance(viper.GetInt64(bk.StorageMaxReadPrice) * 1e10)},
							WritePriceRange:            PriceRange{0, state.Balance(viper.GetInt64(bk.StorageMaxWritePrice) * 1e10)},
							MaxChallengeCompletionTime: viper.GetDuration(bk.StorageMaxChallengeCompletionTime),
						}).encode()
						return string(nar)
					}(),
				},
			},
			{
				FuncName: "blobber_ids",
				Params: map[string]string{
					"blobber_urls": func() string {
						var urls []string
						for i := 0; i < viper.GetInt(bk.NumBlobbersPerAllocation); i++ {
							urls = append(urls, getMockBlobberUrl(i))
						}
						urlBytes, err := json.Marshal(urls)
						if err != nil {
							log.Fatal(err)
						}
						return string(urlBytes)
					}(),
				},
			},
			{
				FuncName: "free_alloc_blobbers",
				Params: map[string]string{
					"free_allocation_data": func() string {
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
						var freeBlobbers []string
						for i := 0; i < viper.GetInt(bk.StorageFasDataShards)+viper.GetInt(bk.StorageFasParityShards); i++ {
							freeBlobbers = append(freeBlobbers, getMockBlobberId(i))
						}
						bytes, _ := json.Marshal(&freeStorageAllocationInput{
							RecipientPublicKey: data.PublicKeys[1],
							Marker:             string(fsmBytes),
							Blobbers:           freeBlobbers,
						})
						return string(bytes)
					}(),
				},
			},
		},
		ADDRESS,
	)
}
