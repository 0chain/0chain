package event

import (
	"fmt"
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
	require.Len(t, mergedEvents, 1)
	err = edb.addStat(mergedEvents[0])
	require.NoError(t, err)

	var rps []ReadPool
	result := edb.Get().Find(&rps)
	fmt.Println("rps", rps)
	result = result

	rp1, err := edb.GetReadPool("user1")
	require.NoError(t, err)
	require.EqualValues(t, rp1.Balance, 5)
	rp2, err := edb.GetReadPool("user2")
	require.EqualValues(t, rp2.Balance, 11)

	updateReadPoolEvent := []Event{
		{
			BlockNumber: 5,
			TxHash:      "tx four",
			Type:        TypeStats,
			Tag:         TagUpdateReadpool,
			Index:       "user1",
			Data: ReadPool{
				UserID:  "user1",
				Balance: 17,
			},
		},
		{
			BlockNumber: 5,
			TxHash:      "tx five",
			Type:        TypeStats,
			Tag:         TagUpdateReadpool,
			Index:       "user1",
			Data: ReadPool{
				UserID:  "user1",
				Balance: 19,
			},
		},
		{
			BlockNumber: 5,
			TxHash:      "tx six",
			Type:        TypeStats,
			Tag:         TagUpdateReadpool,
			Index:       "user2",
			Data: ReadPool{
				UserID:  "user2",
				Balance: 23,
			},
		},
	}

	mergedEvents, err = mergeEvents(3, "three", updateReadPoolEvent)
	require.NoError(t, err, "merging readpoool inserts")
	require.Len(t, mergedEvents, 1)
	err = edb.addStat(mergedEvents[0])
	require.NoError(t, err)

	rp3, err := edb.GetReadPool("user1")
	require.NoError(t, err)
	require.EqualValues(t, rp3.Balance, 19)
	rp4, err := edb.GetReadPool("user2")
	require.EqualValues(t, rp4.Balance, 23)
}
