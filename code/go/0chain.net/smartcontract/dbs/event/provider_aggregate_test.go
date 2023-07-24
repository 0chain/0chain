package event

import (
	"testing"

	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)


func TestCreateNewProviderAggregates(t *testing.T) {
	edb, rb := GetTestEventDB(t)
	defer rb()

	blobbers := []Blobber{
		buildMockBlobber(t, "blobber1"),
		buildMockBlobber(t, "blobber2"),
		buildMockBlobber(t, "blobber3"),
		buildMockBlobber(t, "blobber4"),
		buildMockBlobber(t, "blobber5"),
		buildMockBlobber(t, "blobber6"),
	}
	err := edb.Store.Get().Omit(clause.Associations).Create(&blobbers).Error
	require.NoError(t, err)

	miners := []Miner{
		buildMockMiner(t, OwnerId, "miner1"),
		buildMockMiner(t, OwnerId, "miner2"),
		buildMockMiner(t, OwnerId, "miner3"),
		buildMockMiner(t, OwnerId, "miner4"),
		buildMockMiner(t, OwnerId, "miner5"),
		buildMockMiner(t, OwnerId, "miner6"),
	}
	err = edb.Store.Get().Omit(clause.Associations).Create(&miners).Error
	require.NoError(t, err)

	sharders := []Sharder{
		buildMockSharder(t, OwnerId, "sharder1"),
		buildMockSharder(t, OwnerId, "sharder2"),
		buildMockSharder(t, OwnerId, "sharder3"),
		buildMockSharder(t, OwnerId, "sharder4"),
		buildMockSharder(t, OwnerId, "sharder5"),
		buildMockSharder(t, OwnerId, "sharder6"),
	}
	err = edb.Store.Get().Omit(clause.Associations).Create(&sharders).Error
	require.NoError(t, err)

	validators := []Validator{
		buildMockValidator(t, OwnerId, "validator1"),
		buildMockValidator(t, OwnerId, "validator2"),
		buildMockValidator(t, OwnerId, "validator3"),
		buildMockValidator(t, OwnerId, "validator4"),
		buildMockValidator(t, OwnerId, "validator5"),
		buildMockValidator(t, OwnerId, "validator6"),
	}
	err = edb.Store.Get().Omit(clause.Associations).Create(&validators).Error
	require.NoError(t, err)

	authorizers := []Authorizer{
		buildMockAuthorizer(t, OwnerId, "authorizer1"),
		buildMockAuthorizer(t, OwnerId, "authorizer2"),
		buildMockAuthorizer(t, OwnerId, "authorizer3"),
		buildMockAuthorizer(t, OwnerId, "authorizer4"),
		buildMockAuthorizer(t, OwnerId, "authorizer5"),
		buildMockAuthorizer(t, OwnerId, "authorizer6"),
	}
	err = edb.Store.Get().Omit(clause.Associations).Create(&authorizers).Error
	require.NoError(t, err)


	providers := ProvidersMap{
		spenum.Blobber: {
			"blobber1": &blobbers[0],
			"blobber2": &blobbers[1],
		},
		spenum.Miner: {
			"miner1": &miners[0],
			"miner2": &miners[1],
		},
		spenum.Sharder: {
			"sharder1": &sharders[0],
			"sharder2": &sharders[1],
		},
		spenum.Validator: {
			"validator1": &validators[0],
			"validator2": &validators[1],
		},
		spenum.Authorizer: {
			"authorizer1": &authorizers[0],
			"authorizer2": &authorizers[1],
		},
	}

	err = edb.CreateNewProviderAggregates(providers, 50)
	require.NoError(t, err)

	var blobberAggregates []*BlobberAggregate
	err = edb.Store.Get().Find(&blobberAggregates).Error
	require.NoError(t, err)
	assert.Len(t, blobberAggregates, 2)
	b1, b2 := blobberAggregates[0], blobberAggregates[1]
	if b1.BlobberID == "blobber2" {
		b1, b2 = b2, b1
	}
	assert.Equal(t, int64(50), b1.Round)
	assert.Equal(t, blobbers[0].ID, b1.BlobberID)
	assert.Equal(t, blobbers[0].Capacity, b1.Capacity)
	assert.Equal(t, blobbers[0].Allocated, b1.Allocated)
	assert.Equal(t, blobbers[0].SavedData, b1.SavedData)
	assert.Equal(t, blobbers[0].ReadData, b1.ReadData)
	assert.Equal(t, blobbers[0].TotalStake, b1.TotalStake)
	assert.Equal(t, blobbers[0].Rewards.TotalRewards, b1.TotalRewards)
	assert.Equal(t, blobbers[0].OffersTotal, b1.OffersTotal)
	assert.Equal(t, blobbers[0].OpenChallenges, b1.OpenChallenges)
	assert.Equal(t, blobbers[0].TotalBlockRewards, b1.TotalBlockRewards)
	assert.Equal(t, blobbers[0].TotalStorageIncome, b1.TotalStorageIncome)
	assert.Equal(t, blobbers[0].TotalReadIncome, b1.TotalReadIncome)
	assert.Equal(t, blobbers[0].TotalSlashedStake, b1.TotalSlashedStake)
	assert.Equal(t, blobbers[0].Downtime, b1.Downtime)
	assert.Equal(t, blobbers[0].ChallengesPassed, b1.ChallengesPassed)
	assert.Equal(t, blobbers[0].ChallengesCompleted, b1.ChallengesCompleted)
	if blobbers[0].ChallengesCompleted == 0 {
		assert.Equal(t, float64(0), b1.RankMetric)
	} else {
		assert.Equal(t, float64(blobbers[0].ChallengesPassed)/float64(blobbers[0].ChallengesCompleted), b1.RankMetric)
	}
	assert.Equal(t, int64(50), b2.Round)
	assert.Equal(t, blobbers[1].ID, b2.BlobberID)
	assert.Equal(t, blobbers[1].Capacity, b2.Capacity)
	assert.Equal(t, blobbers[1].Allocated, b2.Allocated)
	assert.Equal(t, blobbers[1].SavedData, b2.SavedData)
	assert.Equal(t, blobbers[1].ReadData, b2.ReadData)
	assert.Equal(t, blobbers[1].TotalStake, b2.TotalStake)
	assert.Equal(t, blobbers[1].Rewards.TotalRewards, b2.TotalRewards)
	assert.Equal(t, blobbers[1].OffersTotal, b2.OffersTotal)
	assert.Equal(t, blobbers[1].OpenChallenges, b2.OpenChallenges)
	assert.Equal(t, blobbers[1].TotalBlockRewards, b2.TotalBlockRewards)
	assert.Equal(t, blobbers[1].TotalStorageIncome, b2.TotalStorageIncome)
	assert.Equal(t, blobbers[1].TotalReadIncome, b2.TotalReadIncome)
	assert.Equal(t, blobbers[1].TotalSlashedStake, b2.TotalSlashedStake)
	assert.Equal(t, blobbers[1].Downtime, b2.Downtime)
	assert.Equal(t, blobbers[1].ChallengesPassed, b2.ChallengesPassed)
	assert.Equal(t, blobbers[1].ChallengesCompleted, b2.ChallengesCompleted)
	if blobbers[1].ChallengesCompleted == 0 {
		assert.Equal(t, float64(0), b2.RankMetric)
	} else {
		assert.Equal(t, float64(blobbers[1].ChallengesPassed)/float64(blobbers[1].ChallengesCompleted), b2.RankMetric)
	}

	var minerAggregates []*MinerAggregate
	err = edb.Store.Get().Find(&minerAggregates).Error
	require.NoError(t, err)
	assert.Len(t, minerAggregates, 2)
	m1, m2 := minerAggregates[0], minerAggregates[1]
	if m1.MinerID == "miner2" {
		m1, m2 = m2, m1
	}
	assert.Equal(t, int64(50), m1.Round)
	assert.Equal(t, miners[0].ID, m1.MinerID)
	assert.Equal(t, miners[0].TotalStake, m1.TotalStake)
	assert.Equal(t, miners[0].ServiceCharge, m1.ServiceCharge)
	assert.Equal(t, miners[0].Rewards.TotalRewards, m1.TotalRewards)
	assert.Equal(t, miners[0].Fees, m1.Fees)
	assert.Equal(t, int64(50), m2.Round)
	assert.Equal(t, miners[1].ID, m2.MinerID)
	assert.Equal(t, miners[1].TotalStake, m2.TotalStake)
	assert.Equal(t, miners[1].ServiceCharge, m2.ServiceCharge)
	assert.Equal(t, miners[1].Rewards.TotalRewards, m2.TotalRewards)
	assert.Equal(t, miners[1].Fees, m2.Fees)

	var sharderAggregates []*SharderAggregate
	err = edb.Store.Get().Find(&sharderAggregates).Error
	require.NoError(t, err)
	assert.Len(t, sharderAggregates, 2)
	s1, s2 := sharderAggregates[0], sharderAggregates[1]
	if s1.SharderID == "sharder2" {
		s1, s2 = s2, s1
	}
	assert.Equal(t, int64(50), s1.Round)
	assert.Equal(t, sharders[0].ID, s1.SharderID)
	assert.Equal(t, sharders[0].TotalStake, s1.TotalStake)
	assert.Equal(t, sharders[0].ServiceCharge, s1.ServiceCharge)
	assert.Equal(t, sharders[0].Rewards.TotalRewards, s1.TotalRewards)
	assert.Equal(t, sharders[0].Fees, s1.Fees)
	assert.Equal(t, int64(50), s2.Round)
	assert.Equal(t, sharders[1].ID, s2.SharderID)
	assert.Equal(t, sharders[1].TotalStake, s2.TotalStake)
	assert.Equal(t, sharders[1].ServiceCharge, s2.ServiceCharge)
	assert.Equal(t, sharders[1].Rewards.TotalRewards, s2.TotalRewards)
	assert.Equal(t, sharders[1].Fees, s2.Fees)

	var validatorAggregates []*ValidatorAggregate
	err = edb.Store.Get().Find(&validatorAggregates).Error
	require.NoError(t, err)
	assert.Len(t, validatorAggregates, 2)
	v1, v2 := validatorAggregates[0], validatorAggregates[1]
	if v1.ValidatorID == "validator2" {
		v1, v2 = v2, v1
	}
	assert.Equal(t, int64(50), v1.Round)
	assert.Equal(t, validators[0].ID, v1.ValidatorID)
	assert.Equal(t, validators[0].TotalStake, v1.TotalStake)
	assert.Equal(t, validators[0].ServiceCharge, v1.ServiceCharge)
	assert.Equal(t, validators[0].Rewards.TotalRewards, v1.TotalRewards)
	assert.Equal(t, int64(50), v2.Round)
	assert.Equal(t, validators[1].ID, v2.ValidatorID)
	assert.Equal(t, validators[1].TotalStake, v2.TotalStake)
	assert.Equal(t, validators[1].ServiceCharge, v2.ServiceCharge)
	assert.Equal(t, validators[1].Rewards.TotalRewards, v2.TotalRewards)

	var authorizerAggregates []*AuthorizerAggregate
	err = edb.Store.Get().Find(&authorizerAggregates).Error
	require.NoError(t, err)
	assert.Len(t, authorizerAggregates, 2)
	a1, a2 := authorizerAggregates[0], authorizerAggregates[1]
	if a1.AuthorizerID == "authorizer2" {
		a1, a2 = a2, a1
	}
	assert.Equal(t, int64(50), a1.Round)
	assert.Equal(t, authorizers[0].ID, a1.AuthorizerID)
	assert.Equal(t, authorizers[0].TotalStake, a1.TotalStake)
	assert.Equal(t, authorizers[0].ServiceCharge, a1.ServiceCharge)
	assert.Equal(t, authorizers[0].Rewards.TotalRewards, a1.TotalRewards)
	assert.Equal(t, int64(50), a2.Round)
	assert.Equal(t, authorizers[1].ID, a2.AuthorizerID)
	assert.Equal(t, authorizers[1].TotalStake, a2.TotalStake)
	assert.Equal(t, authorizers[1].ServiceCharge, a2.ServiceCharge)
	assert.Equal(t, authorizers[1].Rewards.TotalRewards, a2.TotalRewards)
}