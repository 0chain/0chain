package event

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"0chain.net/core/logging"
	"0chain.net/smartcontract/dbs"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestSetupDatabase(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")
	access := dbs.DbAccess{
		Enabled:         true,
		Name:            "events_db",
		User:            "zchain_user",
		Password:        "zchian",
		Host:            "localhost",
		Port:            "5432",
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
}

func (edb *EventDb) drop() error {
	err := edb.Store.Get().Migrator().DropTable(&Event{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Blobber{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Transaction{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Error{})
	if err != nil {
		return err
	}
  
  err = edb.Store.Get().Migrator().DropTable(&WriteMarker{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Validator{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Block{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&ReadMarker{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Miner{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Curator{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().DropTable(&Sharder{})
	if err != nil {
		return err
	}

	return nil
}

func TestEventExists(t *testing.T) {
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
	if err != nil {
		t.Skip("only for local debugging, requires local postgresql")
	}
	defer eventDb.Close()

	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	eventDb.AddEvents(context.Background(), []Event{
		{
			BlockNumber: 1,
			TxHash:      "someHash",
			Type:        int(TypeStats),
			Tag:         0,
			Index:       "someIndex",
			Data:        "some random data",
		},
	})
	gotExists, err := eventDb.exists(context.Background(), Event{
		BlockNumber: 1,
		TxHash:      "someHash",
		Type:        int(TypeStats),
		Tag:         0,
		Index:       "someIndex",
		Data:        "some random data",
	})
	if !gotExists || err != nil {
		t.Errorf("Exists function did not work want true got %v and err was %v", gotExists, err)
	}
	gotExists, err = eventDb.exists(context.Background(), Event{
		BlockNumber: 1,
		TxHash:      "someHash",
		Type:        int(TypeStats),
		Tag:         0,
		Index:       "some1Index",
		Data:        "some random data",
	})
	if gotExists || err != nil {
		t.Errorf("Exists function did not work want false got %v and err was %v", gotExists, err)
	}

	err = eventDb.drop()
	require.NoError(t, err)
}
