package storagesc

import (
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/rest/restinterface"
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
	rh := restinterface.NewTestRestHandler()
	srh := NewStorageRestHandler(rh)
	return bk.GetRestTests(
		[]bk.TestParameters{
			{
				FuncName: "get_blobber_count",
				Endpoint: srh.getBlobberCount,
			},
			{
				FuncName: "get_blobber_total_stakes",
				Endpoint: srh.getBlobberTotalStakes,
			},
			{
				FuncName: "get_blobber_lat_long",
				Endpoint: srh.getBlobberGeoLocation,
			},
			{
				FuncName: "getConfig",
				Endpoint: srh.getConfig,
			},
			{
				FuncName: "transaction",
				Params: map[string]string{
					"transaction_hash": "", // todo add transactions
				},
				Endpoint: srh.getTransactionByHash,
			},
			{
				FuncName: "transactions",
				Params: map[string]string{
					"client_id":  "", // todo add transactions
					"offset":     "",
					"limit":      "",
					"block_hash": "",
				},
				Endpoint: srh.getTransactionByFilter,
			},
			{
				FuncName: "errors",
				Params: map[string]string{
					"transaction_hash": "", // todo add transactions
				},
				Endpoint: srh.getErrors,
			},
			{
				FuncName: "get_block_by_hash",
				Params: map[string]string{
					"block_hash": "", // todo add blocks
				},
				Endpoint: srh.getBlockByHash,
			},
			{
				FuncName: "total_saved_data",
				Endpoint: srh.getTotalData,
			},
			{
				FuncName: "latestreadmarker",
				Params: map[string]string{
					"client":  data.Clients[0],
					"blobber": getMockBlobberId(0),
				},
				Endpoint: srh.getLatestReadMarker,
			},

			{
				FuncName: "readmarkers",
				Params: map[string]string{
					"allocation_id": getMockAllocationId(0),
				},
				Endpoint: srh.getReadMarkers,
			},
			{
				FuncName: "count_readmarkers",
				Params: map[string]string{
					"allocation_id": getMockAllocationId(0),
				},
				Endpoint: srh.getReadMarkersCount,
			},
			{
				FuncName: "allocation",
				Params: map[string]string{
					"allocation": getMockAllocationId(0),
				},
				Endpoint: srh.getAllocation,
			},
			{
				FuncName: "allocations",
				Params: map[string]string{
					"client": data.Clients[0],
				},
				Endpoint: srh.getAllocations,
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
				Endpoint: srh.getAllocationMinLock,
			},
			{
				FuncName: "openchallenges",
				Params: map[string]string{
					"blobber": getMockBlobberId(0),
				},
				Endpoint: srh.getOpenChallenges,
			},
			{
				FuncName: "getchallenge",
				Params: map[string]string{
					"blobber":   getMockBlobberId(0),
					"challenge": getMockChallengeId(0, 0),
				},
				Endpoint: srh.getChallenge,
			},
			{
				FuncName: "getblobbers",
				Endpoint: srh.getBlobbers,
			},
			{
				FuncName: "getBlobber",
				Params: map[string]string{
					"blobber_id": getMockBlobberId(0),
				},
				Endpoint: srh.getBlobber,
			},
			{
				FuncName: "getReadPoolStat",
				Params: map[string]string{
					"client_id": data.Clients[0],
				},
				Endpoint: srh.getReadPoolStat,
			},
			{
				FuncName: "writemarkers", // todo
				Params: map[string]string{
					"offset":        "",
					"limit":         "",
					"is_descending": "",
				},
				Endpoint: srh.getWriteMarker,
			},
			{
				FuncName: "getWriteMarkers",
				Params: map[string]string{
					"allocation_id": getMockAllocationId(0),
					"filename":      "",
				},
				Endpoint: srh.getWriteMarkers,
			},
			{
				FuncName: "getWritePoolStat",
				Params: map[string]string{
					"client_id": data.Clients[0],
				},
				Endpoint: srh.getWritePoolStat,
			},
			{
				FuncName: "getStakePoolStat",
				Params: map[string]string{
					"blobber_id": getMockBlobberId(0),
				},
				Endpoint: srh.getStakePoolStat,
			},
			{
				FuncName: "getUserStakePoolStat",
				Params: map[string]string{
					"client_id": data.Clients[0],
				},
				Endpoint: srh.getUserStakePoolStat,
			},
			{
				FuncName: "getChallengePoolStat",
				Params: map[string]string{
					"allocation_id": getMockAllocationId(0),
				},
				Endpoint: srh.getChallengePoolStat,
			},
			{
				FuncName: "get_validator",
				Params: map[string]string{
					"validator_id": getMockValidatorId(0),
				},
				Endpoint: srh.getValidator,
			},
			{
				FuncName: "alloc_written_size",
				Params: map[string]string{
					"allocation_id": getMockValidatorId(0),
					"block_number":  getMockValidatorId(0),
				},
				Endpoint: srh.getWrittenAmount,
			},
			{
				FuncName: "alloc_read_size",
				Params: map[string]string{
					"allocation_id": getMockValidatorId(0),
					"block_number":  getMockValidatorId(0),
				},
				Endpoint: srh.getReadAmount,
			},
			{
				FuncName: "alloc_write_marker_count",
				Params: map[string]string{
					"allocation_id": getMockValidatorId(0),
				},
				Endpoint: srh.getWriteMarkerCount,
			},
			{
				FuncName: "collected_reward",
				Params: map[string]string{
					"start_block": getMockValidatorId(0),
					"end_block":   getMockValidatorId(0),
					"client_id":   getMockValidatorId(0),
				},
				Endpoint: srh.getCollectedReward,
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
				Endpoint: srh.getAllocationBlobbers,
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
				Endpoint: srh.getBlobberIdsByUrls,
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
				Endpoint: srh.getFreeAllocationBlobbers,
			},
		},
		ADDRESS,
		srh,
	)
}
