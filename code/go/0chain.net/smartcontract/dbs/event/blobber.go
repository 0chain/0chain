package event

import (
	"fmt"
	"time"

	"0chain.net/core/common"
	common2 "0chain.net/smartcontract/common"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"0chain.net/chaincore/currency"
	"github.com/guregu/null"
)

const ActiveBlobbersTimeLimit = 60 * 60 // 1 hour

type Blobber struct {
	gorm.Model
	BlobberID string `json:"id" gorm:"uniqueIndex"`
	BaseURL   string `json:"url" gorm:"uniqueIndex"`

	// geolocation
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`

	// terms
	ReadPrice        currency.Coin `json:"read_price"`
	WritePrice       currency.Coin `json:"write_price"`
	MinLockDemand    float64       `json:"min_lock_demand"`
	MaxOfferDuration int64         `json:"max_offer_duration"`

	Capacity        int64 `json:"capacity"`  // total blobber capacity
	Allocated       int64 `json:"allocated"` // allocated capacity
	Used            int64 `json:"used"`      // total of files saved on blobber
	LastHealthCheck int64 `json:"last_health_check"`
	SavedData       int64 `json:"saved_data"`

	// stake_pool_settings
	DelegateWallet string        `json:"delegate_wallet"`
	MinStake       currency.Coin `json:"min_stake"`
	MaxStake       currency.Coin `json:"max_stake"`
	NumDelegates   int           `json:"num_delegates"`
	ServiceCharge  float64       `json:"service_charge"`

	OffersTotal  currency.Coin `json:"offers_total"`
	UnstakeTotal currency.Coin `json:"unstake_total"`
	TotalStake   currency.Coin `json:"total_stake"`

	Name        string `json:"name" gorm:"name"`
	WebsiteUrl  string `json:"website_url" gorm:"website_url"`
	LogoUrl     string `json:"logo_url" gorm:"logo_url"`
	Description string `json:"description" gorm:"description"`

	ChallengesPassed    uint64  `json:"challenges_passed"`
	ChallengesCompleted uint64  `json:"challenges_completed"`
	RankMetric          float64 `json:"rank_metric" gorm:"index"` // currently ChallengesPassed / ChallengesCompleted

	Rewards ProviderRewards `json:"rewards" gorm:"foreignKey:BlobberID;references:ProviderID"`

	WriteMarkers []WriteMarker `gorm:"foreignKey:BlobberID;references:BlobberID"`
	ReadMarkers  []ReadMarker  `gorm:"foreignKey:BlobberID;references:BlobberID"`
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
		Model(&Blobber{}).Where("blobber_id = ?", id).First(&blobber).Error
	if err != nil {
		return nil, fmt.Errorf("error retrieving blobber %v, error %v", id, err)
	}
	return &blobber, nil
}

func (edb *EventDb) GetBlobberRank(blobberId string) (int64, error) {
	blobber, err := edb.GetBlobber(blobberId)
	if err != nil {
		return 0, err
	}
	var rank int64
	result := edb.Store.Get().
		Model(&Blobber{}).
		Where("rank_metric > ?", blobber.RankMetric).
		Count(&rank)
	return rank + 1, result.Error
}

func (edb *EventDb) BlobberTotalCapacity() (int64, error) {
	var total int64
	return total, edb.Store.Get().Model(&Blobber{}).
		Select("SUM(capacity)").
		Find(&total).Error
}

func (edb *EventDb) BlobberAverageWritePrice() (float64, error) {
	var average float64
	return average, edb.Store.Get().Model(&Blobber{}).
		Select("AVG(write_price)").
		Find(&average).Error
}

func (edb *EventDb) TotalUsedData() (int64, error) {
	var total int64
	return total, edb.Store.Get().Model(&Blobber{}).
		Select("sum(used)").
		Find(&total).Error
}

func (edb *EventDb) GetBlobbers(limit common2.Pagination) ([]Blobber, error) {
	var blobbers []Blobber
	result := edb.Store.Get().
		Preload("Rewards").
		Model(&Blobber{}).Offset(limit.Offset).Limit(limit.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "capacity"},
		Desc:   limit.IsDescending,
	}).Find(&blobbers)

	return blobbers, result.Error
}

func (edb *EventDb) GetActiveBlobbers(limit common2.Pagination) ([]Blobber, error) {
	now := common.Now()
	var blobbers []Blobber
	result := edb.Store.Get().
		Preload("Rewards").
		Model(&Blobber{}).Offset(limit.Offset).
		Where("last_health_check > ?", common.ToTime(now).Add(-time.Hour).Unix()).Limit(limit.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "capacity"},
		Desc:   limit.IsDescending,
	}).Find(&blobbers)
	return blobbers, result.Error
}

func (edb *EventDb) GetBlobbersByRank(limit common2.Pagination) ([]string, error) {
	var blobberIDs []string

	result := edb.Store.Get().
		Model(&Blobber{}).
		Select("blobber_id").
		Offset(limit.Offset).Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "rank_metric"},
			Desc:   true,
		}).Find(&blobberIDs)

	return blobberIDs, result.Error
}

func (edb *EventDb) GetAllBlobberId() ([]string, error) {
	var blobberIDs []string
	result := edb.Store.Get().Model(&Blobber{}).Select("blobber_id").Find(&blobberIDs)

	return blobberIDs, result.Error
}

func (edb *EventDb) GeBlobberByLatLong(
	maxLatitude, minLatitude, maxLongitude, minLongitude float64, limit common2.Pagination,
) ([]string, error) {
	var blobberIDs []string
	result := edb.Store.Get().
		Model(&Blobber{}).
		Select("blobber_id").
		Where("latitude <= ? AND latitude >= ? AND longitude <= ? AND longitude >= ? ",
			maxLatitude, minLatitude, maxLongitude, minLongitude).Offset(limit.Offset).Limit(limit.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "capacity"},
		Desc:   true,
	}).Find(&blobberIDs)

	return blobberIDs, result.Error
}

func (edb *EventDb) GetBlobbersFromIDs(ids []string) ([]Blobber, error) {
	var blobbers []Blobber
	result := edb.Store.Get().Preload("Rewards").
		Model(&Blobber{}).Order("id").Where("blobber_id IN ?", ids).Find(&blobbers)
	return blobbers, result.Error
}

func (edb *EventDb) deleteBlobber(id string) error {
	return edb.Store.Get().Model(&Blobber{}).Where("blobber_id = ?", id).Delete(&Blobber{}).Error
}

func (edb *EventDb) updateBlobbersAllocatedAndHealth(blobbers []Blobber) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "blobber_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"allocated", "last_health_check"}),
	}).Create(&blobbers).Error
}

func mergeUpdateBlobbersEvents() *eventsMergerImpl[Blobber] {
	return newEventsMerger[Blobber](TagUpdateBlobberAllocatedHealth, withUniqueEventOverwrite())
}

func (edb *EventDb) GetBlobberCount() (int64, error) {
	var count int64
	res := edb.Store.Get().Model(Blobber{}).Count(&count)

	return count, res.Error
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
	dbStore = dbStore.Where("base_url IN ?", urls).Limit(data.Limit).Offset(data.Offset).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "id"},
		Desc:   data.IsDescending,
	})
	var blobberIDs []string
	return blobberIDs, dbStore.Select("blobber_id").Find(&blobberIDs).Error
}

func (edb *EventDb) GetBlobbersFromParams(allocation AllocationQuery, limit common2.Pagination, now common.Timestamp) ([]string, error) {
	dbStore := edb.Store.Get().Model(&Blobber{})
	dbStore = dbStore.Where("read_price between ? and ?", allocation.ReadPriceRange.Min, allocation.ReadPriceRange.Max)
	dbStore = dbStore.Where("write_price between ? and ?", allocation.WritePriceRange.Min, allocation.WritePriceRange.Max)
	dbStore = dbStore.Where("max_offer_duration >= ?", allocation.MaxOfferDuration.Nanoseconds())
	dbStore = dbStore.Where("capacity - allocated >= ?", allocation.AllocationSize)
	dbStore = dbStore.Where("last_health_check > ?", common.ToTime(now).Add(-time.Hour).Unix())
	dbStore = dbStore.Where("(total_stake - offers_total) > ? * write_price", allocation.AllocationSizeInGB)
	dbStore = dbStore.Limit(limit.Limit).Offset(limit.Offset).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "capacity"},
		Desc:   limit.IsDescending,
	})
	var blobberIDs []string
	return blobberIDs, dbStore.Select("blobber_id").Find(&blobberIDs).Error
}

func (edb *EventDb) addBlobbers(blobbers []Blobber) error {
	return edb.Store.Get().Create(&blobbers).Error
}

func (edb *EventDb) addOrOverwriteBlobber(blobbers []Blobber) error {
	err := edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "blobber_id"}},
		UpdateAll: true,
	}).Create(&blobbers).Error
	if err != nil {
		bids := make([]string, 0, len(blobbers))
		for _, b := range blobbers {
			bids = append(bids, b.BlobberID)
		}
		logging.Logger.Debug("add or overwrite blobbers failed", zap.Any("ids", bids))
	}
	return err
}

func NewUpdateBlobberTotalStakeEvent(ID string, totalStake currency.Coin) (tag EventTag, data interface{}) {
	return TagUpdateBlobberTotalStake, Blobber{
		BlobberID:  ID,
		TotalStake: totalStake,
	}
}

func NewUpdateBlobberTotalOffersEvent(ID string, totalOffers currency.Coin) (tag EventTag, data interface{}) {
	return TagUpdateBlobberTotalOffers, Blobber{
		BlobberID:   ID,
		OffersTotal: totalOffers,
	}
}

func (edb *EventDb) updateBlobbersTotalStakes(blobbers []Blobber) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "blobber_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"total_stake"}),
	}).Create(&blobbers).Error
}

func mergeUpdateBlobberTotalStakesEvents() *eventsMergerImpl[Blobber] {
	return newEventsMerger[Blobber](TagUpdateBlobberTotalStake, withBlobberTotalStakesAdded())
}

func withBlobberTotalStakesAdded() eventMergeMiddleware {
	return withEventMerge(func(a, b *Blobber) (*Blobber, error) {
		a.TotalStake += b.TotalStake
		return a, nil
	})
}

func (edb *EventDb) updateBlobbersTotalOffers(blobbers []Blobber) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "blobber_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"offers_total"}),
	}).Create(&blobbers).Error
}

func mergeUpdateBlobberTotalOffersEvents() *eventsMergerImpl[Blobber] {
	return newEventsMerger[Blobber](TagUpdateBlobberTotalOffers, withBlobberTotalOffersAdded())
}

func withBlobberTotalOffersAdded() eventMergeMiddleware {
	return withEventMerge(func(a, b *Blobber) (*Blobber, error) {
		a.OffersTotal += b.OffersTotal
		return a, nil
	})
}

func (edb *EventDb) updateBlobbersStats(blobbers []Blobber) error {
	vs := map[string]interface{}{
		"used":       gorm.Expr("blobbers.used + excluded.used"),
		"saved_data": gorm.Expr("blobbers.saved_data + excluded.saved_data"),
	}

	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "blobber_id"}},
		DoUpdates: clause.Assignments(vs),
	}).Create(&blobbers).Error
}

func mergeUpdateBlobberStatsEvents() *eventsMergerImpl[Blobber] {
	return newEventsMerger[Blobber](TagUpdateBlobberStat, withBlobberStatsMerged())
}

func withBlobberStatsMerged() eventMergeMiddleware {
	return withEventMerge(func(a, b *Blobber) (*Blobber, error) {
		a.Used += b.Used
		a.SavedData += b.SavedData
		return a, nil
	})
}

func mergeUpdateBlobberChallengesEvents() *eventsMergerImpl[Blobber] {
	return newEventsMerger[Blobber](TagUpdateBlobberChallenge, withBlobberChallengesMerged())
}

func withBlobberChallengesMerged() eventMergeMiddleware {
	return withEventMerge(func(a, b *Blobber) (*Blobber, error) {
		a.ChallengesCompleted += b.ChallengesCompleted
		a.ChallengesPassed += b.ChallengesPassed
		return a, nil
	})
}

func (edb *EventDb) updateBlobberChallenges(blobbers []Blobber) error {
	vs := map[string]interface{}{
		"challenges_completed": gorm.Expr("blobbers.challenges_completed + excluded.challenges_completed"),
		"challenges_passed":    gorm.Expr("blobbers.challenges_passed + excluded.challenges_passed"),
		"rank_metric":          gorm.Expr("((blobbers.challenges_passed + excluded.challenges_passed)::FLOAT / (blobbers.challenges_completed + excluded.challenges_completed)::FLOAT)::DECIMAL(10,3)"),
	}

	return edb.Store.Get().Model(&Blobber{}).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "blobber_id"}},
		DoUpdates: clause.Assignments(vs),
	}).Create(blobbers).Error
}
