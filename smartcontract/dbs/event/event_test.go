package event

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"0chain.net/chaincore/config"
	"github.com/0chain/common/core/logging"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestSetupDatabase(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")
	access := config.DbAccess{
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
	eventDb, err := NewEventDb(access, config.DbSettings{})
	require.NoError(t, err)
	defer eventDb.Close()

	err = eventDb.Drop()
	require.NoError(t, err)

	err = eventDb.AutoMigrate()
	require.NoError(t, err)
}
