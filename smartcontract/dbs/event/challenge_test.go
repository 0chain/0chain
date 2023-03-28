package event

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChallengeEvent(t *testing.T) {

	t.Run("test addChallenge", func(t *testing.T) {
		eventDb, clear := GetTestEventDB(t)
		defer clear()

		c := Challenge{
			ChallengeID:    "challenge_id",
			CreatedAt:      0,
			AllocationID:   "allocation_id",
			BlobberID:      "blobber_id",
			ValidatorsID:   "validator_id1,validator_id2",
			Seed:           0,
			AllocationRoot: "allocation_root",
			Responded:      false,
		}
	
		err := eventDb.addChallenges([]Challenge{c})
		require.NoError(t, err, "Error while inserting Challenge to event Database")	

		challenge, err := eventDb.GetChallenge(c.ChallengeID)
		require.NoError(t, err)
		require.NotNil(t, challenge)
	})

	t.Run("test updateChallenges", func(t *testing.T) {
		eventDb, clear := GetTestEventDB(t)
		defer clear()
		cid1, cid2 := "challenge_id_1", "challenge_id_2"

	
		err := eventDb.addChallenges([]Challenge{
			{
				ChallengeID:    cid1,
				CreatedAt:      0,
				AllocationID:   "allocation_id",
				BlobberID:      "blobber_id",
				ValidatorsID:   "validator_id1,validator_id2",
				Seed:           0,
				AllocationRoot: "allocation_root",
				Responded:      false,
				Passed:         false,
			},
			{
				ChallengeID:    cid2,
				CreatedAt:      0,
				AllocationID:   "allocation_id",
				BlobberID:      "blobber_id",
				ValidatorsID:   "validator_id1,validator_id2",
				Seed:           0,
				AllocationRoot: "allocation_root",
				Responded:      false,
				Passed:         false,
			},
		})
		require.NoError(t, err, "Error while inserting Challenge to event Database")	
		
		err = eventDb.updateChallenges([]Challenge{
			{
				ChallengeID:    cid1,
				Responded:      true,
				Passed:         true,
			},
			{
				ChallengeID:    cid2,
				Responded:      true,
				Passed:         true,
			},
		})
		require.NoError(t, err, "Error while updating Challenge to event Database")
		
		challenge, err := eventDb.GetChallenge(cid1)
		require.NoError(t, err)
		require.True(t, challenge.Responded)
		require.True(t, challenge.Passed)

		challenge, err = eventDb.GetChallenge(cid2)
		require.NoError(t, err)
		require.True(t, challenge.Responded)
		require.True(t, challenge.Passed)
	})
}
