package event

import (
	"os"
	"testing"
	"time"

	"0chain.net/smartcontract/dbs"
	"github.com/stretchr/testify/assert"
)

func TestReadAllocationPool(t *testing.T) {
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
	err = eventDb.addOrOverwriteReadAllocationPool(ReadAllocationPool{
		PoolID:  "allocationID",
		UserID:  "some user id",
		Balance: 23,
		Blobbers: []BlobberPool{
			{
				Balance:   2,
				BlobberID: "blobberID",
			},
			{
				Balance:   2,
				BlobberID: "blobberID1",
			},
		},
	})
	assert.NoError(t, err, "There should be on error")
	err = eventDb.addOrOverwriteReadAllocationPool(ReadAllocationPool{
		PoolID:  "allocationID",
		UserID:  "some user id",
		Balance: 40,
		Blobbers: []BlobberPool{
			{
				Balance:   2,
				BlobberID: "blobberID",
			},
			{
				Balance:   2,
				BlobberID: "blobberID1",
			},
		},
	})
	assert.NoError(t, err, "There should be on error")
	read := ReadAllocationPool{}
	eventDb.Get().Model(&ReadAllocationPool{}).Where(&ReadAllocationPool{PoolID: "allocationID"}).Scan(&read)
	assert.Equal(t, int64(40), read.Balance, "Update failed")

	err = eventDb.addOrOverwriteReadAllocationPool(ReadAllocationPool{
		AllocationID: "allocation",
		UserID:       "some user id",
		Balance:      23,
		Blobbers: []BlobberPool{
			{
				Balance:   2,
				BlobberID: "blobberID",
			},
			{
				Balance:   2,
				BlobberID: "blobberID1",
			},
		},
	})
	assert.NoError(t, err, "there should be an error")
}
