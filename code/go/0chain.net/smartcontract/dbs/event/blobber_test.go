package event

import (
	"fmt"
	"testing"
	"time"

	"0chain.net/core/common"
	common2 "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"

	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"github.com/go-faker/faker/v4"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testBlobberSavedData = 1000

func init() {
	logging.Logger = zap.NewNop()
}

func TestUpdateBlobber(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	ids := setUpBlobbers(t, edb, 10, false)
	var blobber1, blobber2 Blobber
	blobber1.ID = ids[0]
	blobber1.WritePrice = 176
	blobber1.ReadPrice = 1111
	blobber1.TotalStake = 23
	blobber1.NotAvailable = false
	blobber1.LastHealthCheck = common.Timestamp(123)

	blobber2.ID = ids[1]
	blobber2.WritePrice = 17
	blobber2.ReadPrice = 1
	blobber2.TotalStake = 14783
	blobber2.NotAvailable = false
	blobber2.LastHealthCheck = common.Timestamp(3333333331)

	require.NoError(t, edb.updateBlobber([]Blobber{blobber1, blobber2}))

	b1, err := edb.GetBlobber(blobber1.ID)
	require.NoError(t, err)
	b2, err := edb.GetBlobber(blobber2.ID)
	require.NoError(t, err)
	compareBlobbers(t, blobber1, *b1)
	compareBlobbers(t, blobber2, *b2)

}

func TestUpdateBlobberStats(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	ids := setUpBlobbers(t, edb, 10, true)
	var blobber1, blobber2 Blobber
	blobber1.ID = ids[0]
	blobber1.SavedData = -100

	blobber2.ID = ids[1]
	blobber2.SavedData = 200 // increase the savedData by 200 units

	require.NoError(t, edb.updateBlobbersStats([]Blobber{blobber1, blobber2}))

	b1, err := edb.GetBlobber(blobber1.ID)
	require.NoError(t, err)
	require.Equal(t, int64(testBlobberSavedData-100), b1.SavedData)

	b2, err := edb.GetBlobber(blobber2.ID)
	require.NoError(t, err)
	require.Equal(t, int64(testBlobberSavedData+200), b2.SavedData)
}

func TestEventDb_blobberSpecificRevenue(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	err := edb.Store.Get().Model(&Blobber{}).Omit(clause.Associations).Create([]Blobber{
		{
			Provider: Provider{
				ID: "B000",
			},
			BaseURL:            "https://blobber.zero",
			TotalBlockRewards:  0,
			TotalStorageIncome: 0,
			TotalReadIncome:    0,
			TotalSlashedStake:  0,
		},
		{
			Provider: Provider{
				ID: "B001",
			},
			BaseURL:            "https://blobber.one",
			TotalBlockRewards:  0,
			TotalStorageIncome: 0,
			TotalReadIncome:    0,
			TotalSlashedStake:  0,
		},
		{
			Provider: Provider{
				ID: "B002",
			},
			BaseURL:            "https://blobber.two",
			TotalBlockRewards:  0,
			TotalStorageIncome: 0,
			TotalReadIncome:    0,
			TotalSlashedStake:  0,
		},
		{
			Provider: Provider{
				ID: "B003",
			},
			BaseURL:            "https://blobber.three",
			TotalBlockRewards:  0,
			TotalStorageIncome: 0,
			TotalReadIncome:    0,
			TotalSlashedStake:  0,
		},
	}).Error
	require.NoError(t, err)

	spus := []dbs.StakePoolReward{
		{
			// Shouldn't affect anybody
			ProviderID: dbs.ProviderID{
				ID:   "M000",
				Type: spenum.Miner,
			},
			Reward:     10,
			RewardType: spenum.BlockRewardMiner,
		},
		{
			// Block Reward: blobber zero
			ProviderID: dbs.ProviderID{
				ID:   "B000",
				Type: spenum.Blobber,
			},
			Reward:     10,
			RewardType: spenum.BlockRewardBlobber,
		},
		{
			// Storage income : blobber one
			ProviderID: dbs.ProviderID{
				ID:   "B001",
				Type: spenum.Blobber,
			},
			Reward:     20,
			RewardType: spenum.ChallengePassReward,
		},
		{
			// Read income : blobber two
			ProviderID: dbs.ProviderID{
				ID:   "B002",
				Type: spenum.Blobber,
			},
			Reward:     30,
			RewardType: spenum.FileDownloadReward,
		},
		{
			// Slashed stake : blobber three slashed stake should increase by 60
			ProviderID: dbs.ProviderID{
				ID:   "B003",
				Type: spenum.Blobber,
			},
			Reward:     40,
			RewardType: spenum.ChallengeSlashPenalty,
			DelegatePenalties: map[string]currency.Coin{
				"delegate1": 10,
				"delegate2": 20,
				"delegate3": 30,
			},
			DelegateWallet: "delegate1",
		},
	}

	var (
		blobbersBefore []Blobber
		blobbersAfter  []Blobber
	)

	err = edb.Store.Get().Model(&Blobber{}).Omit(clause.Associations).Order("id ASC").Find(&blobbersBefore).Error
	require.NoError(t, err)

	err = edb.blobberSpecificRevenue(spus)
	require.NoError(t, err)

	err = edb.Store.Get().Model(&Blobber{}).Omit(clause.Associations).Order("id ASC").Find(&blobbersAfter).Error
	require.NoError(t, err)

	assert.Equal(t, blobbersBefore[0].TotalBlockRewards+10, blobbersAfter[0].TotalBlockRewards)
	assert.Equal(t, blobbersBefore[0].TotalStorageIncome, blobbersAfter[0].TotalStorageIncome)
	assert.Equal(t, blobbersBefore[0].TotalReadIncome, blobbersAfter[0].TotalReadIncome)
	assert.Equal(t, blobbersBefore[0].TotalSlashedStake, blobbersAfter[0].TotalSlashedStake)

	assert.Equal(t, blobbersBefore[1].TotalBlockRewards, blobbersAfter[1].TotalBlockRewards)
	assert.Equal(t, blobbersBefore[1].TotalStorageIncome+20, blobbersAfter[1].TotalStorageIncome)
	assert.Equal(t, blobbersBefore[1].TotalReadIncome, blobbersAfter[1].TotalReadIncome)
	assert.Equal(t, blobbersBefore[1].TotalSlashedStake, blobbersAfter[1].TotalSlashedStake)

	assert.Equal(t, blobbersBefore[2].TotalBlockRewards, blobbersAfter[2].TotalBlockRewards)
	assert.Equal(t, blobbersBefore[2].TotalStorageIncome, blobbersAfter[2].TotalStorageIncome)
	assert.Equal(t, blobbersBefore[2].TotalReadIncome+30, blobbersAfter[2].TotalReadIncome)
	assert.Equal(t, blobbersBefore[2].TotalSlashedStake, blobbersAfter[2].TotalSlashedStake)

	assert.Equal(t, blobbersBefore[3].TotalBlockRewards, blobbersAfter[3].TotalBlockRewards)
	assert.Equal(t, blobbersBefore[3].TotalStorageIncome, blobbersAfter[3].TotalStorageIncome)
	assert.Equal(t, blobbersBefore[3].TotalReadIncome, blobbersAfter[3].TotalReadIncome)
	assert.Equal(t, blobbersBefore[3].TotalSlashedStake+10, blobbersAfter[3].TotalSlashedStake)
}

func TestEventDb_updateBlobbersAllocatedSavedAndHealth(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	ids := setUpBlobbers(t, edb, 10, true)
	var blobber1, blobber2 Blobber
	now := common.Now()
	blobber1.ID = ids[0]
	blobber1.LastHealthCheck = now
	blobber1.SavedData = 300

	blobber2.ID = ids[1]
	blobber2.LastHealthCheck = now
	blobber2.SavedData = 200

	require.NoError(t, edb.updateBlobbersAllocatedSavedAndHealth([]Blobber{blobber1, blobber2}))

	b1, err := edb.GetBlobber(blobber1.ID)
	require.NoError(t, err)
	require.Equal(t, int64(300), b1.SavedData)
	require.Equal(t, now, b1.LastHealthCheck)

	b2, err := edb.GetBlobber(blobber2.ID)
	require.NoError(t, err)
	require.Equal(t, int64(200), b2.SavedData)
	require.Equal(t, now, b2.LastHealthCheck)
}

func compareBlobbers(t *testing.T, b1, b2 Blobber) {
	require.Equal(t, b1.ID, b2.ID)
	require.Equal(t, b1.WritePrice, b2.WritePrice)
	require.Equal(t, b1.ReadPrice, b2.ReadPrice)
	require.Equal(t, b1.TotalStake, b2.TotalStake)
	require.Equal(t, b1.NotAvailable, b2.NotAvailable)
	require.Equal(t, b1.LastHealthCheck, b2.LastHealthCheck)
}

func setUpBlobbers(t *testing.T, eventDb *EventDb, number int, withStats bool) []string {
	var ids []string
	var blobbers []Blobber
	for i := 0; i < number; i++ {
		blobber := Blobber{
			Provider: Provider{ID: fmt.Sprintf("somethingNew_%v", i)},
		}
		blobber.BaseURL = blobber.ID + ".com"
		if withStats {
			blobber.SavedData = testBlobberSavedData
		}

		ids = append(ids, blobber.ID)
		blobbers = append(blobbers, blobber)
	}
	require.NoError(t, eventDb.addBlobbers(blobbers))
	return ids
}

func buildMockBlobber(t *testing.T, pid string) Blobber {
	var curBlobber Blobber
	err := faker.FakeData(&curBlobber)
	require.NoError(t, err)
	curBlobber.ID = pid
	curBlobber.DelegateWallet = OwnerId
	curBlobber.BaseURL = fmt.Sprintf("http://url-%v.com", pid)
	curBlobber.WritePrice += 10
	curBlobber.Capacity += int64(curBlobber.TotalStake) * int64(GB)
	curBlobber.IsKilled = false
	curBlobber.IsShutdown = false
	curBlobber.Rewards = ProviderRewards{}
	return curBlobber
}

// -------------------------------------------------------------------------------------------------------------------------------------------

func TestGetBlobbersFromParams(t *testing.T) {

	edb, clean := GetTestEventDB(t)
	defer clean()

	blobbers := []Blobber{
		// Blobber 1 (Matched with the AllocationQuery)
		{
			Provider: Provider{
				ID:              "B000",
				LastHealthCheck: common.Timestamp(time.Now().Unix()),
				TotalStake:      currency.Coin(100),
			},
			BaseURL:     "https://blobber.zero",
			ReadPrice:   currency.Coin(50), // between 1 and 100
			WritePrice:  currency.Coin(50), // between 1 and 100
			Capacity:    5000000000000,
			Allocated:   0,
			OffersTotal: currency.Coin(1),
			// Capacity - Allocated = 5000000000000 (AllocationSize)
		},
		// Blobber 2 (Matched with the AllocationQuery)
		{
			Provider: Provider{
				ID:              "B001",
				LastHealthCheck: common.Timestamp(time.Now().Unix()),
				TotalStake:      currency.Coin(100),
			},
			BaseURL:     "https://blobber.one",
			ReadPrice:   currency.Coin(20),
			WritePrice:  currency.Coin(80),
			Capacity:    4000000000000,
			Allocated:   300000,
			OffersTotal: currency.Coin(1),
		},
		// Blobber 3 (Matched with the AllocationQuery)
		{
			Provider: Provider{
				ID:              "B002",
				LastHealthCheck: common.Timestamp(time.Now().Unix()),
				TotalStake:      currency.Coin(100),
			},
			BaseURL:     "https://blobber.two",
			ReadPrice:   currency.Coin(90),
			WritePrice:  currency.Coin(10),
			Capacity:    500000000000,
			Allocated:   50000,
			OffersTotal: currency.Coin(1),
		},
		// Blobber 4 (Doesn't have the capacity required)
		{
			Provider: Provider{
				ID:              "B003",
				LastHealthCheck: common.Timestamp(time.Now().Unix()),
				TotalStake:      currency.Coin(100),
			},
			BaseURL:     "https://blobber.three",
			ReadPrice:   currency.Coin(50),
			WritePrice:  currency.Coin(50),
			Capacity:    4000000000000,
			Allocated:   3999999999999,
			OffersTotal: currency.Coin(1),
		},
		// Blobber 5 (Not matched, WritePrice is too high)
		{
			Provider: Provider{
				ID:              "B004",
				LastHealthCheck: common.Timestamp(time.Now().Unix()),
				TotalStake:      currency.Coin(10),
			},
			BaseURL:     "https://blobber.four",
			ReadPrice:   currency.Coin(50),
			WritePrice:  currency.Coin(150),
			Capacity:    500000000000,
			Allocated:   50000,
			OffersTotal: currency.Coin(9),
		},
	}
	// Adding 5 blobbers, 3 will have parameters in line 2 shall not have and then the 2 will help me complete the function.

	for _, blobber := range blobbers {
		if err := edb.Store.Get().Create(&blobber).Error; err != nil {
			t.Fatalf("Failed to insert blobber: %v", err)
		}
	}

	// Creating curated Query
	allocation := AllocationQuery{
		ReadPriceRange: struct {
			Min int64
			Max int64
		}{1, 1e13},
		WritePriceRange: struct {
			Min int64
			Max int64
		}{1, 1e13},
		AllocationSize:     1 * 1024 * 1024 * 1024,
		AllocationSizeInGB: 1.0,
		NumberOfDataShards: 1,
	}

	pagination := common2.Pagination{
		Limit:        10,
		Offset:       0,
		IsDescending: true,
	}

	now := common.Timestamp(time.Now().Unix())
	healthCheckPeriod := 1 * time.Hour

	matchedBlobbers, err := edb.GetBlobbersFromParams(allocation, pagination, now, healthCheckPeriod)
	if err != nil {
		t.Fatalf("Error while retrieving blobbers: %v", err)
	}

	assert.Equal(t, 3, len(matchedBlobbers), "Expected 3 blobbers to match criteria")

	cleanupBlobbers(t, edb, blobbers)
}

func cleanupBlobbers(t *testing.T, edb *EventDb, blobbers []Blobber) {
	for _, blobber := range blobbers {
		if err := edb.deleteBlobber(blobber.Provider.ID); err != nil {
			t.Logf("Warning: Failed to cleanup blobber %v: %v", blobber.Provider.ID, err)
		}
	}
}

// -------------------------------------------------------------------------------------------------------------------------------------------
