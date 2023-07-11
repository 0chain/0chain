package event

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadPool(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	insertReadPoolEvents := []Event{
		{
			BlockNumber: 3,
			TxHash:      "tx one",
			Type:        TypeStats,
			Tag:         TagInsertReadpool,
			Index:       "user1",
			Data: ReadPool{
				UserID:  "user1",
				Balance: 5,
			},
		},
		{
			BlockNumber: 3,
			TxHash:      "tx two",
			Type:        TypeStats,
			Tag:         TagInsertReadpool,
			Index:       "user2",
			Data: ReadPool{
				UserID:  "user2",
				Balance: 7,
			},
		},
		{
			BlockNumber: 3,
			TxHash:      "tx three",
			Type:        TypeStats,
			Tag:         TagInsertReadpool,
			Index:       "user2",
			Data: ReadPool{
				UserID:  "user2",
				Balance: 11,
			},
		},
	}
	mergedEvents, err := mergeEvents(3, "three", insertReadPoolEvents)
	require.NoError(t, err, "merging readpoool inserts")
	for _, event := range mergedEvents {
		require.NoError(t, edb.addStat(event))
	}
	rp1, err := edb.GetReadPool("user1")
	require.NoError(t, err)
	require.EqualValues(t, rp1.Balance, 5)
	rp2, err := edb.GetReadPool("uers2")
	require.EqualValues(t, rp2.Balance, 11)
}
