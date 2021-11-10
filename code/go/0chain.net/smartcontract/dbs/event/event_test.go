package event

import (
	"testing"
	"time"

	"0chain.net/core/logging"
	"go.uber.org/zap"

	"github.com/stretchr/testify/require"

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

	err = eventDb.drop()
	require.NoError(t, err)

	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	events := []Event{
		{
			BlockNumber: 1,
			Data:        "one",
		},
		{
			BlockNumber: 2,
			Data:        "one",
		},
		{
			BlockNumber: 2,
			Data:        "one",
		},
		{
			BlockNumber: 3,
			Data:        "one",
		},
		{
			BlockNumber: 4,
			Data:        "one",
		},
	}

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

}
