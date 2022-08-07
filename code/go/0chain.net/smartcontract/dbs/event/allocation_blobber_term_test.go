package event

import (
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/currency"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	common2 "0chain.net/smartcontract/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"strconv"
	"testing"
	"time"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestAllocationBlobberTerms(t *testing.T) {
	t.Skip("only for local debugging, requires local postgres")

	access := config.DbAccess{
		Enabled:         true,
		Name:            "events_db",
		User:            "zchain_user",
		Password:        "zchian",
		Host:            "localhost",
		Port:            "5432",
		MaxIdleConns:    100,
		MaxOpenConns:    200,
		ConnMaxLifetime: 20 * time.Second,
	}
	eventDb, err := NewEventDb(access)
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.Drop()
	require.NoError(t, err)
	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	terms := []AllocationBlobberTerm{
		{
			AllocationID:            encryption.Hash("mockAllocation_" + strconv.Itoa(0)),
			BlobberID:               encryption.Hash("mockBlobber_" + strconv.Itoa(0)),
			ReadPrice:               int64(currency.Coin(29)),
			WritePrice:              int64(currency.Coin(31)),
			MinLockDemand:           37.0,
			MaxOfferDuration:        39 * time.Minute,
			ChallengeCompletionTime: 41 * time.Minute,
		},
		{
			AllocationID:             encryption.Hash("mockAllocation_" + strconv.Itoa(0)),
			BlobberID:               encryption.Hash("mockBlobber_" + strconv.Itoa(1)),
			ReadPrice:               int64(currency.Coin(41)),
			WritePrice:              int64(currency.Coin(43)),
			MinLockDemand:           47.0,
			MaxOfferDuration:        49 * time.Minute,
			ChallengeCompletionTime: 51 * time.Minute,
		},
	}

	err = eventDb.addOrOverwriteAllocationBlobberTerms(terms)
	require.NoError(t, err, "Error while inserting Allocation's Blobber's AllocationBlobberTerm to event database")

	var term *AllocationBlobberTerm
	var res []AllocationBlobberTerm
	limit := common2.Pagination{
		Offset:       0,
		Limit:        20,
		IsDescending: true,
	}
	res, err = eventDb.GetAllocationBlobberTerms(terms[0].AllocationID, terms[0].BlobberID, limit)
	require.Equal(t, int64(1), len(res), "AllocationBlobberTerm not getting inserted")

	res, err = eventDb.GetAllocationBlobberTerms(terms[0].AllocationID, "", limit)
	require.Equal(t, int64(2), len(res), "AllocationBlobberTerm not getting inserted")

	terms[1].MinLockDemand = 70.0
	err = eventDb.addOrOverwriteAllocationBlobberTerms(terms)
	require.NoError(t, err, "Error while inserting Allocation's Blobber's AllocationBlobberTerm to event database")

	term, err = eventDb.GetAllocationBlobberTerm(terms[1].AllocationID, terms[1].BlobberID)
	require.Equal(t, terms[1].MinLockDemand, term.MinLockDemand, "Error while overriding AllocationBlobberTerm in event Database")

	err = eventDb.Drop()
	require.NoError(t, err)
}
