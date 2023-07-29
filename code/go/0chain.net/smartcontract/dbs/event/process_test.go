package event

import (
	"context"
	"os"
	"testing"
	"time"

	"0chain.net/chaincore/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

func TestAddEvents(t *testing.T) {
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
	t.Skip("only for local debugging, requires local postgresql")
	eventDb, err := NewEventDbWithoutWorker(access, config.DbSettings{})
	if err != nil {
		return
	}
	eventDb.AutoMigrate()
	defer eventDb.Drop()

	eventDb.ProcessEvents(context.Background(), []Event{
		{
			TxHash: "somehash",
			Type:   TypeError,
			Data:   "someData",
		},
	}, 100, "hash", 10, CommitNow())
	errObj := Error{}
	time.Sleep(100 * time.Millisecond)
	result := eventDb.Store.Get().Model(&Error{}).Where(&Error{TransactionID: "somehash", Error: "someData"}).Take(&errObj)
	if result.Error != nil {
		t.Errorf("error while trying to find errorObject %v got error %v", errObj, result.Error)
	}
}

func TestUpdateHistoricData(t *testing.T) {
	eventDb, clean := GetTestEventDB(t)
	defer clean()

	s := fillSnapshot(t, eventDb)

	blobbers := []Blobber{
		buildMockBlobber(t, "blobber1"),
		buildMockBlobber(t, "blobber2"),
		buildMockBlobber(t, "blobber3"),
		buildMockBlobber(t, "blobber4"),
		buildMockBlobber(t, "blobber5"),
		buildMockBlobber(t, "blobber6"),
	}
	err := eventDb.Store.Get().Omit(clause.Associations).Create(&blobbers).Error
	require.NoError(t, err)

	blobberSnapshots := []BlobberSnapshot{
		buildMockBlobberSnapshot(t, "blobber1"),
		buildMockBlobberSnapshot(t, "blobber2"),
		buildMockBlobberSnapshot(t, "blobber3"),
		buildMockBlobberSnapshot(t, "blobber4"),
		buildMockBlobberSnapshot(t, "blobber5"),
		buildMockBlobberSnapshot(t, "blobber6"),
	}
	err = eventDb.Store.Get().Create(&blobberSnapshots).Error

	miners := []Miner{
		buildMockMiner(t, OwnerId, "miner1"),
		buildMockMiner(t, OwnerId, "miner2"),
		buildMockMiner(t, OwnerId, "miner3"),
		buildMockMiner(t, OwnerId, "miner4"),
		buildMockMiner(t, OwnerId, "miner5"),
		buildMockMiner(t, OwnerId, "miner6"),
	}
	err = eventDb.Store.Get().Omit(clause.Associations).Create(&miners).Error
	require.NoError(t, err)

	minerSnapshots := []MinerSnapshot{
		buildMockMinerSnapshot(t, "miner1"),
		buildMockMinerSnapshot(t, "miner2"),
		buildMockMinerSnapshot(t, "miner3"),
		buildMockMinerSnapshot(t, "miner4"),
		buildMockMinerSnapshot(t, "miner5"),
		buildMockMinerSnapshot(t, "miner6"),
	}
	err = eventDb.Store.Get().Create(&minerSnapshots).Error
	require.NoError(t, err)

	sharders := []Sharder{
		buildMockSharder(t, OwnerId, "sharder1"),
		buildMockSharder(t, OwnerId, "sharder2"),
		buildMockSharder(t, OwnerId, "sharder3"),
		buildMockSharder(t, OwnerId, "sharder4"),
		buildMockSharder(t, OwnerId, "sharder5"),
		buildMockSharder(t, OwnerId, "sharder6"),
	}
	err = eventDb.Store.Get().Omit(clause.Associations).Create(&sharders).Error
	require.NoError(t, err)

	sharderSnapshots := []SharderSnapshot{
		buildMockSharderSnapshot(t, "sharder1"),
		buildMockSharderSnapshot(t, "sharder2"),
		buildMockSharderSnapshot(t, "sharder3"),
		buildMockSharderSnapshot(t, "sharder4"),
		buildMockSharderSnapshot(t, "sharder5"),
		buildMockSharderSnapshot(t, "sharder6"),
	}
	err = eventDb.Store.Get().Create(&sharderSnapshots).Error
	require.NoError(t, err)

	validators := []Validator{
		buildMockValidator(t, OwnerId, "validator1"),
		buildMockValidator(t, OwnerId, "validator2"),
		buildMockValidator(t, OwnerId, "validator3"),
		buildMockValidator(t, OwnerId, "validator4"),
		buildMockValidator(t, OwnerId, "validator5"),
		buildMockValidator(t, OwnerId, "validator6"),
	}
	err = eventDb.Store.Get().Omit(clause.Associations).Create(&validators).Error
	require.NoError(t, err)

	validatorSnapshots := []ValidatorSnapshot{
		buildMockValidatorSnapshot(t, "validator1"),
		buildMockValidatorSnapshot(t, "validator2"),
		buildMockValidatorSnapshot(t, "validator3"),
		buildMockValidatorSnapshot(t, "validator4"),
		buildMockValidatorSnapshot(t, "validator5"),
		buildMockValidatorSnapshot(t, "validator6"),
	}
	err = eventDb.Store.Get().Create(&validatorSnapshots).Error
	require.NoError(t, err)

	authorizers := []Authorizer{
		buildMockAuthorizer(t, OwnerId, "authorizer1"),
		buildMockAuthorizer(t, OwnerId, "authorizer2"),
		buildMockAuthorizer(t, OwnerId, "authorizer3"),
		buildMockAuthorizer(t, OwnerId, "authorizer4"),
		buildMockAuthorizer(t, OwnerId, "authorizer5"),
		buildMockAuthorizer(t, OwnerId, "authorizer6"),
	}
	err = eventDb.Store.Get().Omit(clause.Associations).Create(&authorizers).Error
	require.NoError(t, err)

	authorizerSnapshots := []AuthorizerSnapshot{
		buildMockAuthorizerSnapshot(t, "authorizer1"),
		buildMockAuthorizerSnapshot(t, "authorizer2"),
		buildMockAuthorizerSnapshot(t, "authorizer3"),
		buildMockAuthorizerSnapshot(t, "authorizer4"),
		buildMockAuthorizerSnapshot(t, "authorizer5"),
		buildMockAuthorizerSnapshot(t, "authorizer6"),
	}
	err = eventDb.Store.Get().Create(&authorizerSnapshots).Error
	require.NoError(t, err)

	events := []Event{
		// Events changing blobbers
		{
			Type: TypeStats,
			Tag:  TagUpdateBlobber,
			Data: []Blobber{
				{Provider: Provider{ID: "blobber1"}},
				{Provider: Provider{ID: "blobber2"}},
			},
		},
		// Events changing miners
		{
			Type: TypeStats,
			Tag:  TagUpdateMiner,
			Data: []Miner{
				{Provider: Provider{ID: "miner2"}},
				{Provider: Provider{ID: "miner3"}},
			},
		},
		// Events changing sharders
		{
			Type: TypeStats,
			Tag:  TagUpdateSharderTotalStake,
			Data: []Sharder{
				{Provider: Provider{ID: "sharder3"}},
				{Provider: Provider{ID: "sharder4"}},
			},
		},
		// Events changing validators
		{
			Type: TypeStats,
			Tag:  TagUpdateValidatorStakeTotal,
			Data: []Validator{
				{Provider: Provider{ID: "validator4"}},
				{Provider: Provider{ID: "validator5"}},
			},
		},
		// Events changing authorizers
		{
			Type: TypeStats,
			Tag:  TagUpdateAuthorizerTotalStake,
			Data: []Authorizer{
				{Provider: Provider{ID: "authorizer2"}},
				{Provider: Provider{ID: "authorizer4"}},
			},
		},
		// Events changing global snapshot values
		{
			Type: TypeStats,
			Tag:  TagLockReadPool,
			Data: []ReadPoolLock{
				{Amount: 100},
				{Amount: 100},
			},
		},
		{
			Type: TypeStats,
			Tag:  TagFromChallengePool,
			Data: ChallengePoolLock{
				Amount: 100,
			},
		},
		// Events that does nothing
		{
			Type: TypeStats,
			Tag:  TagAddChallenge,
			Data: []Challenge{
				{ChallengeID: "challenge1"},
			},
		},
	}

	sBefore := *s
	sBefore.Round = 50

	sAfter, err := eventDb.updateHistoricData(BlockEvents{
		events: events,
		round:  50,
	}, s)
	require.NoError(t, err)

	err = sBefore.ApplyDiffBlobber(&blobbers[0], &blobberSnapshots[0])
	require.NoError(t, err)
	err = sBefore.ApplyDiffBlobber(&blobbers[1], &blobberSnapshots[1])
	require.NoError(t, err)
	err = sBefore.ApplyDiffMiner(&miners[1], &minerSnapshots[1])
	require.NoError(t, err)
	err = sBefore.ApplyDiffMiner(&miners[2], &minerSnapshots[2])
	require.NoError(t, err)
	err = sBefore.ApplyDiffSharder(&sharders[2], &sharderSnapshots[2])
	require.NoError(t, err)
	err = sBefore.ApplyDiffSharder(&sharders[3], &sharderSnapshots[3])
	require.NoError(t, err)
	err = sBefore.ApplyDiffValidator(&validators[3], &validatorSnapshots[3])
	require.NoError(t, err)
	err = sBefore.ApplyDiffValidator(&validators[4], &validatorSnapshots[4])
	require.NoError(t, err)
	err = sBefore.ApplyDiffAuthorizer(&authorizers[1], &authorizerSnapshots[1])
	require.NoError(t, err)
	err = sBefore.ApplyDiffAuthorizer(&authorizers[3], &authorizerSnapshots[3])
	require.NoError(t, err)
	sBefore.ClientLocks += 200
	sBefore.TotalReadPoolLocked += 200
	sBefore.TotalChallengePools -= 100

	assert.Equal(t, sBefore, *sAfter)

	// Check provider aggregates
	var blobberAggregatesAfter []BlobberAggregate
	err = eventDb.Store.Get().Find(&blobberAggregatesAfter).Error
	require.NoError(t, err)

	ba1, ba2 := blobberAggregatesAfter[0], blobberAggregatesAfter[1]
	if ba1.BlobberID != "blobber1" {
		ba1, ba2 = ba2, ba1
	}

	assert.Equal(t, int64(50), ba1.Round)
	assert.Equal(t, blobbers[0].ID, ba1.BlobberID)
	assert.Equal(t, blobbers[0].ID, ba1.BlobberID)
	assert.Equal(t, blobbers[0].Capacity, ba1.Capacity)
	assert.Equal(t, blobbers[0].Allocated, ba1.Allocated)
	assert.Equal(t, blobbers[0].SavedData, ba1.SavedData)
	assert.Equal(t, blobbers[0].ReadData, ba1.ReadData)
	assert.Equal(t, blobbers[0].TotalStake, ba1.TotalStake)
	assert.Equal(t, blobbers[0].Rewards.TotalRewards, ba1.TotalRewards)
	assert.Equal(t, blobbers[0].OffersTotal, ba1.OffersTotal)
	assert.Equal(t, blobbers[0].OpenChallenges, ba1.OpenChallenges)
	assert.Equal(t, blobbers[0].TotalBlockRewards, ba1.TotalBlockRewards)
	assert.Equal(t, blobbers[0].TotalStorageIncome, ba1.TotalStorageIncome)
	assert.Equal(t, blobbers[0].TotalReadIncome, ba1.TotalReadIncome)
	assert.Equal(t, blobbers[0].TotalSlashedStake, ba1.TotalSlashedStake)
	assert.Equal(t, blobbers[0].Downtime, ba1.Downtime)
	assert.Equal(t, blobbers[0].ChallengesPassed, ba1.ChallengesPassed)
	assert.Equal(t, blobbers[0].ChallengesCompleted, ba1.ChallengesCompleted)
	if blobbers[0].ChallengesCompleted == 0 {
		assert.Equal(t, float64(0), ba1.RankMetric)
	} else {
		assert.Equal(t, float64(blobbers[0].ChallengesPassed)/float64(blobbers[0].ChallengesCompleted), ba1.RankMetric)
	}
	assert.Equal(t, int64(50), ba2.Round)
	assert.Equal(t, blobbers[1].ID, ba2.BlobberID)
	assert.Equal(t, blobbers[1].Capacity, ba2.Capacity)
	assert.Equal(t, blobbers[1].Allocated, ba2.Allocated)
	assert.Equal(t, blobbers[1].SavedData, ba2.SavedData)
	assert.Equal(t, blobbers[1].ReadData, ba2.ReadData)
	assert.Equal(t, blobbers[1].TotalStake, ba2.TotalStake)
	assert.Equal(t, blobbers[1].Rewards.TotalRewards, ba2.TotalRewards)
	assert.Equal(t, blobbers[1].OffersTotal, ba2.OffersTotal)
	assert.Equal(t, blobbers[1].OpenChallenges, ba2.OpenChallenges)
	assert.Equal(t, blobbers[1].TotalBlockRewards, ba2.TotalBlockRewards)
	assert.Equal(t, blobbers[1].TotalStorageIncome, ba2.TotalStorageIncome)
	assert.Equal(t, blobbers[1].TotalReadIncome, ba2.TotalReadIncome)
	assert.Equal(t, blobbers[1].TotalSlashedStake, ba2.TotalSlashedStake)
	assert.Equal(t, blobbers[1].Downtime, ba2.Downtime)
	assert.Equal(t, blobbers[1].ChallengesPassed, ba2.ChallengesPassed)
	assert.Equal(t, blobbers[1].ChallengesCompleted, ba2.ChallengesCompleted)
	if blobbers[1].ChallengesCompleted == 0 {
		assert.Equal(t, float64(0), ba2.RankMetric)
	} else {
		assert.Equal(t, float64(blobbers[1].ChallengesPassed)/float64(blobbers[1].ChallengesCompleted), ba2.RankMetric)
	}

	// Check miner aggregates
	var minerAggregatesAfter []MinerAggregate
	err = eventDb.Store.Get().Find(&minerAggregatesAfter).Error
	require.NoError(t, err)

	ma1, ma2 := minerAggregatesAfter[0], minerAggregatesAfter[1]
	if ma1.MinerID != "miner2" {
		ma1, ma2 = ma2, ma1
	}
	assert.Equal(t, int64(50), ma1.Round)
	assert.Equal(t, miners[1].ID, ma1.MinerID)
	assert.Equal(t, miners[1].TotalStake, ma1.TotalStake)
	assert.Equal(t, miners[1].ServiceCharge, ma1.ServiceCharge)
	assert.Equal(t, miners[1].Rewards.TotalRewards, ma1.TotalRewards)
	assert.Equal(t, miners[1].Fees, ma1.Fees)
	assert.Equal(t, int64(50), ma2.Round)
	assert.Equal(t, miners[2].ID, ma2.MinerID)
	assert.Equal(t, miners[2].TotalStake, ma2.TotalStake)
	assert.Equal(t, miners[2].ServiceCharge, ma2.ServiceCharge)
	assert.Equal(t, miners[2].Rewards.TotalRewards, ma2.TotalRewards)
	assert.Equal(t, miners[2].Fees, ma2.Fees)

	// Check sharder aggregates
	var sharderAggregatesAfter []SharderAggregate
	err = eventDb.Store.Get().Find(&sharderAggregatesAfter).Error
	require.NoError(t, err)

	sa1, sa2 := sharderAggregatesAfter[0], sharderAggregatesAfter[1]
	if sa1.SharderID != "sharder3" {
		sa1, sa2 = sa2, sa1
	}
	assert.Equal(t, int64(50), sa1.Round)
	assert.Equal(t, sharders[2].ID, sa1.SharderID)
	assert.Equal(t, sharders[2].TotalStake, sa1.TotalStake)
	assert.Equal(t, sharders[2].ServiceCharge, sa1.ServiceCharge)
	assert.Equal(t, sharders[2].Rewards.TotalRewards, sa1.TotalRewards)
	assert.Equal(t, sharders[2].Fees, sa1.Fees)
	assert.Equal(t, int64(50), sa2.Round)
	assert.Equal(t, sharders[3].ID, sa2.SharderID)
	assert.Equal(t, sharders[3].TotalStake, sa2.TotalStake)
	assert.Equal(t, sharders[3].ServiceCharge, sa2.ServiceCharge)
	assert.Equal(t, sharders[3].Rewards.TotalRewards, sa2.TotalRewards)
	assert.Equal(t, sharders[3].Fees, sa2.Fees)

	// Check validator aggregates
	var validatorAggregatesAfter []ValidatorAggregate
	err = eventDb.Store.Get().Find(&validatorAggregatesAfter).Error
	require.NoError(t, err)

	va1, va2 := validatorAggregatesAfter[0], validatorAggregatesAfter[1]
	if va1.ValidatorID != "validator4" {
		va1, va2 = va2, va1
	}

	assert.Equal(t, int64(50), va1.Round)
	assert.Equal(t, validators[3].ID, va1.ValidatorID)
	assert.Equal(t, validators[3].TotalStake, va1.TotalStake)
	assert.Equal(t, validators[3].ServiceCharge, va1.ServiceCharge)
	assert.Equal(t, validators[3].Rewards.TotalRewards, va1.TotalRewards)
	assert.Equal(t, int64(50), va2.Round)
	assert.Equal(t, validators[4].ID, va2.ValidatorID)
	assert.Equal(t, validators[4].TotalStake, va2.TotalStake)
	assert.Equal(t, validators[4].ServiceCharge, va2.ServiceCharge)
	assert.Equal(t, validators[4].Rewards.TotalRewards, va2.TotalRewards)

	// Check authorizer aggregates
	var authorizerAggregatesAfter []AuthorizerAggregate
	err = eventDb.Store.Get().Find(&authorizerAggregatesAfter).Error
	require.NoError(t, err)

	aa1, aa2 := authorizerAggregatesAfter[0], authorizerAggregatesAfter[1]
	if aa1.AuthorizerID != "authorizer2" {
		aa1, aa2 = aa2, aa1
	}

	assert.Equal(t, int64(50), aa1.Round)
	assert.Equal(t, authorizers[1].ID, aa1.AuthorizerID)
	assert.Equal(t, authorizers[1].TotalStake, aa1.TotalStake)
	assert.Equal(t, authorizers[1].ServiceCharge, aa1.ServiceCharge)
	assert.Equal(t, authorizers[1].Rewards.TotalRewards, aa1.TotalRewards)
	assert.Equal(t, int64(50), aa2.Round)
	assert.Equal(t, authorizers[3].ID, aa2.AuthorizerID)
	assert.Equal(t, authorizers[3].TotalStake, aa2.TotalStake)
	assert.Equal(t, authorizers[3].ServiceCharge, aa2.ServiceCharge)
	assert.Equal(t, authorizers[3].Rewards.TotalRewards, aa2.TotalRewards)

	// Check blobber snapshots
	var blobberSnapshotsAfter []BlobberSnapshot
	err = eventDb.Store.Get().Find(&blobberSnapshotsAfter).Error
	require.NoError(t, err)

	assert.Len(t, blobberSnapshotsAfter, 6)
	unchangedSnapshots := map[int]string{2: "blobber3", 3: "blobber4", 4: "blobber5", 5: "blobber6"}
	for i, bid := range unchangedSnapshots {
		var blobberSnapshotFromDb BlobberSnapshot
		err := eventDb.Store.Get().Where("blobber_id = ?", bid).First(&blobberSnapshotFromDb).Error
		require.NoError(t, err)
		assert.Equal(t, blobberSnapshots[i], blobberSnapshotFromDb)
	}
	changedSnapshots := map[int]string{0: "blobber1", 1: "blobber2"}
	for i, bid := range changedSnapshots {
		var blobberSnapshotFromDb BlobberSnapshot
		err := eventDb.Store.Get().Where("blobber_id = ?", bid).First(&blobberSnapshotFromDb).Error
		require.NoError(t, err)
		snapExpected := createBlobberSnapshotFromBlobber(&blobbers[i], 50)
		assert.Equal(t, *snapExpected, blobberSnapshotFromDb)
	}

	// Check miner snapshots
	var minerSnapshotsAfter []MinerSnapshot
	err = eventDb.Store.Get().Find(&minerSnapshotsAfter).Error
	require.NoError(t, err)

	assert.Len(t, minerSnapshotsAfter, 6)
	unchangedSnapshots = map[int]string{0: "miner1", 3: "miner4", 4: "miner5", 5: "miner6"}
	for i, mid := range unchangedSnapshots {
		var minerSnapshotFromDb MinerSnapshot
		err := eventDb.Store.Get().Where("miner_id = ?", mid).First(&minerSnapshotFromDb).Error
		require.NoError(t, err)
		assert.Equal(t, minerSnapshots[i], minerSnapshotFromDb)
	}
	changedSnapshots = map[int]string{1: "miner2", 2: "miner3"}
	for i, mid := range changedSnapshots {
		var minerSnapshotFromDb MinerSnapshot
		err := eventDb.Store.Get().Where("miner_id = ?", mid).First(&minerSnapshotFromDb).Error
		require.NoError(t, err)
		snapExpected := createMinerSnapshotFromMiner(&miners[i], 50)
		assert.Equal(t, *snapExpected, minerSnapshotFromDb)
	}

	// Check sharder snapshots
	var sharderSnapshotsAfter []SharderSnapshot
	err = eventDb.Store.Get().Find(&sharderSnapshotsAfter).Error
	require.NoError(t, err)

	assert.Len(t, sharderSnapshotsAfter, 6)
	unchangedSnapshots = map[int]string{0: "sharder1", 1: "sharder2", 4: "sharder5", 5: "sharder6"}
	for i, sid := range unchangedSnapshots {
		var sharderSnapshotFromDb SharderSnapshot
		err := eventDb.Store.Get().Where("sharder_id = ?", sid).First(&sharderSnapshotFromDb).Error
		require.NoError(t, err)
		assert.Equal(t, sharderSnapshots[i], sharderSnapshotFromDb)
	}
	changedSnapshots = map[int]string{2: "sharder3", 3: "sharder4"}
	for i, sid := range changedSnapshots {
		var sharderSnapshotFromDb SharderSnapshot
		err := eventDb.Store.Get().Where("sharder_id = ?", sid).First(&sharderSnapshotFromDb).Error
		require.NoError(t, err)
		snapExpected := createSharderSnapshotFromSharder(&sharders[i], 50)
		assert.Equal(t, *snapExpected, sharderSnapshotFromDb)
	}

	// Check validator snapshots
	var validatorSnapshotsAfter []ValidatorSnapshot
	err = eventDb.Store.Get().Find(&validatorSnapshotsAfter).Error
	require.NoError(t, err)

	assert.Len(t, validatorSnapshotsAfter, 6)
	unchangedSnapshots = map[int]string{0: "validator1", 1: "validator2", 2: "validator3", 5: "validator6"}
	for i, vid := range unchangedSnapshots {
		var validatorSnapshotFromDb ValidatorSnapshot
		err := eventDb.Store.Get().Where("validator_id = ?", vid).First(&validatorSnapshotFromDb).Error
		require.NoError(t, err)
		assert.Equal(t, validatorSnapshots[i], validatorSnapshotFromDb)
	}
	changedSnapshots = map[int]string{3: "validator4", 4: "validator5"}
	for i, vid := range changedSnapshots {
		var validatorSnapshotFromDb ValidatorSnapshot
		err := eventDb.Store.Get().Where("validator_id = ?", vid).First(&validatorSnapshotFromDb).Error
		require.NoError(t, err)
		snapExpected := createValidatorSnapshotFromValidator(&validators[i], 50)
		assert.Equal(t, *snapExpected, validatorSnapshotFromDb)
	}

	// Check authorizer snapshots
	var authorizerSnapshotsAfter []AuthorizerSnapshot
	err = eventDb.Store.Get().Find(&authorizerSnapshotsAfter).Error
	require.NoError(t, err)

	assert.Len(t, authorizerSnapshotsAfter, 6)
	unchangedSnapshots = map[int]string{0: "authorizer1", 2: "authorizer3", 4: "authorizer5", 5: "authorizer6"}
	for i, aid := range unchangedSnapshots {
		var authorizerSnapshotFromDb AuthorizerSnapshot
		err := eventDb.Store.Get().Where("authorizer_id = ?", aid).First(&authorizerSnapshotFromDb).Error
		require.NoError(t, err)
		assert.Equal(t, authorizerSnapshots[i], authorizerSnapshotFromDb)
	}
	changedSnapshots = map[int]string{1: "authorizer2", 3: "authorizer4"}
	for i, aid := range changedSnapshots {
		var authorizerSnapshotFromDb AuthorizerSnapshot
		err := eventDb.Store.Get().Where("authorizer_id = ?", aid).First(&authorizerSnapshotFromDb).Error
		require.NoError(t, err)
		snapExpected := createAuthorizerSnapshotFromAuthorizer(&authorizers[i], 50)
		assert.Equal(t, *snapExpected, authorizerSnapshotFromDb)
	}
}
