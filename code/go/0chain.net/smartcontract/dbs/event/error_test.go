package event

import (
	"os"
	"testing"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/common"
	"github.com/stretchr/testify/require"
)

func TestAddAndGetError(t *testing.T) {
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
	eventDb, err := NewEventDb(access)
	if err != nil {
		return
	}
	defer eventDb.Close()
	err = eventDb.AutoMigrate()
	defer eventDb.Drop()
	require.NoError(t, err)
	wantErr := Error{
		TransactionID: "someTransaction",
		Error:         "Some random error",
	}
	err = eventDb.addError(wantErr)
	if err != nil {
		t.Errorf("Error was not inserted in the table")
	}
	gotErr, err := eventDb.GetErrorByTransactionHash("someTransaction", common.Pagination{Limit: 20, IsDescending: true})
	require.Equal(t, 1, len(gotErr), "There should be 1 error")
	gotErr[0].ID = wantErr.ID
	gotErr[0].CreatedAt = wantErr.CreatedAt
	gotErr[0].UpdatedAt = wantErr.UpdatedAt
	require.Equal(t, []Error{wantErr}, gotErr, "The error should be equal")

	gotErr, err = eventDb.GetErrorByTransactionHash("someT", common.Pagination{Limit: 20, IsDescending: true})
	require.Equal(t, 0, len(gotErr), "We should get 0 errors")
}
