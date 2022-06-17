package event

import (
	"errors"
	"fmt"
	"time"

	"0chain.net/chaincore/currency"
	"0chain.net/smartcontract/dbs"

	"github.com/guregu/null"
	"gorm.io/gorm"
)

type Blobber struct {
	gorm.Model
	BlobberID string `json:"id" gorm:"uniqueIndex"`
	BaseURL   string `json:"url" gorm:"uniqueIndex"`

	// geolocation
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`

	// terms
	ReadPrice               currency.Coin `json:"read_price"`
	WritePrice              currency.Coin `json:"write_price"`
	MinLockDemand           float64       `json:"min_lock_demand"`
	MaxOfferDuration        int64         `json:"max_offer_duration"`
	ChallengeCompletionTime int64         `json:"challenge_completion_time"`

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

	OffersTotal        currency.Coin `json:"offers_total"`
	UnstakeTotal       currency.Coin `json:"unstake_total"`
	Reward             currency.Coin `json:"reward"`
	TotalServiceCharge currency.Coin `json:"total_service_charge"`
	TotalStake         currency.Coin `json:"total_stake"`

	Name        string `json:"name" gorm:"name"`
	WebsiteUrl  string `json:"website_url" gorm:"website_url"`
	LogoUrl     string `json:"logo_url" gorm:"logo_url"`
	Description string `json:"description" gorm:"description"`

	WriteMarkers []WriteMarker `gorm:"foreignKey:BlobberID;references:BlobberID"`
	ReadMarkers  []ReadMarker  `gorm:"foreignKey:BlobberID;references:BlobberID"`
}

// BlobberPriceRange represents a price range allowed by user to filter blobbers.
type BlobberPriceRange struct {
	Min null.Int `json:"min"`
	Max null.Int `json:"max"`
}

type blobberAggregateStats struct {
	Reward             currency.Coin `json:"reward"`
	TotalServiceCharge currency.Coin `json:"total_service_charge"`
}

func (edb *EventDb) GetBlobber(id string) (*Blobber, error) {
	var blobber Blobber
	err := edb.Store.Get().Model(&Blobber{}).Where("blobber_id = ?", id).First(&blobber).Error
	if err != nil {
		return nil, fmt.Errorf("error retrieving blobber %v, error %v", id, err)
	}
	return &blobber, nil
}

func (edb *EventDb) IncrementDataStored(id string, stored int64) error {
	blobber, err := edb.GetBlobber(id)
	if err != nil {
		return err
	}
	update := dbs.DbUpdates{
		Id: id,
		Updates: map[string]interface{}{
			"used": blobber.Used + stored,
		},
	}
	return edb.updateBlobber(update)
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

func (edb *EventDb) blobberAggregateStats(id string) (*blobberAggregateStats, error) {
	var blobber blobberAggregateStats
	err := edb.Store.Get().Model(&Blobber{}).Where("blobber_id = ?", id).First(&blobber).Error
	if err != nil {
		return nil, fmt.Errorf("error retrieving blobber %v, error %v", id, err)
	}

	return &blobber, nil
}

func (edb *EventDb) TotalUsedData() (int64, error) {
	var total int64
	return total, edb.Store.Get().Model(&Blobber{}).
		Select("sum(used)").
		Find(&total).Error
}

func (edb *EventDb) GetBlobbers() ([]Blobber, error) {
	var blobbers []Blobber
	result := edb.Store.Get().Model(&Blobber{}).Find(&blobbers)

	return blobbers, result.Error
}

func (edb *EventDb) GetAllBlobberId() ([]string, error) {
	var blobberIDs []string
	result := edb.Store.Get().Model(&Blobber{}).Select("blobber_id").Find(&blobberIDs)

	return blobberIDs, result.Error
}

func (edb *EventDb) GeBlobberByLatLong(
	maxLatitude, minLatitude, maxLongitude, minLongitude float64,
) ([]string, error) {
	var blobberIDs []string
	result := edb.Store.Get().
		Model(&Blobber{}).
		Select("blobber_id").
		Where("latitude <= ? AND latitude >= ? AND longitude <= ? AND longitude >= ? ",
			maxLatitude, minLatitude, maxLongitude, minLongitude).
		Find(&blobberIDs)

	return blobberIDs, result.Error
}

func (edb *EventDb) GetBlobbersFromIDs(ids []string) ([]Blobber, error) {
	var blobbers []Blobber
	result := edb.Store.Get().Model(&Blobber{}).Order("id").Where("blobber_id IN ?", ids).Find(&blobbers)

	return blobbers, result.Error
}

func (edb *EventDb) deleteBlobber(id string) error {
	return edb.Store.Get().Model(&Blobber{}).Where("blobber_id = ?", id).Delete(&Blobber{}).Error
}

func (edb *EventDb) updateBlobber(updates dbs.DbUpdates) error {
	var blobber = Blobber{BlobberID: updates.Id}
	exists, err := blobber.exists(edb)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("blobber %v not in database cannot update", blobber.BlobberID)
	}

	return edb.Store.Get().Model(&Blobber{}).Where("blobber_id = ?", blobber.BlobberID).Updates(updates.Updates).Error
}

func (edb *EventDb) GetBlobberCount() (int64, error) {
	var count int64
	res := edb.Store.Get().Model(Blobber{}).Count(&count)

	return count, res.Error
}

type AllocationQuery struct {
	MaxChallengeCompletionTime time.Duration
	MaxOfferDuration           time.Duration
	ReadPriceRange             struct {
		Min int64
		Max int64
	}
	WritePriceRange struct {
		Min int64
		Max int64
	}
	Size              int
	AllocationSize    int64
	PreferredBlobbers []string
	NumberOfBlobbers  int
}

func (edb *EventDb) GetBlobberIdsFromUrls(urls []string) ([]string, error) {
	dbStore := edb.Store.Get().Model(&Blobber{})
	dbStore = dbStore.Where("base_url IN ?", urls)
	var blobberIDs []string
	return blobberIDs, dbStore.Select("blobber_id").Find(&blobberIDs).Error
}

func (edb *EventDb) GetBlobbersFromParams(allocation AllocationQuery) ([]string, error) {
	dbStore := edb.Store.Get().Model(&Blobber{})
	//dbStore = dbStore.Where("challenge_completion_time <= ?", allocation.MaxChallengeCompletionTime.Nanoseconds())
	dbStore = dbStore.Where("read_price between ? and ?", allocation.ReadPriceRange.Min, allocation.ReadPriceRange.Max)
	dbStore = dbStore.Where("write_price between ? and ?", allocation.WritePriceRange.Min, allocation.WritePriceRange.Max)
	dbStore = dbStore.Where("max_offer_duration >= ?", allocation.MaxOfferDuration.Nanoseconds())
	dbStore = dbStore.Where("capacity - allocated >= ?", allocation.AllocationSize)
	dbStore = dbStore.Where("last_health_check > ?", time.Now().Add(-time.Hour).Unix())
	dbStore = dbStore.Where("(total_stake - offers_total) > ?/write_price", allocation.AllocationSize/int64(allocation.NumberOfBlobbers))
	var blobberIDs []string
	return blobberIDs, dbStore.Select("blobber_id").Find(&blobberIDs).Error
}

func (edb *EventDb) overwriteBlobber(blobber Blobber) error {
	return edb.Store.Get().Model(&Blobber{}).Where("blobber_id = ?", blobber.BlobberID).
		Updates(map[string]interface{}{
			"base_url":                  blobber.BaseURL,
			"latitude":                  blobber.Latitude,
			"longitude":                 blobber.Longitude,
			"read_price":                blobber.ReadPrice,
			"write_price":               blobber.WritePrice,
			"min_lock_demand":           blobber.MinLockDemand,
			"max_offer_duration":        blobber.MaxOfferDuration,
			"challenge_completion_time": blobber.ChallengeCompletionTime,
			"capacity":                  blobber.Capacity,
			"allocated":                 blobber.Allocated,
			"last_health_check":         blobber.LastHealthCheck,
			"delegate_wallet":           blobber.DelegateWallet,
			"min_stake":                 blobber.MinStake,
			"max_stake":                 blobber.MaxStake,
			"num_delegates":             blobber.NumDelegates,
			"service_charge":            blobber.ServiceCharge,
			"offers_total":              blobber.OffersTotal,
			"unstake_total":             blobber.UnstakeTotal,
			"reward":                    blobber.Reward,
			"total_service_charge":      blobber.TotalServiceCharge,
			"saved_data":                blobber.SavedData,
			"name":                      blobber.Name,
			"website_url":               blobber.WebsiteUrl,
			"logo_url":                  blobber.LogoUrl,
			"description":               blobber.Description,
		}).Error
}

func (edb *EventDb) addOrOverwriteBlobber(blobber Blobber) error {
	exists, err := blobber.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return edb.overwriteBlobber(blobber)
	}

	return edb.Store.Get().Create(&blobber).Error
}

func (bl *Blobber) exists(edb *EventDb) (bool, error) {
	var blobber Blobber
	err := edb.Store.Get().Model(&Blobber{}).Where("blobber_id = ?", bl.BlobberID).Take(&blobber).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check Blobber existence %v, error %v", bl, err)
	}

	return true, nil
}
