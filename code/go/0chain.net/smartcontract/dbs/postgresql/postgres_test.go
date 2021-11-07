package postgresql

import (
	"testing"
	"time"

	"0chain.net/smartcontract/dbs/event"

	"github.com/stretchr/testify/require"

	"0chain.net/smartcontract/dbs"
)

func TestSetupDatabase(t *testing.T) {
	t.Skip()
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
	err := SetupDatabase(access)
	require.NoError(t, err)

	err = event.MigrateEventDb()
	require.NoError(t, err)

	events := []event.Event{
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

	event.AddEvents(events)

	oldEvents, err := event.GetEvents(0)
	require.NoError(t, err)
	require.Len(t, oldEvents, len(events))
}
