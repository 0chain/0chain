package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	bk "0chain.net/smartcontract/benchmark"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type RestBenchTest struct {
	name        string
	params      map[string]string
	shownResult bool
}

func (rbt *RestBenchTest) Name() string {
	return "storage_rest." + rbt.name
}

func (rbt *RestBenchTest) Transaction() *transaction.Transaction {
	return &transaction.Transaction{}
}

func (rbt *RestBenchTest) Run(balances cstate.StateContextI, b *testing.B) error {
	b.StopTimer()
	req := httptest.NewRequest("GET", "http://localhost/v1/screst/"+ADDRESS+"/"+rbt.name, nil)
	rec := httptest.NewRecorder()
	if len(rbt.params) > 0 {
		q := req.URL.Query()
		for k, v := range rbt.params {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	b.StartTimer()

	http.DefaultServeMux.ServeHTTP(rec, req)

	b.StopTimer()
	resp := rec.Result()
	if viper.GetBool(bk.ShowOutput) && !rbt.shownResult {
		body, _ := io.ReadAll(resp.Body)
		var prettyJSON bytes.Buffer
		err := json.Indent(&prettyJSON, body, "", "\t")
		require.NoError(b, err)
		fmt.Println(req.URL.String()+" : ", prettyJSON.String())
		rbt.shownResult = true
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code %v not ok: %v", resp.StatusCode, resp.Status)
	}
	b.StartTimer()

	return nil
}

func BenchmarkRestTests(
	data bk.BenchData, sigScheme bk.SignatureScheme,
) bk.TestSuite {
	var tests = []*RestBenchTest{
		{
			name: "get_blobber_count",
		},
		{
			name: "get_blobber_total_stakes",
		},
		{
			name: "get_blobber_lat_long",
		},
		{
			name: "getConfig",
		},
		{
			name: "transaction",
			params: map[string]string{
				"transaction_hash": "", // todo add transactions
			},
		},
		{
			name: "transactions",
			params: map[string]string{
				"client_id":  "", // todo add transactions
				"offset":     "",
				"limit":      "",
				"block_hash": "",
			},
		},
		{
			name: "errors",
			params: map[string]string{
				"transaction_hash": "", // todo add transactions
			},
		},
		{
			name: "get_block_by_hash",
			params: map[string]string{
				"block_hash": "", // todo add blocks
			},
		},
		{
			name: "total_saved_data",
		},
		{
			name: "latestreadmarker",
			params: map[string]string{
				"client":  data.Clients[0],
				"blobber": getMockBlobberId(0),
			},
		},

		{
			name: "readmarkers",
			params: map[string]string{
				"allocation_id": getMockAllocationId(0),
			},
		},
		{
			name: "count_readmarkers",
			params: map[string]string{
				"allocation_id": getMockAllocationId(0),
			},
		},
		{
			name: "allocation",
			params: map[string]string{
				"allocation": getMockAllocationId(0),
			},
		},
		{
			name: "allocations",
			params: map[string]string{
				"client": data.Clients[0],
			},
		},
		{
			name: "allocation_min_lock",
			params: map[string]string{
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
			name: "openchallenges",
			params: map[string]string{
				"blobber": getMockBlobberId(0),
			},
		},
		{
			name: "getchallenge",
			params: map[string]string{
				"blobber":   getMockBlobberId(0),
				"challenge": getMockChallengeId(0, 0),
			},
		},
		{
			name: "getblobbers",
		},
		{
			name: "getBlobber",
			params: map[string]string{
				"blobber_id": getMockBlobberId(0),
			},
		},
		{
			name: "getReadPoolStat",
			params: map[string]string{
				"client_id": data.Clients[0],
			},
		},
		{
			name: "getReadPoolAllocBlobberStat",
			params: map[string]string{
				"client_id":     data.Clients[0],
				"allocation_id": getMockAllocationId(0),
				"blobber_id":    getMockBlobberId(0),
			},
		},
		{
			name: "writemarkers", // todo
			params: map[string]string{
				"offset":        "",
				"limit":         "",
				"is_descending": "",
			},
		},
		{
			name: "getWriteMarkers",
			params: map[string]string{
				"allocation_id": getMockAllocationId(0),
				"filename":      "",
			},
		},
		{
			name: "getWritePoolStat",
			params: map[string]string{
				"client_id": data.Clients[0],
			},
		},
		{
			name: "getWritePoolAllocBlobberStat",
			params: map[string]string{
				"client_id":     data.Clients[0],
				"allocation_id": getMockAllocationId(0),
				"blobber_id":    getMockBlobberId(0),
			},
		},
		{
			name: "getStakePoolStat",
			params: map[string]string{
				"blobber_id": getMockBlobberId(0),
			},
		},
		{
			name: "getUserStakePoolStat",
			params: map[string]string{
				"client_id": data.Clients[0],
			},
		},
		{
			name: "getChallengePoolStat",
			params: map[string]string{
				"allocation_id": getMockAllocationId(0),
			},
		},
		{
			name: "get_validator",
			params: map[string]string{
				"validator_id": getMockValidatorId(0),
			},
		},
		{
			name: "alloc_written_size",
			params: map[string]string{
				"allocation_id": getMockValidatorId(0),
				"block_number":  getMockValidatorId(0),
			},
		},
		{
			name: "alloc_read_size",
			params: map[string]string{
				"allocation_id": getMockValidatorId(0),
				"block_number":  getMockValidatorId(0),
			},
		},
		{
			name: "alloc_write_marker_count",
			params: map[string]string{
				"allocation_id": getMockValidatorId(0),
			},
		},
		{
			name: "collected_reward",
			params: map[string]string{
				"start_block": getMockValidatorId(0),
				"end_block":   getMockValidatorId(0),
				"client_id":   getMockValidatorId(0),
			},
		},
		{
			name: "alloc_blobbers",
			params: map[string]string{
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
			name: "blobber_ids",
			params: map[string]string{
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
			name: "free_alloc_blobbers",
			params: map[string]string{
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
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuite{
		Source:     bk.StorageRest,
		Benchmarks: testsI,
		ReadOnly:   true,
	}
}
