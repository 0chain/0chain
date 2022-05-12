package event

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"0chain.net/smartcontract/dbs"
)

func TestChallengeEvent(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")
	access := dbs.DbAccess{
		Enabled:         true,
		Name:            "events_db",
		User:            os.Getenv("POSTGRES_USER"),
		Password:        os.Getenv("POSTGRES_PASSWORD"),
		Host:            os.Getenv("POSTGRES_HOST"),
		Port:            os.Getenv("POSTGRES_PORT"),
		MaxIdleConns:    100,
		MaxOpenConns:    200,
		ConnMaxLifetime: 20 * time.Second,
	}
	eventDb, err := NewEventDb(access)
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.Drop()
	require.NoError(t, err)
	err = eventDb.AutoMigrate()
	require.NoError(t, err)

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

	err = eventDb.addChallenge(&c)
	require.NoError(t, err, "Error while inserting Challenge to event Database")

	var count int64
	eventDb.Get().Table("curators").Count(&count)
	require.Equal(t, int64(1), count, "Challenge not getting inserted")

	update := dbs.DbUpdates{
		Id:      c.ChallengeID,
		Updates: map[string]interface{}{"responded": true},
	}
	err = eventDb.updateChallenge(update)
	require.NoError(t, err, "Error while updating challenge to event Database")

	challenge, err := eventDb.GetChallenge(c.ChallengeID)
	require.NoError(t, err, "Error while listing challenge")
	require.EqualValues(t, challenge.Responded, true, "Challenge fetch failed")

	cs, err := eventDb.GetChallengeForBlobber(c.BlobberID, c.ChallengeID)
	require.NoError(t, err)
	require.Equal(t, c.BlobberID, cs.BlobberID)
}
