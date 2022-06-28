package event

import (
	"fmt"
	"os"
	"testing"
	"time"

	"0chain.net/chaincore/config"
	"github.com/stretchr/testify/require"
)

func TestAddOrUpdateMint(t *testing.T) {
	eventDb := SetupDatabase(t)
	defer eventDb.Close()
	err := eventDb.AutoMigrate()
	defer eventDb.Drop()
	require.NoError(t, err)

	err = eventDb.addOrUpdateTotalMint(&Mint{
		BlockHash: "test2",
		Round:     100,
		Amount:    40,
	})
	require.NoError(t, err)
	MintTotalAmount, err := eventDb.GetRoundsMintTotal(100, 100)
	require.NoError(t, err)
	require.Equal(t, int64(40), MintTotalAmount, "Total amount not correct")

	eventDb.addOrUpdateTotalMint(&Mint{
		BlockHash: "test2",
		Round:     100,
		Amount:    60,
	})
	require.NoError(t, err)
	MintTotalAmount, err = eventDb.GetRoundsMintTotal(100, 100)
	require.NoError(t, err)
	require.Equal(t, int64(60), MintTotalAmount, "Total amount not correct")
}

func TestRoundMintSum(t *testing.T) {
	eventDb := SetupDatabase(t)
	defer eventDb.Close()
	err := eventDb.AutoMigrate()
	defer eventDb.Drop()
	if err != nil {
		t.Errorf("Cannot migrate database")
		return
	}
	count := 10
	AddMints(t, eventDb, count)
	total, err := eventDb.GetRoundsMintTotal(2, 8)
	require.NoError(t, err)
	require.Equal(t, int64(35), total, "Total is not correct")
}

func AddMints(t *testing.T, eventdb *EventDb, count int) {
	for i := 1; i <= count; i++ {
		hash := fmt.Sprintf("blockHash_%v", i)
		if err := eventdb.addOrUpdateTotalMint(&Mint{
			Round:     int64(i),
			BlockHash: hash,
			Amount:    int64(i),
		}); err != nil {
			t.Error(err)
			return
		}
	}
}

func SetupDatabase(t *testing.T) *EventDb {

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
	}
	return eventDb
}
