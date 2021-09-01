package storagesc

import (
	"context"
	"net/url"
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

func (bt RestBenchTest) Transaction() transaction.Transaction {
	return transaction.Transaction{}
}

func (rbt RestBenchTest) Run(balances cstate.StateContextI) {
	_, err := rbt.endpoint(context.TODO(), rbt.params, balances)
	if err != nil {
		panic(err)
	}
}

func BenchmarkRestTests(
	data bk.BenchData, sigScheme bk.SignatureScheme,
) bk.TestSuit {
	var ssc = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	ssc.setSC(ssc.SmartContract, &smartcontract.BCContext{})
	var tests = []RestBenchTest{
		{
			name:     "getConfig",
			endpoint: ssc.getConfigHandler,
		},
		{
			name:     "latestreadmarker",
			endpoint: ssc.LatestReadMarkerHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("client", data.Clients[0])
				values.Set("blobber", getMockBlobberId(0))
				return values
			}(),
		},
		{
			name:     "allocation",
			endpoint: ssc.AllocationStatsHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("allocation", getMockAllocationId(0))
				return values
			}(),
		},
		{
			name:     "allocations",
			endpoint: ssc.GetAllocationsHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("client", data.Clients[0])
				return values
			}(),
		},
		{
			name:     "allocation_min_lock",
			endpoint: ssc.GetAllocationMinLockHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				now := common.Timestamp(time.Now().Unix())
				nar, _ := ((&newAllocationRequest{
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
				}).encode())
				values.Set("allocation_data", string(nar))
				return values
			}(),
		},
		{
			name:     "openchallenges",
			endpoint: ssc.OpenChallengeHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("blobber", getMockBlobberId(0))
				return values
			}(),
		},
		{
			name:     "getchallenge",
			endpoint: ssc.GetChallengeHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("blobber", getMockBlobberId(0))
				values.Set("challenge", getMockChallengeId(getMockBlobberId(0), 0))
				return values
			}(),
		},
		{
			name:     "getblobbers",
			endpoint: ssc.GetBlobbersHandler,
		},
		{
			name:     "getBlobber",
			endpoint: ssc.GetBlobberHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("blobber_id", getMockBlobberId(0))
				return values
			}(),
		},
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuit{bk.StorageRest, testsI}
}
