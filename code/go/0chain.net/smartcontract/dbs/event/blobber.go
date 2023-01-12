package event

import (
	"fmt"
	"time"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	"0chain.net/core/common"
	common2 "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs"
	"gorm.io/gorm/clause"

	"github.com/0chain/common/core/currency"
	"github.com/guregu/null"
)

const ActiveBlobbersTimeLimit = 5 * time.Minute // 5 Minutes

type Blobber struct {
	Provider
	BaseURL string `json:"url" gorm:"uniqueIndex"`

	// geolocation
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`

	// terms
	ReadPrice        currency.Coin `json:"read_price"`
	WritePrice       currency.Coin `json:"write_price"`
	MinLockDemand    float64       `json:"min_lock_demand"`
	MaxOfferDuration int64         `json:"max_offer_duration"`

	Capacity  int64 `json:"capacity"`   // total blobber capacity
	Allocated int64 `json:"allocated"`  // allocated capacity
	Used      int64 `json:"used"`       // total of files saved on blobber
	SavedData int64 `json:"saved_data"` // total of files saved on blobber
	ReadData  int64 `json:"read_data"`

	OffersTotal currency.Coin `json:"offers_total"`
	//todo update
	TotalServiceCharge currency.Coin `json:"total_service_charge"`

	Name        string `json:"name" gorm:"name"`
	WebsiteUrl  string `json:"website_url" gorm:"website_url"`
	LogoUrl     string `json:"logo_url" gorm:"logo_url"`
	Description string `json:"description" gorm:"description"`

	ChallengesPassed    uint64  `json:"challenges_passed"`
	ChallengesCompleted uint64  `json:"challenges_completed"`
	OpenChallenges      uint64  `json:"open_challenges"`
	RankMetric          float64 `json:"rank_metric" gorm:"index"` // currently ChallengesPassed / ChallengesCompleted

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
		Where("last_health_check > ?", common.ToTime(now).Add(-ActiveBlobbersTimeLimit).Unix()).Limit(limit.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "capacity"},
		Desc:   limit.IsDescending,
	}).Find(&blobbers)
	return blobbers, result.Error
}

func (edb *EventDb) GetBlobbersByRank(limit common2.Pagination) ([]string, error) {
	var blobberIDs []string

	result := edb.Store.Get().
		Model(&Blobber{}).
		Select("id").
		Offset(limit.Offset).Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "rank_metric"},
			Desc:   true,
		}).Find(&blobberIDs)

	return blobberIDs, result.Error
}

func (edb *EventDb) GetAllBlobberId() ([]string, error) {
	var blobberIDs []string
	result := edb.Store.Get().Model(&Blobber{}).Select("id").Find(&blobberIDs)

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
			maxLatitude, minLatitude, maxLongitude, minLongitude).Offset(limit.Offset).Limit(limit.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "capacity"},
		Desc:   true,
	}).Find(&blobberIDs)

	return blobberIDs, result.Error
}

func (edb *EventDb) GetBlobbersFromIDs(ids []string) ([]Blobber, error) {
	var blobbers []Blobber
	result := edb.Store.Get().Preload("Rewards").
		Model(&Blobber{}).Order("id").Where("id IN ?", ids).Find(&blobbers)
	return blobbers, result.Error
}

func (edb *EventDb) deleteBlobber(id string) error {
	return edb.Store.Get().Model(&Blobber{}).Where("id = ?", id).Delete(&Blobber{}).Error
}

func (edb *EventDb) updateBlobbersAllocatedAndHealth(blobbers []Blobber) error {
	var ids []string
	var allocated []int64
	var lastHealthCheck []int64
	for _, m := range blobbers {
		ids = append(ids, m.ID)
		allocated = append(allocated, m.Allocated)
		lastHealthCheck = append(lastHealthCheck, int64(m.LastHealthCheck))
	}

	return CreateBuilder("blobbers", "id", ids).
		AddUpdate("allocated", allocated).AddUpdate("last_health_check", lastHealthCheck).Exec(edb).Error

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
	return blobberIDs, dbStore.Select("id").Find(&blobberIDs).Error
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
	return blobberIDs, dbStore.Select("id").Find(&blobberIDs).Error
}

func (edb *EventDb) addBlobbers(blobbers []Blobber) error {
	return edb.Store.Get().Create(&blobbers).Error
}

func (edb *EventDb) addOrOverwriteBlobber(blobbers []Blobber) error {
	err := edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(&blobbers).Error
	if err != nil {
		bids := make([]string, 0, len(blobbers))
		for _, b := range blobbers {
			bids = append(bids, b.ID)
		}
		logging.Logger.Debug("add or overwrite blobbers failed", zap.Any("ids", bids))
	}
	return err
}

func NewUpdateBlobberTotalStakeEvent(ID string, totalStake currency.Coin) (tag EventTag, data interface{}) {
	return TagUpdateBlobberTotalStake, Blobber{
		Provider: Provider{
			ID:         ID,
			TotalStake: totalStake,
		},
	}
}

func NewUpdateBlobberTotalUnStakeEvent(ID string, totalUnStake currency.Coin) (tag EventTag, data interface{}) {
	return TagUpdateBlobberTotalUnStake, Blobber{
		Provider: Provider{
			ID:         ID,
			TotalStake: totalUnStake,
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

func (edb *EventDb) updateBlobbersTotalUnStakes(blobbers []Blobber) error {
	var provs []Provider
	for _, b := range blobbers {
		provs = append(provs, b.Provider)
	}
	return edb.updateProvidersTotalUnStakes(provs, "blobbers")
}

func mergeUpdateBlobberTotalStakesEvents() *eventsMergerImpl[Blobber] {
	return newEventsMerger[Blobber](TagUpdateBlobberTotalStake, withUniqueEventOverwrite())
}
func mergeUpdateBlobberTotalUnStakesEvents() *eventsMergerImpl[Blobber] {
	return newEventsMerger[Blobber](TagUpdateBlobberTotalUnStake, withUniqueEventOverwrite())
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
		AddUpdate("saved_data", savedData, "blobbers.saved_data + t.saved_data").Exec(edb).Error
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

type ChallengeStatsDeltas struct {
	Id             string `json:"id"`
	PassedDelta    int64  `json:"passed_delta"`
	CompletedDelta int64  `json:"completed_delta"`
	OpenDelta      int64  `json:"open_delta"`
}

func mergeUpdateBlobberChallengesEvents() *eventsMergerImpl[ChallengeStatsDeltas] {
	return newEventsMerger[ChallengeStatsDeltas](TagUpdateBlobberChallenge, withBlobberChallengesMerged())
}

func withBlobberChallengesMerged() eventMergeMiddleware {
	return withEventMerge(func(a, b *ChallengeStatsDeltas) (*ChallengeStatsDeltas, error) {
		a.CompletedDelta += b.CompletedDelta
		a.PassedDelta += b.PassedDelta
		a.OpenDelta += b.OpenDelta
		return a, nil
	})
}

func mergeAddChallengesToBlobberEvents() *eventsMergerImpl[ChallengeStatsDeltas] {
	return newEventsMerger[ChallengeStatsDeltas](TagUpdateBlobberOpenChallenges, withBlobberChallengesMerged())
}

func (edb *EventDb) updateOpenBlobberChallenges(deltas []ChallengeStatsDeltas) error {
	return edb.Store.Get().Raw(sqlUpdateOpenChallenges(deltas)).Scan(&Blobber{}).Error
}

func sqlUpdateOpenChallenges(deltas []ChallengeStatsDeltas) string {
	if len(deltas) == 0 {
		return ""
	}
	sql := "UPDATE blobbers \n"
	sql += "SET "
	sql += "  open_challenges = open_challenges + v.open\n"
	sql += "FROM ( VALUES"
	first := true
	for _, delta := range deltas {
		if first {
			first = false
		} else {
			sql += ","
		}
		sql += fmt.Sprintf("('%s', %d)", delta.Id, delta.OpenDelta)
	}
	sql += "  )\n"
	sql += "AS v (id, open)\n"
	sql += "WHERE\n"
	sql += "  blobbers.id = v.id"

	return sql
}

func (edb *EventDb) updateBlobberChallenges(deltas []ChallengeStatsDeltas) error {
	return edb.Store.Get().Raw(sqlUpdateBlobberChallenges(deltas)).Scan(&Blobber{}).Error
}

// ref https://www.postgresql.org/docs/9.1/sql-values.html
func sqlUpdateBlobberChallenges(deltas []ChallengeStatsDeltas) string {
	if len(deltas) == 0 {
		return ""
	}
	sql := "UPDATE blobbers \n"
	sql += "SET "
	sql += "  challenges_completed = challenges_completed + v.completed,\n"
	sql += "  challenges_passed = challenges_passed + v.passed\n"
	//sql += ",  rank_metric = (challenges_passed + v.passed)::FLOAT /  (blobbers.challenges_completed + v.completed)::FLOAT)::DECIMAL(10,3)\n" todo
	sql += "FROM ( VALUES "
	first := true
	for _, delta := range deltas {
		if first {
			first = false
		} else {
			sql += ",\n"
		}
		sql += fmt.Sprintf("('%s', %d, %d)", delta.Id, delta.PassedDelta, delta.CompletedDelta)
	}
	sql += ")\n"
	sql += "AS v (id, passed, completed)\n"
	sql += "WHERE\n"
	sql += "blobbers.id = v.id"

	return sql
}

func mergeBlobberHealthCheckEvents() *eventsMergerImpl[dbs.DbHealthCheck] {
	return newEventsMerger[dbs.DbHealthCheck](TagBlobberHealthCheck, withUniqueEventOverwrite())
}
