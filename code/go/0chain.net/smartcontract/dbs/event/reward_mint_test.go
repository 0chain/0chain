package event

import (
	"os"
	"testing"
	"time"

	"0chain.net/chaincore/config"
	"github.com/stretchr/testify/require"
)

func TestRewardEvents(t *testing.T) {
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

	reward := RewardMint{
		Amount:       500,
		BlockNumber:  345,
		ClientID:     "new_wallet_id",
		PoolID:       "new_pool_id",
		ProviderType: "blobber",
		ProviderID:   "blobber_id",
	}

	err = eventDb.addRewardMint(reward)
	require.NoError(t, err, "Error while inserting reward data to event Database")

	var count int64
	eventDb.Get().Table("rewards").Count(&count)
	require.Equal(t, int64(1), count, "Rewards not getting inserted")

	reward.BlockNumber = 890
	reward.ClientID = "another_wallet_id"
	err = eventDb.addRewardMint(reward)
	require.NoError(t, err, "Error while inserting reward to event Database")

	eventDb.Get().Table("rewards").Count(&count)
	require.Equal(t, int64(2), count, "Rewards not getting inserted")

	rewardQuery := RewardMintQuery{
		StartBlock: 0,
		EndBlock:   900,
	}
	claimedReward, err := eventDb.GetRewardClaimedTotalBetweenBlocks(rewardQuery)
	require.NoError(t, err, "Error while getting sum of rewards")
	require.Equal(t, int64(1000), claimedReward, "Not all rewards were calculated")

	rewardQuery.ClientID = "new_wallet_id"
	claimedReward, err = eventDb.GetRewardClaimedTotalBetweenBlocks(rewardQuery)
	require.NoError(t, err, "Error while getting sum of rewards")
	require.Equal(t, int64(500), claimedReward, "Specific reward was not calculated")

	rewardQuery.StartBlock = 0
	rewardQuery.EndBlock = 350
	claimedReward, err = eventDb.GetRewardClaimedTotalBetweenBlocks(rewardQuery)
	require.NoError(t, err, "Error while getting sum of rewards")
	require.Equal(t, int64(500), claimedReward, "Specific reward was not calculated")

	rewardQuery.ClientID = ""
	rewardQuery.StartBlock = 350
	claimedReward, err = eventDb.GetRewardClaimedTotalBetweenBlocks(rewardQuery)
	require.NoError(t, err, "Error while getting sum of rewards")
	require.Equal(t, int64(0), claimedReward, "Specific reward was not calculated")

	rewardQuery = RewardMintQuery{
		ClientID: "another_wallet_id",
	}
	err = removeReward(eventDb, rewardQuery)
	require.NoError(t, err, "Error while removing reward from event Database")

	eventDb.Get().Table("rewards").Count(&count)
	require.Equal(t, int64(1), count, "Rewards not getting inserted")

	rewardQuery.ClientID = ""
	err = removeReward(eventDb, rewardQuery)
	require.NoError(t, err, "Error while removing reward from event Database")

	eventDb.Get().Table("curators").Count(&count)
	require.Equal(t, int64(0), count, "Curator not getting deleted")

	err = eventDb.Drop()
	require.NoError(t, err)
}

func TestRewardMintIssue1899(t *testing.T) {
	eventDb, clean := GetTestEventDB(t)
	defer clean()

	block := Block{
		Round:        1,
		CreationDate: time.Now().AddDate(0, 0, -5).UnixNano(),
		Hash:         "new_hash",
		ChainId:      "new_chain_id",
		MinerID:      "new_miner_id",
	}

	err := eventDb.addOrUpdateBlock(block)
	require.NoError(t, err, "Error while inserting block to event Database")

	var blocksCount int64
	eventDb.Get().Table("blocks").Count(&blocksCount)
	require.Equal(t, int64(1), blocksCount, "Blocks not getting inserted")

	block.Round = 800
	block.CreationDate = time.Now().AddDate(0, 0, -1).UnixNano()
	block.Hash = "another_hash"

	err = eventDb.addOrUpdateBlock(block)
	require.NoError(t, err, "Error while inserting block to event Database")

	eventDb.Get().Table("blocks").Count(&blocksCount)
	require.Equal(t, int64(2), blocksCount, "Blocks not getting inserted")

	reward := RewardMint{
		Amount:       500,
		BlockNumber:  345,
		ClientID:     "new_wallet_id",
		PoolID:       "new_pool_id",
		ProviderType: "blobber",
		ProviderID:   "blobber_id",
	}

	err = eventDb.addRewardMint(reward)
	require.NoError(t, err, "Error while inserting reward data to event Database")

	var count int64
	eventDb.Get().Table("reward_mints").Count(&count)
	require.Equal(t, int64(1), count, "Rewards not getting inserted")

	reward.BlockNumber = 800
	reward.ClientID = "another_wallet_id"
	err = eventDb.addRewardMint(reward)
	require.NoError(t, err, "Error while inserting reward to event Database")

	eventDb.Get().Table("reward_mints").Count(&count)
	require.Equal(t, int64(2), count, "Rewards not getting inserted")

	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -7)

	rewardQuery := RewardMintQuery{
		StartDate:  startTime,
		EndDate:    endTime,
		DataPoints: 1,
		ClientID:   "new_wallet_id",
	}

	rewardQuery.ClientID = "new_wallet_id"
	claimedReward, err := eventDb.GetRewardClaimedTotalBetweenDates(rewardQuery)
	require.NoError(t, err, "Error while getting sum of rewards")
	require.Len(t, claimedReward, 1)
	require.Equal(t, int64(500), claimedReward[0], "Specific reward was not calculated")
}

func removeReward(edb *EventDb, query RewardMintQuery) error {
	reward := RewardMint{
		ClientID:     query.ClientID,
		PoolID:       query.PoolID,
		ProviderType: query.ProviderType,
		ProviderID:   query.ProviderID,
	}
	q := edb.Store.Get().Model(&RewardMint{}).Where(&RewardMintQuery{ClientID: query.ClientID})

	if query.EndBlock > 0 {
		q = q.Where("block_number >= ? AND block_number <= ?", query.StartBlock, query.EndBlock)
	} else if query.StartBlock > 0 {
		q = q.Where("block_number >= ?", query.StartBlock)
	}

	return q.Delete(&reward).Error
}
