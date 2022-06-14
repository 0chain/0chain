package event

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"0chain.net/chaincore/config"
	"0chain.net/core/logging"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestReadMarkersPaginated(t *testing.T) {
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
	eventDb, err := NewEventDb(access)
	if err != nil {
		t.Skip("only for local debugging, requires local postgresql")
		return
	}
	defer eventDb.Close()
	err = eventDb.AutoMigrate()
	if err != nil {
		t.Error("Error migrating database")
		return
	}
	defer eventDb.Drop()
	insertMultipleReadMarker(t, eventDb)
	t.Run("get all readmarker with allocationID", func(t *testing.T) {
		rms, err := eventDb.GetReadMarkersFromQueryPaginated(ReadMarker{AllocationID: "1"}, -1, -1, false)
		assert.NoError(t, err)
		for _, rm := range rms {
			assert.Equal(t, "1", rm.AllocationID, "Allocation ID was not correct")
		}
		assert.Equal(t, 10, len(rms), "Not all allocation ID was sent")
	})
	t.Run("get all readmarker with allocationID", func(t *testing.T) {
		rms, err := eventDb.GetReadMarkersFromQueryPaginated(ReadMarker{AllocationID: "1"}, -1, -1, false)
		assert.NoError(t, err)
		for _, rm := range rms {
			assert.Equal(t, "1", rm.AllocationID, "Allocation ID was not correct")
		}
		assert.Equal(t, 10, len(rms), "Not all allocation ID was sent")
	})
	t.Run("get all readmarker with authticket", func(t *testing.T) {
		rms, err := eventDb.GetReadMarkersFromQueryPaginated(ReadMarker{AuthTicket: "1"}, -1, -1, false)
		assert.NoError(t, err)
		for _, rm := range rms {
			assert.Equal(t, "1", rm.AuthTicket, "AuthTicket was not correct")
		}
		assert.Equal(t, 10, len(rms), "Not all AuthTicket was sent")
	})

	t.Run("get all readmarker with pagination limit", func(t *testing.T) {
		rms, err := eventDb.GetReadMarkersFromQueryPaginated(ReadMarker{}, 0, 10, false)
		assert.NoError(t, err)
		for i, rm := range rms {
			transactionHash := fmt.Sprintf("transactionHash %v 0", i)
			want := ReadMarker{TransactionID: transactionHash, BlobberID: "blobberID 0", ClientID: "someClientID", AllocationID: strconv.Itoa(i), AuthTicket: strconv.Itoa(i), BlockNumber: int64(i), ReadSize: float64(i)}
			want.ID = rm.ID
			want.CreatedAt = rm.CreatedAt
			want.UpdatedAt = rm.UpdatedAt
			assert.Equal(t, want, rm, "RM was not correct")
		}
		assert.Equal(t, 10, len(rms), "Not all readmarker are sent correctly")
	})

	t.Run("get all readmarker with pagination offset and limit", func(t *testing.T) {
		rms, err := eventDb.GetReadMarkersFromQueryPaginated(ReadMarker{}, 5, 5, false)
		assert.NoError(t, err)
		for i, rm := range rms {
			transactionHash := fmt.Sprintf("transactionHash %v 0", i+5)
			want := ReadMarker{TransactionID: transactionHash, BlobberID: "blobberID 0", ClientID: "someClientID", AllocationID: strconv.Itoa(i + 5), AuthTicket: strconv.Itoa(i + 5), BlockNumber: int64(i + 5), ReadSize: float64(i + 5)}
			want.ID = rm.ID
			want.CreatedAt = rm.CreatedAt
			want.UpdatedAt = rm.UpdatedAt
			assert.Equal(t, want, rm, "RM was not correct")
		}
		assert.Equal(t, 5, len(rms), "Not all readmarker are sent correctly")
	})

	t.Run("get all readmarker with pagination descending", func(t *testing.T) {
		rms, err := eventDb.GetReadMarkersFromQueryPaginated(ReadMarker{}, 0, 10, true)
		assert.NoError(t, err)
		for i, rm := range rms {
			transactionHash := fmt.Sprintf("transactionHash %v 9", 9-i)
			want := ReadMarker{TransactionID: transactionHash, BlobberID: "blobberID 9", ClientID: "someClientID", AllocationID: strconv.Itoa(9 - i), AuthTicket: strconv.Itoa(9 - i), BlockNumber: int64(9 - i), ReadSize: float64(9 - i)}
			want.ID = rm.ID
			want.CreatedAt = rm.CreatedAt
			want.UpdatedAt = rm.UpdatedAt
			assert.Equal(t, want, rm, "RM was not correct")
		}
		assert.Equal(t, 10, len(rms), "Not all readmarker are sent correctly")
	})

	t.Run("ReadMarkers size total", func(t *testing.T) {
		gotWM, err := eventDb.GetDataReadFromAllocationForLastNBlocks(5, "")
		assert.NoError(t, err)
		assert.Equal(t, int64(300), gotWM)
	})

}

func insertMultipleReadMarker(t *testing.T, eventDb *EventDb) {
	for j := 0; j < 10; j++ {
		blobberID := fmt.Sprintf("blobberID %v", j)
		err := eventDb.addOrOverwriteBlobber(Blobber{BlobberID: blobberID})
		if !assert.NoError(t, err, "Error while writing blobber marker") {
			return
		}
		for i := 0; i < 10; i++ {
			transactionHash := fmt.Sprintf("transactionHash %v %v", i, j)
			err = eventDb.addTransaction(Transaction{Hash: transactionHash})
			if !assert.NoError(t, err, "Error while writing blobber marker") {
				return
			}
			err = eventDb.addOrOverwriteReadMarker(ReadMarker{TransactionID: transactionHash, BlobberID: blobberID, ClientID: "someClientID", AllocationID: strconv.Itoa(i), AuthTicket: strconv.Itoa(i), BlockNumber: int64(i), ReadSize: float64(i)})
			if !assert.NoError(t, err, "Error while writing read marker") {
				return
			}
		}
	}
}
