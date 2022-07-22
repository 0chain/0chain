package storagesc

import (
	"strconv"

	"0chain.net/core/encryption"

	"0chain.net/smartcontract/dbs/benchmark"

	"0chain.net/chaincore/currency"

	"encoding/hex"
	"encoding/json"
	"log"

	"0chain.net/core/common"
	bk "0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/rest"
	"github.com/spf13/viper"
)

func BenchmarkRestTests(
	data bk.BenchData, sigScheme bk.SignatureScheme,
) bk.TestSuite {
	rh := rest.NewRestHandler(&rest.TestQueryChainer{})
	srh := NewStorageRestHandler(rh)
	maxReadPrice, err := currency.ParseZCN(viper.GetFloat64(bk.StorageMaxReadPrice))
	if err != nil {
		panic(err)
	}
	maxWritePrice, err := currency.ParseZCN(viper.GetFloat64(bk.StorageMaxWritePrice))
	if err != nil {
		panic(err)
	}
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
				FuncName: "total-blobber-capacity",
				Endpoint: srh.getTotalBlobberCapacity,
			},
			{
				FuncName: "blobbers-by-geolocation",
				Params: map[string]string{
					"max_latitude":  "40",
					"min_latitude":  "-40",
					"max_longitude": "40",
					"min_longitude": "-40",
				},
				Endpoint: srh.getBlobbersByGeoLocation,
			},
			{
				FuncName: "storage_config",
				Endpoint: srh.getConfig,
			},
			{
				FuncName: "get_blocks",
				Endpoint: srh.getBlocks,
			},
			{
				FuncName: "transaction",
				Params: map[string]string{
					"transaction_hash": benchmark.GetMockTransactionHash(1, 1),
				},
				Endpoint: srh.getTransactionByHash,
			},
			{
				FuncName: "transactions",
				Params: map[string]string{
					"client_id":    data.Clients[1],
					"to_client_id": data.Clients[2],
					"block_hash":   benchmark.GetMockBlockHash(1),
					"block-start":  "1",
					"block-end":    "100",
				},
				Endpoint: srh.getTransactionByFilter,
			},
			{
				FuncName: "transactions",
				Params: map[string]string{
					"look_up_hash": benchmark.GetMockWriteMarkerLookUpHash(1, 1),
					"name":         benchmark.GetMockWriteMarkerContentHash(1, 1),
					"content_hash": benchmark.GetMockWriteMarkerFileName(1),
				},
				Endpoint: srh.getTransactionHashesByFilter,
			},
			{
				FuncName: "errors",
				Params: map[string]string{
					"transaction_hash": benchmark.GetMockTransactionHash(3, 3),
				},
				Endpoint: srh.getErrors,
			},
			{
				FuncName: "get_block",
				Params: map[string]string{
					"block_hash": benchmark.GetMockBlockHash(1),
					"date":       strconv.FormatInt(int64(data.Now.Duration()), 10),
					"round":      "1",
				},
				Endpoint: srh.getBlock,
			},
			{
				FuncName: "total-saved-data",
				Endpoint: srh.getTotalData,
			},
			{
				FuncName: "average-write-price",
				Endpoint: srh.getAverageWritePrice,
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
					"limit":  "20",
					"offset": "1",
				},
				Endpoint: srh.getAllocations,
			},
			{
				FuncName: "allocation_min_lock",
				Params: map[string]string{
					"allocation_data": func() string {
						nar, _ := (&newAllocationRequest{
							DataShards:      viper.GetInt(bk.NumBlobbersPerAllocation) / 2,
							ParityShards:    viper.GetInt(bk.NumBlobbersPerAllocation) / 2,
							Size:            100 * viper.GetInt64(bk.StorageMinAllocSize),
							Expiration:      2 * common.Timestamp(viper.GetDuration(bk.StorageMinAllocDuration).Seconds()),
							Owner:           data.Clients[0],
							OwnerPublicKey:  data.PublicKeys[0],
							Blobbers:        []string{},
							ReadPriceRange:  PriceRange{0, maxReadPrice},
							WritePriceRange: PriceRange{0, maxWritePrice},
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
				FuncName: "blobber-rank",
				Params: map[string]string{
					"id": getMockBlobberId(3),
				},
				Endpoint: srh.getBlobberRank,
			},
			{
				FuncName: "getchallenge",
				Params: map[string]string{
					"blobber":   getMockBlobberId(0),
					"challenge": getMockChallengeId(encryption.Hash("0"), encryption.Hash("0")),
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
				FuncName: "writemarkers",
				Params: map[string]string{
					"offset":        "",
					"limit":         "",
					"is_descending": "true",
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
				FuncName: "validators",
				Endpoint: srh.validators,
			},
			{
				FuncName: "alloc_written_size",
				Params: map[string]string{
					"allocation_id": getMockAllocationId(0),
					"block_number":  "1",
				},
				Endpoint: srh.getWrittenAmount,
			},
			{
				FuncName: "allocWrittenSizePerPeriod",
				Params: map[string]string{
					"block-start": "1",
					"block-end":   "100",
				},
				Endpoint: srh.getWrittenAmountPerPeriod,
			},
			{
				FuncName: "alloc_read_size",
				Params: map[string]string{
					"allocation_id": getMockAllocationId(0),
					"block_number":  "1",
				},
				Endpoint: srh.getReadAmount,
			},
			{
				FuncName: "alloc_write_marker_count",
				Params: map[string]string{
					"allocation_id": getMockAllocationId(0),
				},
				Endpoint: srh.getWriteMarkerCount,
			},
			{
				FuncName: "collected_reward",
				Params: map[string]string{
					"start_block": "1",
					"end_block":   "100",
					"client_id":   data.Clients[1],
				},
				Endpoint: srh.getCollectedReward,
			},
			{
				FuncName: "alloc_blobbers",
				Params: map[string]string{
					"allocation_data": func() string {
						//now := common.Timestamp(time.Now().Unix())
						nar, _ := (&newAllocationRequest{
							DataShards:      viper.GetInt(bk.NumBlobbersPerAllocation) / 2,
							ParityShards:    viper.GetInt(bk.NumBlobbersPerAllocation) / 2,
							Size:            100 * viper.GetInt64(bk.StorageMinAllocSize),
							Expiration:      2 * common.Timestamp(viper.GetDuration(bk.StorageMinAllocDuration).Seconds()),
							Owner:           data.Clients[0],
							OwnerPublicKey:  data.PublicKeys[0],
							Blobbers:        []string{},
							ReadPriceRange:  PriceRange{0, maxReadPrice},
							WritePriceRange: PriceRange{0, maxWritePrice},
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
		bk.StorageRest,
	)
}
