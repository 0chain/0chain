package event

import (
	"fmt"
	"time"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	"0chain.net/core/common"
	common2 "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"
	"gorm.io/gorm/clause"

	"github.com/0chain/common/core/currency"
	"github.com/guregu/null"
)

type Blobber struct {
	Provider
	BaseURL string `json:"url" gorm:"uniqueIndex"`

	// geolocation
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`

	// terms
	ReadPrice     currency.Coin `json:"read_price"`
	WritePrice    currency.Coin `json:"write_price"`
	MinLockDemand float64       `json:"min_lock_demand"`

	Capacity     int64 `json:"capacity"`   // total blobber capacity
	Allocated    int64 `json:"allocated"`  // allocated capacity
	Used         int64 `json:"used"`       // total of files saved on blobber
	SavedData    int64 `json:"saved_data"` // total of files saved on blobber
	ReadData     int64 `json:"read_data"`
	NotAvailable bool  `json:"not_available"`

	OffersTotal currency.Coin `json:"offers_total"`
	//todo update
	TotalServiceCharge currency.Coin `json:"total_service_charge"`

	Name        string `json:"name" gorm:"name"`
	WebsiteUrl  string `json:"website_url" gorm:"website_url"`
	LogoUrl     string `json:"logo_url" gorm:"logo_url"`
	Description string `json:"description" gorm:"description"`

	ChallengesPassed    uint64        `json:"challenges_passed"`
	ChallengesCompleted uint64        `json:"challenges_completed"`
	OpenChallenges      uint64        `json:"open_challenges"`
	RankMetric          float64       `json:"rank_metric" gorm:"index"` // currently ChallengesPassed / ChallengesCompleted
	TotalBlockRewards   currency.Coin `json:"total_block_rewards"`
	TotalStorageIncome  currency.Coin `json:"total_storage_income"`
	TotalReadIncome     currency.Coin `json:"total_read_income"`
	TotalSlashedStake   currency.Coin `json:"total_slashed_stake"`

	WriteMarkers []WriteMarker `gorm:"foreignKey:BlobberID;references:ID"`
	ReadMarkers  []ReadMarker  `gorm:"foreignKey:BlobberID;references:ID"`

	CreationRound int64 `json:"creation_round" gorm:"index:idx_blobber_creation_round"`
}

// BlobberPriceRange represents a price range allowed by user to filter blobbers.
type BlobberPriceRange struct {
	Min null.Int `json:"min"`
	Max null.Int `json:"max"`
}

func (edb *EventDb) GetBlobber(id string) (*Blobber, error) {
	var blobber Blobber
	err := edb.Store.Get().
		Preload("Rewards").
		Model(&Blobber{}).Where("id = ?", id).First(&blobber).Error
	if err != nil {
		return nil, fmt.Errorf("error retrieving blobber %v, error %v", id, err)
	}
	return &blobber, nil
}

func (edb *EventDb) GetBlobbers(limit common2.Pagination) ([]Blobber, error) {
	var blobbers []Blobber
	result := edb.Store.Get().
		Preload("Rewards").
		Model(&Blobber{}).Offset(limit.Offset).
		Where("is_killed = ? AND is_shutdown = ?", false, false).
		Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "capacity"},
			Desc:   limit.IsDescending,
		}).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
			Desc:   limit.IsDescending,
		}).
		Find(&blobbers)

	return blobbers, result.Error
}

func (edb *EventDb) GetActiveBlobbers(limit common2.Pagination, healthCheckTimeLimit time.Duration) ([]Blobber, error) {
	now := common.Now()
	var blobbers []Blobber
	result := edb.Store.Get().
		Preload("Rewards").
		Model(&Blobber{}).Offset(limit.Offset).
		Where("last_health_check > ? AND is_killed = ? AND is_shutdown = ?",
			common.ToTime(now).Add(-healthCheckTimeLimit).Unix(), false, false).
		Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "capacity"},
			Desc:   limit.IsDescending,
		}).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
			Desc:   limit.IsDescending,
		}).
		Find(&blobbers)
	return blobbers, result.Error
}

func (edb *EventDb) GetBlobbersByRank(limit common2.Pagination) ([]string, error) {
	var blobberIDs []string

	result := edb.Store.Get().
		Model(&Blobber{}).
		Select("id").
		Where("is_killed = ? AND is_shutdown = ?", false, false).
		Offset(limit.Offset).Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "rank_metric"},
			Desc:   true,
		}).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
			Desc:   true,
		}).
		Find(&blobberIDs)

	return blobberIDs, result.Error
}

func (edb *EventDb) GeBlobberByLatLong(
	maxLatitude, minLatitude, maxLongitude, minLongitude float64, limit common2.Pagination,
) ([]string, error) {
	var blobberIDs []string
	result := edb.Store.Get().
		Model(&Blobber{}).
		Select("id").
		Where("latitude <= ? AND latitude >= ? AND longitude <= ? AND longitude >= ? ",
			maxLatitude, minLatitude, maxLongitude, minLongitude).
		Offset(limit.Offset).
		Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "capacity"},
			Desc:   true,
		}).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
			Desc:   true,
		}).
		Find(&blobberIDs)

	return blobberIDs, result.Error
}

func (edb *EventDb) GetBlobbersFromIDs(ids []string) ([]Blobber, error) {
	var blobbers []Blobber
	result := edb.Store.Get().Preload("Rewards").
		Model(&Blobber{}).
		Order("id").
		Where("id IN ?", ids).
		Find(&blobbers)
	return blobbers, result.Error
}

func (edb *EventDb) deleteBlobber(id string) error {
	return edb.Store.Get().Model(&Blobber{}).Where("id = ?", id).Delete(&Blobber{}).Error
}

func (edb *EventDb) updateBlobbersAllocatedSavedAndHealth(blobbers []Blobber) error {
	var ids []string
	var allocated []int64
	var savedData []int64
	var lastHealthCheck []int64
	for _, m := range blobbers {
		ids = append(ids, m.ID)
		allocated = append(allocated, m.Allocated)
		savedData = append(savedData, m.SavedData)
		lastHealthCheck = append(lastHealthCheck, int64(m.LastHealthCheck))
	}

	return CreateBuilder("blobbers", "id", ids).
		AddUpdate("allocated", allocated).
		AddUpdate("last_health_check", lastHealthCheck).
		AddUpdate("saved_data", savedData).
		Exec(edb).Error

}

func mergeUpdateBlobbersEvents() *eventsMergerImpl[Blobber] {
	return newEventsMerger[Blobber](TagUpdateBlobberAllocatedSavedHealth, withUniqueEventOverwrite())
}

type AllocationQuery struct {
	MaxOfferDuration time.Duration
	ReadPriceRange   struct {
		Min int64
		Max int64
	}
	WritePriceRange struct {
		Min int64
		Max int64
	}
	AllocationSize     int64
	AllocationSizeInGB float64
	NumberOfDataShards int
}

func (edb *EventDb) GetBlobberIdsFromUrls(urls []string, data common2.Pagination) ([]string, error) {
	dbStore := edb.Store.Get().Model(&Blobber{})
	dbStore = dbStore.Where("base_url IN ?", urls).
		Limit(data.Limit).
		Offset(data.Offset).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
			Desc:   data.IsDescending,
		})
	var blobberIDs []string
	return blobberIDs, dbStore.Select("id").Find(&blobberIDs).Error
}

func (edb *EventDb) GetBlobbersFromParams(allocation AllocationQuery, limit common2.Pagination, now common.Timestamp, healthCheckPeriod time.Duration) ([]string, error) {
	dbStore := edb.Store.Get().Model(&Blobber{})
	dbStore = dbStore.Where("read_price between ? and ?", allocation.ReadPriceRange.Min, allocation.ReadPriceRange.Max)
	dbStore = dbStore.Where("write_price between ? and ?", allocation.WritePriceRange.Min, allocation.WritePriceRange.Max)
	dbStore = dbStore.Where("capacity - allocated >= ?", allocation.AllocationSize)
	dbStore = dbStore.Where("last_health_check > ?", common.ToTime(now).Add(-healthCheckPeriod).Unix())
	dbStore = dbStore.Where("(total_stake - offers_total) > ? * write_price", allocation.AllocationSizeInGB)
	dbStore = dbStore.Where("is_killed = false")
	dbStore = dbStore.Where("is_shutdown = false")
	dbStore = dbStore.Where("not_available = false")
	dbStore = dbStore.Limit(limit.Limit).
		Offset(limit.Offset).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "write_price"},
			Desc:   limit.IsDescending,
		}).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
			Desc:   limit.IsDescending,
		})
	var blobberIDs []string
	return blobberIDs, dbStore.Select("id").Find(&blobberIDs).Error
}

func (edb *EventDb) addBlobbers(blobbers []Blobber) error {
	return edb.Store.Get().Create(&blobbers).Error
}

func (edb *EventDb) updateBlobber(blobbers []Blobber) error {
	ts := time.Now()

	// fields match storagesc.emitUpdateBlobber
	updateColumns := []string{
		"latitude",
		"longitude",
		"read_price",
		"write_price",
		"min_lock_demand",
		"max_offer_duration",
		"capacity",
		"allocated",
		"saved_data",
		"not_available",
		"offers_total",
		"delegate_wallet",
		"num_delegates",
		"service_charge",
		"last_health_check",
		"total_stake",
	}
	columns, err := Columnize(blobbers)
	if err != nil {
		return err
	}
	ids, ok := columns["id"]
	if !ok {
		return common.NewError("update_blobbers", "no id field provided in event Data")
	}

	updater := CreateBuilder("blobbers", "id", ids)
	for _, fieldKey := range updateColumns {
		if fieldKey == "id" {
			continue
		}

		fieldList, ok := columns[fieldKey]
		if !ok {
			logging.Logger.Warn("update_blobbers required update field not found in event data", zap.String("field", fieldKey))
		} else {
			updater = updater.AddUpdate(fieldKey, fieldList)
		}
	}

	defer func() {
		du := time.Since(ts)
		if du.Milliseconds() > 50 {
			logging.Logger.Debug("event db - update blobbers slow",
				zap.Duration("duration", du),
				zap.Int("num", len(blobbers)))
		}
	}()

	return updater.Exec(edb).Debug().Error
}

func NewUpdateBlobberTotalStakeEvent(ID string, totalStake currency.Coin) (tag EventTag, data interface{}) {
	return TagUpdateBlobberTotalStake, Blobber{
		Provider: Provider{
			ID:         ID,
			TotalStake: totalStake,
		},
	}
}

func NewUpdateBlobberTotalOffersEvent(ID string, totalOffers currency.Coin) (tag EventTag, data interface{}) {
	return TagUpdateBlobberTotalOffers, Blobber{
		Provider:    Provider{ID: ID},
		OffersTotal: totalOffers,
	}
}

func (edb *EventDb) updateBlobbersTotalStakes(blobbers []Blobber) error {
	var provs []Provider
	for _, b := range blobbers {
		provs = append(provs, b.Provider)
	}
	return edb.updateProviderTotalStakes(provs, "blobbers")
}

func mergeUpdateBlobberTotalStakesEvents() *eventsMergerImpl[Blobber] {
	return newEventsMerger[Blobber](TagUpdateBlobberTotalStake, withUniqueEventOverwrite())
}

func (edb *EventDb) updateBlobbersTotalOffers(blobbers []Blobber) error {
	var ids []string
	var offers []uint64
	for _, m := range blobbers {
		ids = append(ids, m.ID)
		offers = append(offers, uint64(m.OffersTotal))
	}

	return CreateBuilder("blobbers", "id", ids).
		AddUpdate("offers_total", offers).Exec(edb).Error
}

func mergeUpdateBlobberTotalOffersEvents() *eventsMergerImpl[Blobber] {
	return newEventsMerger[Blobber](TagUpdateBlobberTotalOffers, withUniqueEventOverwrite())
}

func (edb *EventDb) updateBlobbersStats(blobbers []Blobber) error {
	var ids []string
	var used []int64
	var savedData []int64
	for _, m := range blobbers {
		ids = append(ids, m.ID)
		used = append(used, m.Used)
		savedData = append(savedData, m.SavedData)
	}

	return CreateBuilder("blobbers", "id", ids).
		AddUpdate("used", used, "blobbers.used + t.used").
		AddUpdate("saved_data", savedData, "blobbers.saved_data + t.saved_data").
		AddUpdate("read_data", savedData, "blobbers.read_data + t.read_data").Exec(edb).Error
}

func mergeUpdateBlobberStatsEvents() *eventsMergerImpl[Blobber] {
	return newEventsMerger[Blobber](TagUpdateBlobberStat, withBlobberStatsMerged())
}

func withBlobberStatsMerged() eventMergeMiddleware {
	return withEventMerge(func(a, b *Blobber) (*Blobber, error) {
		a.Used += b.Used
		a.SavedData += b.SavedData
		a.ReadData += b.ReadData
		return a, nil
	})
}

func mergeUpdateBlobberChallengesEvents() *eventsMergerImpl[Blobber] {
	return newEventsMerger[Blobber](TagUpdateBlobberChallenge, withUniqueEventOverwrite())
}

func mergeAddChallengesToBlobberEvents() *eventsMergerImpl[Blobber] {
	return newEventsMerger[Blobber](TagUpdateBlobberOpenChallenges, withUniqueEventOverwrite())
}

func (edb *EventDb) updateOpenBlobberChallenges(blobbers []Blobber) error {

	blobberIdList := make([]string, 0, len(blobbers))
	openChallengesList := make([]uint64, 0, len(blobbers))

	for _, blobber := range blobbers {
		blobberIdList = append(blobberIdList, blobber.ID)
		openChallengesList = append(openChallengesList, blobber.OpenChallenges)
	}

	return CreateBuilder("blobbers", "id", blobberIdList).
		AddUpdate("open_challenges", openChallengesList).Exec(edb).Error
}

func (edb *EventDb) updateBlobberChallenges(blobbers []Blobber) error {
	blobberIdList := make([]string, 0, len(blobbers))
	challengesPassedList := make([]uint64, 0, len(blobbers))
	challengesCompletedList := make([]uint64, 0, len(blobbers))

	for _, blobber := range blobbers {
		blobberIdList = append(blobberIdList, blobber.ID)
		challengesPassedList = append(challengesPassedList, blobber.ChallengesPassed)
		challengesCompletedList = append(challengesCompletedList, blobber.ChallengesCompleted)
	}

	return CreateBuilder("blobbers", "id", blobberIdList).
		AddUpdate("challenges_passed", challengesPassedList).
		AddUpdate("challenges_completed", challengesCompletedList).Exec(edb).Error
}

func (edb *EventDb) blobberSpecificRevenue(spus []dbs.StakePoolReward) error {
	var (
		ids                []string
		totalBlockRewards  []int64
		totalStorageIncome []int64
		totalReadIncome    []int64
		totalSlashedStake  []int64
		totalChanges       = 0
	)

	blobberIdx := -1
	for _, spu := range spus {
		if spu.Type != spenum.Blobber {
			continue
		}
		blobberIdx++
		ids = append(ids, spu.ProviderID.ID)
		totalBlockRewards = append(totalBlockRewards, 0)
		totalStorageIncome = append(totalStorageIncome, 0)
		totalReadIncome = append(totalReadIncome, 0)
		totalSlashedStake = append(totalSlashedStake, 0)

		switch spu.RewardType {
		case spenum.BlockRewardBlobber:
			totalChanges++
			totalBlockRewards[blobberIdx] = int64(spu.Reward)
		case spenum.ChallengePassReward:
			totalChanges++
			totalStorageIncome[blobberIdx] = int64(spu.Reward)
		case spenum.FileDownloadReward:
			totalChanges++
			totalReadIncome[blobberIdx] = int64(spu.Reward)
		case spenum.ChallengeSlashPenalty:
			totalChanges++
			for _, penalty := range spu.DelegatePenalties {
				totalSlashedStake[blobberIdx] += int64(penalty)
			}
		}
	}

	if totalChanges == 0 {
		return nil
	}

	return CreateBuilder("blobbers", "id", ids).
		AddUpdate("total_block_rewards", totalBlockRewards, "blobbers.total_block_rewards + t.total_block_rewards").
		AddUpdate("total_storage_income", totalStorageIncome, "blobbers.total_storage_income + t.total_storage_income").
		AddUpdate("total_read_income", totalReadIncome, "blobbers.total_read_income + t.total_read_income").
		AddUpdate("total_slashed_stake", totalSlashedStake, "blobbers.total_slashed_stake + t.total_slashed_stake").
		Exec(edb).Debug().Error
}

func mergeBlobberHealthCheckEvents() *eventsMergerImpl[dbs.DbHealthCheck] {
	return newEventsMerger[dbs.DbHealthCheck](TagBlobberHealthCheck, withUniqueEventOverwrite())
}
