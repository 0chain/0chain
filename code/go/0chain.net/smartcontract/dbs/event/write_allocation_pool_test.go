package event

import (
	"os"
	"testing"
	"time"

	"0chain.net/smartcontract/dbs"
	"github.com/guregu/null"
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
	err = eventDb.addOrOverwriteWriteAllocationPool(WriteAllocationPool{
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
	err = eventDb.addOrOverwriteWriteAllocationPool(WriteAllocationPool{
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
	assert.NoError(t, err, "There should be no error")
	write := WriteAllocationPool{}
	eventDb.Get().Model(&WriteAllocationPool{}).Where(&WriteAllocationPool{PoolID: "allocationID"}).Scan(&write)
	assert.Equal(t, int64(40), write.Balance, "Update failed")
	want := WriteAllocationPool{
		PoolID:  "allocation",
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
	}
	err = eventDb.addOrOverwriteWriteAllocationPool(want)
	assert.NoError(t, err, "there should be an error")

	w, err := eventDb.GetWriteAllocationPoolWithFilterAndPagination(WriteAllocationPoolFilter{PoolID: null.StringFrom("allocation")}, 0, 0)
	assert.NoError(t, err, "should be no error")
	assert.Equal(t, int64(2), w[0].Blobbers[0].Balance, "should be equal")
}
