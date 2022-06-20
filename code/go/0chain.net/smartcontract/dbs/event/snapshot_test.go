package event

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddOrUpdateMint(t *testing.T) {
	eventDb := SetupDatabase(t)
	defer eventDb.Close()
	err := eventDb.AutoMigrate()
	defer eventDb.drop()
	require.NoError(t, err)

	err = eventDb.addOrUpdateTotalMint(Mint{
		BlockHash: "test2",
		Round:     100,
		Amount:    40,
	})
	require.NoError(t, err)
	MintTotalAmount, err := eventDb.GetRoundsMintTotal(100, 100)
	require.NoError(t, err)
	require.Equal(t, int64(40), MintTotalAmount, "Total amount not correct")

	eventDb.addOrUpdateTotalMint(Mint{
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
	defer eventDb.drop()
	if err != nil {
		t.Errorf("Cannot migrate database")
		return
	}
	count := 10
	AddSnapshots(t, eventDb, count)
	total, err := eventDb.GetRoundsMintTotal(2, 8)
	require.NoError(t, err)
	require.Equal(t, int64(35), total, "Total is not correct")
}

func AddSnapshots(t *testing.T, eventdb *EventDb, count int) {
	for i := 1; i <= count; i++ {
		hash := fmt.Sprintf("blockHash_%v", i)
		if err := eventdb.addSnapshot(&Snapshot{
			Round:           int64(i),
			BlockHash:       hash,
			MintTotalAmount: int64(i),
		}); err != nil {
			t.Error(err)
			return
		}
	}
}
