package event

import (
	"os"
	"testing"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/common"
	"github.com/stretchr/testify/require"
)

var (
	ChallengePoolID = "id_challenge_pool"
	AllocationID    = "id_allocation"
)

func TestChallengePoolEvent(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")
	access := config.DbAccess{
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

	eventDb, err := NewEventDb(access, config.DbSettings{})
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	c := ChallengePool{
		ID:           ChallengePoolID,
		AllocationID: AllocationID,
		Balance:      0,
		StartTime:    0,
		Expiration:   0,
		Finalized:    false,
	}

	err = eventDb.addOrUpdateChallengePools([]ChallengePool{c})
	require.NoError(t, err, "Error while inserting ChallengePool to event Database")

	var count int64
	eventDb.Get().Model(&ChallengePool{}).Count(&count)
	require.Equal(t, int64(1), count, "ChallengePool not getting inserted")

	c.Balance = 11223344
	err = eventDb.addOrUpdateChallengePools([]ChallengePool{c})
	require.NoError(t, err, "Error while updating ChallengePool to event Database")

	eventDb.Get().Model(&ChallengePool{}).Count(&count)
	require.Equal(t, int64(1), count, "ChallengePool not getting updated")

	cp, err := eventDb.GetChallengePool(AllocationID, common.Pagination{0, 20, true})
	require.NoError(t, err, "Error while getting challengePools for allocation ID")
	require.Equal(t, int64(11223344), cp.Balance, "ChallengePool balance is not getting updated")

	c.ID = c.ID + "_2"
	err = eventDb.addOrUpdateChallengePools([]ChallengePool{c})
	require.NoError(t, err, "Error while inserting ChallengePool to event Database")

	eventDb.Get().Model(&ChallengePool{}).Count(&count)
	require.Equal(t, int64(2), count, "Second ChallengePool is not getting inserted")

	eventDb.Get().Model(&ChallengePool{}).Where("allocation_id = ?", c.ID).Delete(&ChallengePool{})
	eventDb.Get().Model(&ChallengePool{}).Count(&count)
	require.Equal(t, int64(1), count, "ChallengePool not getting deleted")
	eventDb.Get().Model(&ChallengePool{}).Where("allocation_id = ?", AllocationID).Delete(&ChallengePool{})
	require.Equal(t, int64(0), count, "ChallengePool not getting deleted")

	err = eventDb.Drop()
	require.NoError(t, err)
}
