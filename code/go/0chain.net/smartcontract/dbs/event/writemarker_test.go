package event

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	common2 "0chain.net/smartcontract/common"
	"github.com/0chain/common/core/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestWriteMarker(t *testing.T) {
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

	access := config.DbAccess{
		Enabled:         true,
		Name:            "events_db",
		User:            os.Getenv("POSTGRES_USER"),
		Password:        os.Getenv("POSTGRES_PASSWORD"),
		Host:            os.Getenv("POSTGRES_HOST"),
		Port:            os.Getenv("POSTGRES_PORT"),
		MaxIdleConns:    100,
		MaxOpenConns:    200,
		ConnMaxLifetime: 20 * time.Second,
	}
	eventDb, err := NewEventDb(access, config.DbSettings{})
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.Drop()
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
	data, err := json.Marshal(&eWriteMarker)
	require.NoError(t, err)

	eventAddOrOverwriteWm := Event{
		BlockNumber: eWriteMarker.BlockNumber,
		TxHash:      eWriteMarker.TransactionID,
		Type:        TypeStats,
		Tag:         TagAddWriteMarker,
		Data:        string(data),
	}
	events := []Event{eventAddOrOverwriteWm}
	eventDb.ProcessEvents(context.TODO(), events, 100, "hash", 10)

	wm, err := eventDb.GetWriteMarker(eWriteMarker.TransactionID)
	require.NoError(t, err)
	require.EqualValues(t, wm.BlockNumber, eWriteMarker.BlockNumber)

	wms, err := eventDb.GetWriteMarkersForAllocationID(eWriteMarker.AllocationID, common2.Pagination{Offset: 20})
	require.NoError(t, err)
	require.EqualValues(t, 1, len(wms))
	require.EqualValues(t, eWriteMarker.BlockNumber, (wms)[0].BlockNumber)
}

func TestGetWriteMarkers(t *testing.T) {
	access := config.DbAccess{
		Enabled:         true,
		Name:            os.Getenv("POSTGRES_DB"),
		User:            os.Getenv("POSTGRES_USER"),
		Password:        os.Getenv("POSTGRES_PASSWORD"),
		Host:            os.Getenv("POSTGRES_HOST"),
		Port:            os.Getenv("POSTGRES_PORT"),
		MaxIdleConns:    100,
		MaxOpenConns:    200,
		ConnMaxLifetime: 20 * time.Second,
	}
	t.Skip("only for local debugging, requires local postgresql")
	eventDb, err := NewEventDb(access, config.DbSettings{})
	if err != nil {
		return
	}

	defer eventDb.Close()
	err = eventDb.AutoMigrate()
	if err != nil {
		t.Errorf("Error while migrating")
		return
	}
	defer eventDb.Drop()
	err = eventDb.addOrOverwriteBlobber([]Blobber{{Provider: Provider{ID: "someHash"}}})
	if !assert.NoError(t, err, "Error while writing blobber marker") {
		return
	}
	err = eventDb.addTransactions([]Transaction{{Hash: "something"}})
	if !assert.NoError(t, err, "Error while writing blobber marker") {
		return
	}

	addWriterMarkers(t, eventDb, "someHash")

	t.Run("GetWriteMarkers ascending", func(t *testing.T) {
		gotWM, err := eventDb.GetWriteMarkers(common2.Pagination{Limit: 10})
		assert.NoError(t, err)
		compareWriteMarker(t, gotWM, "someHash", 0, 10, false)
	})
	t.Run("GetWriteMarkers descending", func(t *testing.T) {
		gotWM, err := eventDb.GetWriteMarkers(common2.Pagination{Limit: 10})
		assert.NoError(t, err)
		compareWriteMarker(t, gotWM, "someHash", 0, 10, true)
	})
	t.Run("GetWriteMarkers 5 limit asecending", func(t *testing.T) {
		gotWM, err := eventDb.GetWriteMarkers(common2.Pagination{Limit: 5})
		assert.NoError(t, err)
		compareWriteMarker(t, gotWM, "someHash", 0, 5, false)
	})
	t.Run("GetWriteMarkers 5 limit descending", func(t *testing.T) {
		gotWM, err := eventDb.GetWriteMarkers(common2.Pagination{Limit: 5})
		assert.NoError(t, err)
		compareWriteMarker(t, gotWM, "someHash", 0, 5, true)
	})
	t.Run("GetWriteMarkers 5 offset 5 limit asecending", func(t *testing.T) {
		gotWM, err := eventDb.GetWriteMarkers(common2.Pagination{Offset: 5, Limit: 10})
		assert.NoError(t, err)
		compareWriteMarker(t, gotWM, "someHash", 5, 5, false)
	})
	t.Run("GetWriteMarkers 5 offset 5 limit descending", func(t *testing.T) {
		gotWM, err := eventDb.GetWriteMarkers(common2.Pagination{Offset: 5, Limit: 10, IsDescending: true})
		assert.NoError(t, err)
		compareWriteMarker(t, gotWM, "someHash", 5, 5, true)
	})
	t.Run("GetWriteMarkersForAllocationID", func(t *testing.T) {
		gotWM, err := eventDb.GetWriteMarkersForAllocationID("allocation_id", common2.Pagination{Offset: 20})
		assert.NoError(t, err)
		compareWriteMarker(t, gotWM, "someHash", 5, 5, true)
	})
	t.Run("GetWriteMarkersForAllocationFile", func(t *testing.T) {
		gotWM, err := eventDb.GetWriteMarkersForAllocationFile("allocation_id", "name_txt", common2.Pagination{Offset: 20})
		assert.NoError(t, err)
		compareWriteMarker(t, gotWM, "someHash", 5, 5, true)
	})
	t.Run("WriteMarkers size total", func(t *testing.T) {
		gotWM, err := eventDb.GetAllocationWrittenSizeInLastNBlocks(5, "")
		assert.NoError(t, err)
		assert.Equal(t, int64(30), gotWM)
	})
	t.Run("writeMarker count", func(t *testing.T) {
		gotCount, err := eventDb.GetWriteMarkerCount("allocation_id")
		assert.NoError(t, err)
		assert.Equal(t, int64(10), gotCount, "count should be 10")
	})
}

