package event

import (
	"testing"

	"0chain.net/core/common"
	"github.com/stretchr/testify/require"
)

func TestQueryDataForBlobber(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()
	ids := setUpBlobbers(t, edb, 10, false)
	var blobber1, blobber2 Blobber
	blobber1.ID = ids[0]
	blobber1.WritePrice = 176
	blobber1.ReadPrice = 1111
	blobber1.TotalStake = 23
	blobber1.NotAvailable = false
	blobber1.LastHealthCheck = common.Timestamp(123)
	blobber1.BaseURL = "http://random_blobber_1.com"

	blobber2.ID = ids[1]
	blobber2.WritePrice = 17
	blobber2.ReadPrice = 1
	blobber2.TotalStake = 14783
	blobber2.NotAvailable = false
	blobber2.LastHealthCheck = common.Timestamp(3333333331)
	blobber2.BaseURL = "http://random_blobber_2.com"

	require.NoError(t, edb.updateBlobber([]Blobber{blobber1, blobber2}))

	blobbers, err := edb.GetQueryData("id,write_price,read_price,total_stake,not_available,last_health_check,base_url", &Blobber{})
	require.NoError(t, err)
	require.Len(t, blobbers, 2)
	require.EqualValues(t, blobber1, blobbers[0])
	require.EqualValues(t, blobber2, blobbers[1])

}
