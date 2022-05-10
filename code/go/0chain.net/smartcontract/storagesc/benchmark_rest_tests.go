package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	bk "0chain.net/smartcontract/benchmark"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"io"
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
		fmt.Println(rbt.Name()+" : ", string(prettyJSON.Bytes()))
		rbt.shownResult = true
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code %v not ok: %v", resp.StatusCode, resp.Status)
	}
	b.StartTimer()

	return nil
}

func BenchmarkRestTests(
	data bk.BenchData, _ bk.SignatureScheme,
) bk.TestSuite {
	var tests = []*RestBenchTest{
		{
			name: "getConfig",
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
			params: func() map[string]string {
				var values = make(map[string]string)
				now := common.Timestamp(time.Now().Unix())
				nar, _ := (&newAllocationRequest{
					DataShards:                 viper.GetInt(bk.NumBlobbersPerAllocation) / 2,
					ParityShards:               viper.GetInt(bk.NumBlobbersPerAllocation) / 2,
					Size:                       100 * viper.GetInt64(bk.StorageMinAllocSize),
					Expiration:                 2*common.Timestamp(viper.GetDuration(bk.StorageMinAllocDuration).Seconds()) + now,
					Owner:                      data.Clients[0],
					OwnerPublicKey:             data.PublicKeys[0],
					PreferredBlobbers:          []string{},
					ReadPriceRange:             PriceRange{0, state.Balance(viper.GetInt64(bk.StorageMaxReadPrice) * 1e10)},
					WritePriceRange:            PriceRange{0, state.Balance(viper.GetInt64(bk.StorageMaxWritePrice) * 1e10)},
					MaxChallengeCompletionTime: viper.GetDuration(bk.StorageMaxChallengeCompletionTime),
					DiversifyBlobbers:          false,
				}).encode()
				values["allocation_data"] = string(nar)
				return values
			}(),
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
			name: "getWriteMarkers",
			params: map[string]string{
				"allocation_id": getMockAllocationId(0),
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
