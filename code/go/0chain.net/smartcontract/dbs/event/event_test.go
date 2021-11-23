package event

import (
	"testing"
	"time"

	"0chain.net/core/logging"
	"0chain.net/smartcontract/dbs"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestSetupDatabase(t *testing.T) {
	//t.Skip("only for local debugging, requires local postgresql")
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

	events := []Event{
		{
			BlockNumber: 1,
			TxHash:      "a",
			Type:        "some type",
			Tag:         "green",
			Data:        "one",
		},
		{
			BlockNumber: 2,
			TxHash:      "b",
			Type:        "Error",
			Data:        "two",
		},
		{
			BlockNumber: 2,
			TxHash:      "c",
			Type:        "Some type",
			Tag:         "blue",
			Data:        "three",
		},
		{
			BlockNumber: 3,
			TxHash:      "d",
			Type:        "some other type",
			Tag:         "yellow",
			Data:        "four",
		},
		{
			BlockNumber: 4,
			TxHash:      "f",
			Type:        "Error",
			Data:        "five",
		},
	}

	eventDb.AddEvents(events)
	eventDb.AddEvents(events)

	oldEvents, err := eventDb.GetEvents(0)
	require.NoError(t, err)
	require.Len(t, oldEvents, len(events))

	filter := Event{
		BlockNumber: 2,
	}
	filterEvents, err := eventDb.FindEvents(filter)
	require.NoError(t, err)
	require.Len(t, filterEvents, 2)

	filter = Event{
		TxHash: "d",
	}
	filterEvents, err = eventDb.FindEvents(filter)
	require.NoError(t, err)
	require.Len(t, filterEvents, 1)

	filter = Event{
		Type: "Error",
	}
	filterEvents, err = eventDb.FindEvents(filter)
	require.NoError(t, err)
	require.Len(t, filterEvents, 2)

	filter = Event{
		BlockNumber: 2,
		Type:        "Error",
	}
	filterEvents, err = eventDb.FindEvents(filter)
	require.NoError(t, err)
	require.Len(t, filterEvents, 1)
}

/*




































 */
