package storagesc

import (
	"context"
	"net/url"
	"testing"
	"time"

	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"github.com/spf13/viper"

	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"

	cstate "0chain.net/chaincore/chain/state"
	bk "0chain.net/smartcontract/benchmark"
)

type RestBenchTest struct {
	name     string
	endpoint func(
		context.Context,
		url.Values,
		cstate.StateContextI,
	) (interface{}, error)
	params url.Values
}

func (rbt RestBenchTest) Name() string {
	return rbt.name
}

func (rbt RestBenchTest) Transaction() *transaction.Transaction {
	return &transaction.Transaction{}
}

func (rbt RestBenchTest) Run(balances cstate.StateContextI, _ *testing.B) error {
	_, err := rbt.endpoint(context.TODO(), rbt.params, balances)
	return err
}

func BenchmarkRestTests(
	data bk.BenchData, _ bk.SignatureScheme,
) bk.TestSuite {
	var ssc = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	ssc.setSC(ssc.SmartContract, &smartcontract.BCContext{})
	var tests = []RestBenchTest{
		{
			name:     "storage_rest.getConfig",
			endpoint: ssc.getConfigHandler,
		},
		{
			name:     "storage_rest.get_mpt_key.sc_config",
			endpoint: ssc.GetMptKey,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("key", scConfigKey(ADDRESS))
				return values
			}(),
		},
		{
			name:     "storage_rest.latestreadmarker",
			endpoint: ssc.LatestReadMarkerHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("client", data.Clients[0])
				values.Set("blobber", getMockBlobberId(0))
				return values
			}(),
		},
		{
			name:     "storage_rest.allocation",
			endpoint: ssc.AllocationStatsHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("allocation", getMockAllocationId(0))
				return values
			}(),
		},
		{
			name:     "storage_rest.allocations",
			endpoint: ssc.GetAllocationsHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("client", data.Clients[0])
				return values
			}(),
		},
		{
			name:     "storage_rest.allocation_min_lock",
			endpoint: ssc.GetAllocationMinLockHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
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
				values.Set("allocation_data", string(nar))
				return values
			}(),
		},
		{
			name:     "storage_rest.openchallenges",
			endpoint: ssc.OpenChallengeHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("blobber", getMockBlobberId(0))
				return values
			}(),
		},
		{
			name:     "storage_rest.getchallenge",
			endpoint: ssc.GetChallengeHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("blobber", getMockBlobberId(0))
				values.Set("challenge", getMockChallengeId(0, 0))
				return values
			}(),
		},
		{
			name:     "storage_rest.getblobbers",
			endpoint: ssc.GetBlobbersHandler,
		},
		{
			name:     "storage_rest.getBlobber",
			endpoint: ssc.GetBlobberHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("blobber_id", getMockBlobberId(0))
				return values
			}(),
		},
		{
			name:     "storage_rest.getReadPoolStat",
			endpoint: ssc.getReadPoolStatHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("client_id", data.Clients[0])
				return values
			}(),
		},
		{
			name:     "storage_rest.getReadPoolAllocBlobberStat",
			endpoint: ssc.getReadPoolAllocBlobberStatHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("client_id", data.Clients[0])
				values.Set("allocation_id", getMockAllocationId(0))
				values.Set("blobber_id", getMockBlobberId(0))
				return values
			}(),
		},
		{
			name:     "storage_rest.getWriteMarkers",
			endpoint: ssc.GetWriteMarkersHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("allocation_id", getMockAllocationId(0))
				return values
			}(),
		},
		{
			name:     "storage_rest.getWritePoolStat",
			endpoint: ssc.getWritePoolStatHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("client_id", data.Clients[0])
				return values
			}(),
		},
		{
			name:     "storage_rest.getWritePoolAllocBlobberStat",
			endpoint: ssc.getWritePoolAllocBlobberStatHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("client_id", data.Clients[0])
				values.Set("allocation_id", getMockAllocationId(0))
				values.Set("blobber_id", getMockBlobberId(0))
				return values
			}(),
		},
		{
			name:     "storage_rest.getStakePoolStat",
			endpoint: ssc.getStakePoolStatHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("blobber_id", getMockBlobberId(0))
				return values
			}(),
		},
		{
			name:     "storage_rest.getUserStakePoolStat",
			endpoint: ssc.getUserStakePoolStatHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("client_id", data.Clients[0])
				return values
			}(),
		},
		{
			name:     "storage_rest.getChallengePoolStat",
			endpoint: ssc.getChallengePoolStatHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("allocation_id", getMockAllocationId(0))
				return values
			}(),
		},
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuite{
		Source:     bk.StorageRest,
		Benchmarks: testsI,
	}
}
