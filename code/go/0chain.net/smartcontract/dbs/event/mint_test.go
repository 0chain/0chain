package event

import (
	"os"
	"testing"
	"time"

	"0chain.net/chaincore/config"
	"github.com/stretchr/testify/require"
)

func TestAddMint(t *testing.T) {
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
	eventDb, err := NewEventDb(access)
	if err != nil {
		t.Skip("only for local debugging, requires local postgresql")
		return
	}
	defer eventDb.Close()
	err = eventDb.AutoMigrate()
	defer eventDb.Drop()
	require.NoError(t, err)
	m := &Mint{
		BlockHash: "some_hash",
		Round:     100,
		Amount:    39,
	}
	eventDb.addOrUpdateTotalMint(m)
	var gotMint *Mint

	require.NoError(t, eventDb.Get().Model(&Mint{}).Where(&m).Scan(&gotMint).Error, "Mint was not found")
}
