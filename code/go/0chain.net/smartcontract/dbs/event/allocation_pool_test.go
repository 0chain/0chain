package event

import (
	"os"
	"testing"
	"time"

	"0chain.net/smartcontract/dbs"
	"github.com/stretchr/testify/assert"
)

func TestWriteAllocationPool(t *testing.T) {
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
	defer eventDb.drop()
	assert.NoError(t, err, "error while migrating")
	err = eventDb.addAllocationPool(AllocationPool{
		AllocationID:  "allocationID",
		TransactionId: "transaction id",
		UserID:        "some user id",
		Balance:       23,
		Blobbers: []BlobberPool{
			{
				AllocationPoolID: "allocationID",
				Balance:          2,
				BlobberID:        "blobberID",
			},
			{
				AllocationPoolID: "allocationID",
				Balance:          2,
				BlobberID:        "blobberID1",
			},
		},
	})
	assert.NoError(t, err, "There should be on error")
	err = eventDb.addAllocationPool(AllocationPool{
		AllocationID:  "allocation",
		TransactionId: "transaction id",
		UserID:        "some user id",
		Balance:       23,
		Blobbers: []BlobberPool{
			{
				AllocationPoolID: "allocation1",
				Balance:          2,
				BlobberID:        "blobberID",
			},
			{
				AllocationPoolID: "allocation2",
				Balance:          2,
				BlobberID:        "blobberID1",
			},
		},
	})
	assert.Error(t, err, "there should be an error")
}
