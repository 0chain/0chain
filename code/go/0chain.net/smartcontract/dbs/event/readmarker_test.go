package event

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"0chain.net/core/common"
	"0chain.net/core/logging"
	"0chain.net/smartcontract/dbs"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestReadMarkers(t *testing.T) {
	t.Skip("only for local debugging, requires local postgres")

	type ModelReadMarker struct {
		ClientID        string           `json:"client_id"`
		ClientPublicKey string           `json:"client_public_key"`
		BlobberID       string           `json:"blobber_id"`
		AllocationID    string           `json:"allocation_id"`
		OwnerID         string           `json:"owner_id"`
		Timestamp       common.Timestamp `json:"timestamp"`
		ReadCounter     int64            `json:"counter"`
		Signature       string           `json:"signature"`
		PayerID         string           `json:"payer_id"`
		AuthTicket      string           `json:"auth_ticket"`
		ReadSize        float64          `json:"read_size"`
	}

	convertMrm := func(mrm *ModelReadMarker, txnHash string, blockNumber int64) ReadMarker {
		return ReadMarker{
			Model:         gorm.Model{},
			ClientID:      mrm.ClientID,
			BlobberID:     mrm.BlobberID,
			AllocationID:  mrm.AllocationID,
			TransactionID: txnHash,
			OwnerID:       mrm.OwnerID,
			Timestamp:     int64(mrm.Timestamp),
			ReadCounter:   mrm.ReadCounter,
			ReadSize:      mrm.ReadSize,
			Signature:     mrm.Signature,
			PayerID:       mrm.PayerID,
			AuthTicket:    mrm.AuthTicket,
			BlockNumber:   blockNumber,
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

	mrm := ModelReadMarker{
		ClientID:        "client_id",
		ClientPublicKey: "client_public_key",
		BlobberID:       "blobber_id",
		AllocationID:    "allocation_id",
		OwnerID:         "owner_id",
		Timestamp:       111121,
		ReadCounter:     11,
		Signature:       "signature",
		PayerID:         "payer_id",
		AuthTicket:      "auth_ticket",
		ReadSize:        100,
	}

	eReadMarker := convertMrm(&mrm, "t_hash", 1)
	data, err := json.Marshal(&eReadMarker)
	require.NoError(t, err)

	eventAddOrOverwriteRm := Event{
		BlockNumber: eReadMarker.BlockNumber,
		TxHash:      eReadMarker.TransactionID,
		Type:        int(TypeStats),
		Tag:         int(TagAddOrOverwriteReadMarker),
		Data:        string(data),
	}
	events := []Event{eventAddOrOverwriteRm}
	eventDb.AddEvents(context.TODO(), events)

	query := &ReadMarker{TransactionID: eReadMarker.TransactionID}
	rms, err := eventDb.GetReadMarkersFromQuery(query)

	require.NoError(t, err)
	require.EqualValues(t, 1, len(*rms))
	require.EqualValues(t, eReadMarker.BlockNumber, (*rms)[0].BlockNumber)

	count, err := eventDb.CountReadMarkersFromQuery(query)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
}

func TestLastReadMarker(t *testing.T) {
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
	_, err = eventDb.GetLatestReadMarker()
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

	want := ReadMarker{TransactionID: "something", BlobberID: "someHash"}
	err = eventDb.addOrOverwriteReadMarker(ReadMarker{TransactionID: "something", BlobberID: "someHash"})
	if !assert.NoError(t, err, "Error while writing read marker") {
		return
	}

	err = eventDb.addOrOverwriteReadMarker(want)
	if !assert.NoError(t, err, "Error while writing read marker") {
		return
	}

	got, err := eventDb.GetLatestReadMarker()
	if err != nil {
		t.Errorf("Read marker should not return error %v", err)
		return
	}
	got.CreatedAt = want.CreatedAt
	got.UpdatedAt = want.UpdatedAt
	got.ID = want.ID
	assert.Equal(t, want, got, "Latest transaction should be returned")
}
