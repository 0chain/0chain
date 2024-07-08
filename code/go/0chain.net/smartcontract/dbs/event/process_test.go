package event

import (
	"context"
	"os"
	"testing"
	"time"

	"0chain.net/core/config"
)

func TestAddEvents(t *testing.T) {
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
	eventDb, err := NewEventDbWithoutWorker(access, config.DbSettings{})
	if err != nil {
		return
	}
	eventDb.AutoMigrate()
	defer eventDb.Drop()

	eventDb.ProcessEvents(context.Background(), []Event{
		{
			TxHash: "somehash",
			Type:   TypeError,
			Data:   "someData",
		},
	}, 100, "hash", 10,
		func(be BlockEvents) error { return nil },
		CommitNow())
	errObj := Error{}
	time.Sleep(100 * time.Millisecond)
	result := eventDb.Store.Get().Model(&Error{}).Where(&Error{TransactionID: "somehash", Error: "someData"}).Take(&errObj)
	if result.Error != nil {
		t.Errorf("error while trying to find errorObject %v got error %v", errObj, result.Error)
	}
}
