package event

import (
	"os"
	"strconv"
	"testing"
	"time"

	"0chain.net/smartcontract/dbs"
	"github.com/guregu/null"
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
	err = eventDb.addReadAllocationPool(ReadAllocationPool{
		AllocationID:  "allocationID",
		TransactionId: "transaction id",
		UserID:        "some user id",
		Balance:       23,
		Blobbers: []BlobberPool{
			{
				ReadAllocationPoolID: null.StringFrom("allocationID"),
				Balance:              2,
				BlobberID:            "blobberID",
			},
			{
				ReadAllocationPoolID: null.StringFrom("allocationID"),
				Balance:              2,
				BlobberID:            "blobberID1",
			},
		},
	})
	assert.NoError(t, err, "There should be on error")
	err = eventDb.addReadAllocationPool(ReadAllocationPool{
		AllocationID:  "allocation",
		TransactionId: "transaction id",
		UserID:        "some user id",
		Balance:       23,
		Blobbers: []BlobberPool{
			{
				ReadAllocationPoolID: null.StringFrom("allocation1"),
				Balance:              2,
				BlobberID:            "blobberID",
			},
			{
				ReadAllocationPoolID: null.StringFrom("allocation2"),
				Balance:              2,
				BlobberID:            "blobberID1",
			},
		},
	})
	assert.Error(t, err, "there should be an error")
}

func TestReadAllocationPoolFilter(t *testing.T) {
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
	createReadAllocationPool(t, eventDb, 20)
	t.Run("return only read allocation with userid", func(t *testing.T) {
		allocations, err := eventDb.GetReadAllocationPoolWithFilterAndPagination(ReadAllocationPoolFilter{
			UserID: null.StringFrom("userid"),
		}, 0, 10)
		assert.NoError(t, err, "There should be no error")
		assert.Equal(t, 10, len(allocations), "not all read allocations were returned")
		for _, allocation := range allocations {
			assert.Equal(t, "userid", allocation.UserID, "write pool should not be returned")
		}
	})
}

func createReadAllocationPool(t *testing.T, eventDb *EventDb, count int) {
	for i := 0; i < count; i++ {
		indexString := strconv.Itoa(i)
		err := eventDb.addReadAllocationPool(
			ReadAllocationPool{
				AllocationID:  "allocation" + indexString,
				TransactionId: "transaction" + indexString,
				UserID:        "userid",
				Balance:       int64(i),
				Blobbers: []BlobberPool{
					{
						ReadAllocationPoolID: null.StringFrom("allocation" + indexString),
						BlobberID:            "blobber1",
						Balance:              2,
					},
					{
						ReadAllocationPoolID: null.StringFrom("allocation" + indexString),
						BlobberID:            "blobber2",
						Balance:              2,
					},
				},
			},
		)
		assert.NoError(t, err, "error while creating allocation")
	}
}
