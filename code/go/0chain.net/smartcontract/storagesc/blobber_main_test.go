package storagesc

import (
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const (
	numBlobbers                  = 30
	blobberBalance state.Balance = 50 * x10
	clientBalance  state.Balance = 100 * x10

	aValue                int64 = 15 * x10
	aMaxReadPrice               = 10 * x10
	aMinReadPrice               = 1 * x10
	aMinWritePrice              = 2 * x10
	aMaxWritePrice              = 20 * x10
	aMaxChallengeCompTime       = 200 * time.Hour
	aRequestSize                = 2 * GB // ---- restMinLockDemand -----
	aDataShards                 = 10     // ---- restMinLockDemand -----
	aParityShards               = 10     // ---- restMinLockDemand -----
)

type mock0ChainBlobberYml struct {
	ReadPrice               float64 //token / GB for reading
	WritePrice              float64 //token / GB / time_unit for writing
	Capacity                int64   // 1 GB bytes total blobber capacity
	MinLockDemand           float64
	MaxOfferDuration        time.Duration
	ChallengeCompletionTime time.Duration
	MinStake                float64
	MaxStake                float64
	NumDelegates            int
	ServiceCharge           float64
}

var (
	blobberYaml = mock0ChainBlobberYml{
		Capacity:                2 * GB,
		ReadPrice:               1,
		WritePrice:              5,   // ---- restMinLockDemand -----
		MinLockDemand:           0.1, // ---- restMinLockDemand -----
		MaxOfferDuration:        1 * time.Hour,
		ChallengeCompletionTime: 200 * time.Second,
		MinStake:                0,
		MaxStake:                1000 * x10,
		NumDelegates:            100,
		ServiceCharge:           0.3,
	}
	aExpiration int64 = int64(toSeconds(time.Hour))
)

func TestCosts(t *testing.T) {
	var ssc = newTestStorageSC()
	var balances = newTestBalances(t, false)
	var blobbers []*Client
	var now int64 = 100
	var terms = Terms{
		ReadPrice:               convertZcnToValue(blobberYaml.ReadPrice),
		WritePrice:              convertZcnToValue(blobberYaml.WritePrice),
		MinLockDemand:           blobberYaml.MinLockDemand,
		MaxOfferDuration:        blobberYaml.MaxOfferDuration,
		ChallengeCompletionTime: blobberYaml.ChallengeCompletionTime,
	}
	var scYaml = setConfig(t, balances)

	// attach blobbers
	for i := 0; i < numBlobbers; i++ {
		var blobber = addBlobber(t, ssc, blobberYaml.Capacity, now, terms, blobberBalance, balances)
		blobbers = append(blobbers, blobber)
	}

	// new allocation
	now += 100
	var client = newClient(clientBalance, balances)
	var nar = newAllocationRequest{
		DataShards:                 aDataShards,
		ParityShards:               aParityShards,
		Expiration:                 common.Timestamp(aExpiration),
		Owner:                      client.id,
		OwnerPublicKey:             client.pk,
		ReadPriceRange:             PriceRange{aMinReadPrice, aMaxReadPrice},
		WritePriceRange:            PriceRange{aMinWritePrice, aMaxWritePrice},
		Size:                       aRequestSize,
		MaxChallengeCompletionTime: aMaxChallengeCompTime,
	}
	resp, err := nar.callNewAllocReq(t, client.id, state.Balance(aValue), ssc, now, balances)
	require.NoError(t, err)
	var decodeResp StorageAllocation
	require.NoError(t, decodeResp.Decode([]byte(resp)))
	alloc, err := ssc.getAllocation(decodeResp.ID, balances)
	require.NoError(t, err)

	t.Run("new allocation", func(t *testing.T) {
		alloc = alloc
		scYaml = scYaml
		var f formulae = formulae{
			blobber: blobberYaml,
			sc:      *scYaml,
			ar:      nar,
		}
		require.EqualValues(t, f.ResMinLockDemandTotal(common.Timestamp(now)), alloc.restMinLockDemand())
	})

}

// ConvertToValue converts ZCN tokens to value
func convertZcnToValue(token float64) state.Balance {
	return state.Balance(token * float64(x10))
}
