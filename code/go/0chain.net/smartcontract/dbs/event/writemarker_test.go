package event

import (
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"0chain.net/smartcontract/dbs"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"testing"
	"time"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestWriteMarkers(t *testing.T) {
	t.Skip("only for local debugging, requires local postgres")

	type StorageWriteMarker struct {
		AllocationRoot         string           `json:"allocation_root"`
		PreviousAllocationRoot string           `json:"prev_allocation_root"`
		AllocationID           string           `json:"allocation_id"`
		Size                   int64            `json:"size"`
		BlobberID              string           `json:"blobber_id"`
		Timestamp              common.Timestamp `json:"timestamp"`
		ClientID               string           `json:"client_id"`
		Signature              string           `json:"signature"`
	}

	convertSwm := func(swm StorageWriteMarker, txnHash string, blockNumber int64) WriteMarker {
		return WriteMarker{
			ClientID:               swm.ClientID,
			BlobberID:              swm.BlobberID,
			AllocationID:           swm.AllocationID,
			TransactionID:          txnHash,
			AllocationRoot:         swm.AllocationRoot,
			PreviousAllocationRoot: swm.PreviousAllocationRoot,
			Size:                   swm.Size,
			Timestamp:              int64(swm.Timestamp),
			Signature:              swm.Signature,
			BlockNumber:            blockNumber,
		}
	}

	access := dbs.DbAccess{
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
	err = eventDb.drop()
	require.NoError(t, err)
	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	swm := StorageWriteMarker{
		AllocationRoot:         "allocation root",
		PreviousAllocationRoot: "previous allocation root",
		AllocationID:           "allocation if",
		Size:                   10,
		BlobberID:              "blobber id",
		Timestamp:              15015,
		ClientID:               "client id",
		Signature:              "signature",
	}

	eWriteMarker := convertSwm(swm, "t_hash", 1)
	_, err = json.Marshal(&eWriteMarker)
	require.NoError(t, err)
}
