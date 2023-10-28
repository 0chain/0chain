package event

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventDb_userSnapshotFunctions(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	testSnapshots := []*UserSnapshot{
		{
			UserID:          "user1",
			Round:           1,
			CollectedReward: 1,
			PayedFees:       2,
			TotalStake:      3,
			ReadPoolTotal:   4,
			WritePoolTotal:  5,
		},
		{
			UserID:          "user2",
			Round:           10,
			CollectedReward: 10,
			PayedFees:       20,
			TotalStake:      30,
			ReadPoolTotal:   40,
			WritePoolTotal:  50,
		},
		{
			UserID:          "user3",
			Round:           100,
			CollectedReward: 100,
			PayedFees:       200,
			TotalStake:      300,
			ReadPoolTotal:   400,
			WritePoolTotal:  500,
		},
	}
	err := edb.Store.Get().Create(testSnapshots).Error
	require.NoError(t, err)

	t.Run("GetUserSnapshotsByIds", func(t *testing.T) {
		snapshots, err := edb.GetUserSnapshotsByIds([]string{"user1", "user2", "user3"})
		require.NoError(t, err)
		require.Len(t, snapshots, 3)

		uniqueSnaps := make(map[string]bool, 3)
		for _, snap := range snapshots {
			uniqueSnaps[snap.UserID] = true
			var expectedSnapshot *UserSnapshot
			switch snap.UserID {
			case "user1":
				expectedSnapshot = testSnapshots[0]
			case "user2":
				expectedSnapshot = testSnapshots[1]
			case "user3":
				expectedSnapshot = testSnapshots[2]
			}
			assert.Equal(t, expectedSnapshot.UserID, snap.UserID)
			assert.Equal(t, expectedSnapshot.CollectedReward, snap.CollectedReward)
			assert.Equal(t, expectedSnapshot.PayedFees, snap.PayedFees)
			assert.Equal(t, expectedSnapshot.TotalStake, snap.TotalStake)
			assert.Equal(t, expectedSnapshot.ReadPoolTotal, snap.ReadPoolTotal)
			assert.Equal(t, expectedSnapshot.WritePoolTotal, snap.WritePoolTotal)
		}
		assert.Len(t, uniqueSnaps, 3)
	})

	t.Run("AddOrOverwriteUserSnapshots", func(t *testing.T) {
		testSnapshots[1].Round *= 2
		testSnapshots[1].CollectedReward *= 2
		testSnapshots[1].PayedFees *= 2
		testSnapshots[1].TotalStake *= 2
		testSnapshots[1].ReadPoolTotal *= 2
		testSnapshots[1].WritePoolTotal *= 2

		testSnapshots[2].Round *= 2
		testSnapshots[2].CollectedReward /= 2
		testSnapshots[2].PayedFees /= 2
		testSnapshots[2].TotalStake /= 2
		testSnapshots[2].ReadPoolTotal /= 2
		testSnapshots[2].WritePoolTotal /= 2

		testSnapshots = append(testSnapshots, &UserSnapshot{
			UserID:          "user4",
			Round:           1000,
			CollectedReward: 1000,
			PayedFees:       2000,
			TotalStake:      3000,
			ReadPoolTotal:   4000,
			WritePoolTotal:  5000,
		})

		err := edb.AddOrOverwriteUserSnapshots(testSnapshots)
		require.NoError(t, err)

		snapshots, err := edb.GetUserSnapshotsByIds([]string{"user1", "user2", "user3", "user4"})
		require.NoError(t, err)
		require.Len(t, snapshots, 4)

		uniqueSnaps := make(map[string]bool, 4)
		for _, snap := range snapshots {
			uniqueSnaps[snap.UserID] = true
			var expectedSnapshot *UserSnapshot
			switch snap.UserID {
			case "user1":
				expectedSnapshot = testSnapshots[0]
			case "user2":
				expectedSnapshot = testSnapshots[1]
			case "user3":
				expectedSnapshot = testSnapshots[2]
			case "user4":
				expectedSnapshot = testSnapshots[3]
			}
			assert.Equal(t, expectedSnapshot.UserID, snap.UserID)
			assert.Equal(t, expectedSnapshot.CollectedReward, snap.CollectedReward)
			assert.Equal(t, expectedSnapshot.PayedFees, snap.PayedFees)
			assert.Equal(t, expectedSnapshot.TotalStake, snap.TotalStake)
			assert.Equal(t, expectedSnapshot.ReadPoolTotal, snap.ReadPoolTotal)
			assert.Equal(t, expectedSnapshot.WritePoolTotal, snap.WritePoolTotal)
			assert.WithinDuration(t, time.Now(), snap.UpdatedAt, 2*time.Second)
		}
		assert.Len(t, uniqueSnaps, 4)
	})
}
