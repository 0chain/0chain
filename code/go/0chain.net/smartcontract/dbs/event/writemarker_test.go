package event

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/logging"
	"0chain.net/smartcontract/dbs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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
		User:            os.Getenv("POSTGRES_USER"),
		Password:        os.Getenv("POSTGRES_PASSWORD"),
		Host:            os.Getenv("POSTGRES_HOST"),
		Port:            os.Getenv("POSTGRES_PORT"),
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
	data, err := json.Marshal(&eWriteMarker)
	require.NoError(t, err)

	eventAddOrOverwriteWm := Event{
		BlockNumber: eWriteMarker.BlockNumber,
		TxHash:      eWriteMarker.TransactionID,
		Type:        int(TypeStats),
		Tag:         int(TagAddOrOverwriteWriteMarker),
		Data:        string(data),
	}
	events := []Event{eventAddOrOverwriteWm}
	eventDb.AddEvents(context.TODO(), events)

	wm, err := eventDb.GetWriteMarker(eWriteMarker.TransactionID)
	require.NoError(t, err)
	require.EqualValues(t, wm.BlockNumber, eWriteMarker.BlockNumber)

	wms, err := eventDb.GetWriteMarkersForAllocationID(eWriteMarker.AllocationID)
	require.NoError(t, err)
	require.EqualValues(t, 1, len(*wms))
	require.EqualValues(t, eWriteMarker.BlockNumber, (*wms)[0].BlockNumber)
}

func TestLatestWriteMarker(t *testing.T) {
	access := dbs.DbAccess{
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
	eventDb, err := NewEventDb(access)
	if err != nil {
		t.Skip("only for local debugging, requires local postgresql")
	}

	defer eventDb.Close()
	err = eventDb.AutoMigrate()
	if err != nil {
		t.Errorf("Error while migrating")
		return
	}

	defer eventDb.drop()
	_, err = eventDb.GetLatestWriteMarker()
	if !assert.Error(t, err, "Empty Readmarker should return an erro") {
		return
	}
	err = eventDb.addOrOverwriteBlobber(Blobber{BlobberID: "someHash"})
	if !assert.NoError(t, err, "Error while writing blobber marker") {
		return
	}
	err = eventDb.addTransaction(Transaction{Hash: "something"})
	if !assert.NoError(t, err, "Error while writing blobber marker") {
		return
	}

	want := WriteMarker{TransactionID: "something", BlobberID: "someHash"}
	err = eventDb.addOrOverwriteWriteMarker(WriteMarker{TransactionID: "something", BlobberID: "someHash"})
	if !assert.NoError(t, err, "Error while writing read marker") {
		return
	}

	err = eventDb.addOrOverwriteWriteMarker(want)
	if !assert.NoError(t, err, "Error while writing read marker") {
		return
	}

	got, err := eventDb.GetLatestWriteMarker()
	if err != nil {
		t.Errorf("Read marker should not return error %v", err)
		return
	}
	got.CreatedAt = want.CreatedAt
	got.UpdatedAt = want.UpdatedAt
	got.ID = want.ID
	assert.Equal(t, want, got, "Latest write transaction should be returned")
}
