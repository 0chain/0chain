package event

import (
	"fmt"
	"testing"

	"0chain.net/core/common"

	"go.uber.org/zap"

	"github.com/0chain/common/core/logging"

	"github.com/stretchr/testify/require"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestUpdateBlobber(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	ids := setUpBlobbers(t, edb, 10)
	var blobber1, blobber2 Blobber
	blobber1.ID = ids[0]
	blobber1.Latitude = 7
	blobber1.Longitude = -31
	blobber1.WritePrice = 176
	blobber1.ReadPrice = 1111
	blobber1.TotalStake = 23
	blobber1.IsAvailable = true
	blobber1.LastHealthCheck = common.Timestamp(123)

	blobber2.ID = ids[1]
	blobber2.Latitude = -87
	blobber2.Longitude = 3
	blobber2.WritePrice = 17
	blobber2.ReadPrice = 1
	blobber2.TotalStake = 14783
	blobber2.IsAvailable = true
	blobber2.LastHealthCheck = common.Timestamp(3333333331)

	require.NoError(t, edb.updateBlobber([]Blobber{blobber1, blobber2}))

	b1, err := edb.GetBlobber(blobber1.ID)
	require.NoError(t, err)
	b2, err := edb.GetBlobber(blobber2.ID)
	require.NoError(t, err)
	compareBlobbers(t, blobber1, *b1)
	compareBlobbers(t, blobber2, *b2)

}

func compareBlobbers(t *testing.T, b1, b2 Blobber) {
	require.Equal(t, b1.ID, b2.ID)
	require.Equal(t, b1.Latitude, b2.Latitude)
	require.Equal(t, b1.Longitude, b2.Longitude)
	require.Equal(t, b1.WritePrice, b2.WritePrice)
	require.Equal(t, b1.ReadPrice, b2.ReadPrice)
	require.Equal(t, b1.TotalStake, b2.TotalStake)
	require.Equal(t, b1.IsAvailable, b2.IsAvailable)
	require.Equal(t, b1.LastHealthCheck, b2.LastHealthCheck)
}

func setUpBlobbers(t *testing.T, eventDb *EventDb, number int) []string {
	var ids []string
	var blobbers []Blobber
	for i := 0; i < number; i++ {
		blobber := Blobber{
			Provider: Provider{ID: fmt.Sprintf("somethingNew_%v", i)},
		}
		blobber.BaseURL = blobber.ID + ".com"
		ids = append(ids, blobber.ID)
		blobbers = append(blobbers, blobber)
	}
	require.NoError(t, eventDb.addBlobbers(blobbers))
	return ids
}