func addWriterMarkers(t *testing.T, eventDb *EventDb, blobberID string) {
	for i := 0; i < 10; i++ {
		transactionID := fmt.Sprintf("transactionHash_%d", i)
		err := eventDb.addTransactions([]Transaction{{Hash: transactionID}})
		if !assert.NoError(t, err, "Error while writing blobber marker") {
			return
		}
		wm := WriteMarker{TransactionID: transactionID, BlobberID: blobberID, BlockNumber: int64(i), Size: int64(i), AllocationID: "allocation_id", Name: "name.txt"}
		err = eventDb.addWriteMarkers([]WriteMarker{wm})
		if !assert.NoError(t, err, "Error while writing read marker") {
			return
		}
	}
}

func compareWriteMarker(t *testing.T, gotWM []WriteMarker, blobberID string, offset, limit int, isDescending bool) {
	if isDescending {
		t.Log(offset, limit, offset+limit-1)
		for j, i := 0, 9-offset; j < limit; i, j = i-1, j+1 {
			transactionID := fmt.Sprintf("transactionHash_%d", i)
			want := WriteMarker{TransactionID: transactionID, BlobberID: blobberID, BlockNumber: int64(i), Size: int64(i), AllocationID: "allocation_id"}
			want.ID = gotWM[j].ID
			want.CreatedAt = gotWM[j].CreatedAt
			want.UpdatedAt = gotWM[j].UpdatedAt
			assert.Equal(t, want, gotWM[j], "Got invalid WM")
		}
		return
	}
	for i, j := offset, 0; i < offset+limit; i, j = i+1, j+1 {
		transactionID := fmt.Sprintf("transactionHash_%d", i)
		want := WriteMarker{TransactionID: transactionID, BlobberID: blobberID, BlockNumber: int64(i), Size: int64(i), AllocationID: "allocation_id"}
		want.ID = gotWM[j].ID
		want.CreatedAt = gotWM[j].CreatedAt
		want.UpdatedAt = gotWM[j].UpdatedAt
		assert.Equal(t, want, gotWM[j], "Got invalid WM")
	}
}
