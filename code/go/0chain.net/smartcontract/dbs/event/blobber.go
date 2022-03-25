package event

import (
	"errors"
	"fmt"

	"0chain.net/smartcontract/dbs"

	"gorm.io/gorm"
)

type Blobber struct {
	gorm.Model
	BlobberID string `json:"id" gorm:"uniqueIndex"`
	BaseURL   string `json:"url"`

	// geolocation
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`

	// terms
	ReadPrice               int64   `json:"read_price"`
	WritePrice              int64   `json:"write_price"`
	MinLockDemand           float64 `json:"min_lock_demand"`
	MaxOfferDuration        string  `json:"max_offer_duration"`
	ChallengeCompletionTime string  `json:"challenge_completion_time"`

	Capacity        int64 `json:"capacity"`          // total blobber capacity
	Used            int64 `json:"used"`              // allocated capacity
	TotalDataStored int64 `json:"total_data_stored"` // total of files saved on blobber
	LastHealthCheck int64 `json:"last_health_check"`
	SavedData       int64 `json:"saved_data"`

	// stake_pool_settings
	DelegateWallet string  `json:"delegate_wallet"`
	MinStake       int64   `json:"min_stake"`
	MaxStake       int64   `json:"max_stake"`
	NumDelegates   int     `json:"num_delegates"`
	ServiceCharge  float64 `json:"service_charge"`

	OffersTotal        int64 `json:"offers_total"`
	UnstakeTotal       int64 `json:"unstake_total"`
	Reward             int64 `json:"reward"`
	TotalServiceCharge int64 `json:"total_service_charge"`
	TotalStake         int64 `json:"total_stake"`

	WriteMarkers []WriteMarker `gorm:"foreignKey:BlobberID;references:BlobberID"`
	ReadMarkers  []ReadMarker  `gorm:"foreignKey:BlobberID;references:BlobberID"`
}

type BlobberLatLong struct {
	// geolocation
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type blobberAggregateStats struct {
	Reward             int64 `json:"reward"`
	TotalServiceCharge int64 `json:"total_service_charge"`
}

type UpdateTotalStake struct {
	BlobberID  string
	TotalStake int64
}

func (edb *EventDb) updateBlobberTotalStake(uts UpdateTotalStake) error {
	return edb.Store.Get().Model(&Blobber{}).Where(&Blobber{BlobberID: uts.BlobberID}).
		Updates(map[string]interface{}{
			"total_stake": uts.TotalStake,
		}).Error
}

func (edb *EventDb) GetBlobber(id string) (*Blobber, error) {
	var blobber Blobber
	result := edb.Store.Get().
		Model(&Blobber{}).
		Where(&Blobber{BlobberID: id}).
		First(&blobber)
	if result.Error != nil {
		return nil, fmt.Errorf("error retrieving blobber %v, error %v",
			id, result.Error)
	}

	return &blobber, nil
}

func (edb *EventDb) blobberAggregateStats(id string) (*blobberAggregateStats, error) {
	var blobber blobberAggregateStats
	result := edb.Store.Get().
		Model(&Blobber{}).
		Where(&Blobber{BlobberID: id}).
		First(&blobber)
	if result.Error != nil {
		return nil, fmt.Errorf("error retrieving blobber %v, error %v",
			id, result.Error)
	}

	return &blobber, nil
}

func (edb *EventDb) GetBlobbers() ([]Blobber, error) {
	var blobbers []Blobber
	result := edb.Store.Get().
		Model(&Blobber{}).
		Find(&blobbers)
	return blobbers, result.Error
}

func (edb *EventDb) GetAllBlobberId() ([]string, error) {
	blobberIDs := []string{}
	result := edb.Store.Get().Model(&Blobber{}).Select("blobber_id").Find(&blobberIDs)
	return blobberIDs, result.Error
}

func (edb *EventDb) GetAllBlobberLatLong() ([]BlobberLatLong, error) {
	blobbers := []BlobberLatLong{}
	result := edb.Store.Get().Model(&Blobber{}).Find(&blobbers)
	return blobbers, result.Error
}

func (edb *EventDb) deleteBlobber(id string) error {
	result := edb.Store.Get().
		Where(&Blobber{BlobberID: id}).
		Delete(&Blobber{})
	return result.Error
}

func (edb *EventDb) updateBlobber(updates dbs.DbUpdates) error {
	var blobber = Blobber{BlobberID: updates.Id}
	exists, err := blobber.exists(edb)

	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("blobber %v not in database cannot update",
			blobber.BlobberID)
	}

	result := edb.Store.Get().
		Model(&Blobber{}).
		Where(&Blobber{BlobberID: blobber.BlobberID}).
		Updates(updates.Updates)
	return result.Error
}

func (edb *EventDb) GetBlobberCount() (int64, error) {
	var count int64
	res := edb.Store.Get().Model(Blobber{}).Count(&count)
	return count, res.Error
}

func (edb *EventDb) overwriteBlobber(blobber Blobber) error {
	result := edb.Store.Get().
		Model(&Blobber{}).
		Where(&Blobber{BlobberID: blobber.BlobberID}).
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
			"used":                      blobber.Used,
			"last_health_check":         blobber.LastHealthCheck,
			"delegate_wallet":           blobber.DelegateWallet,
			"min_stake":                 blobber.MaxStake,
			"max_stake":                 blobber.MaxStake,
			"num_delegates":             blobber.NumDelegates,
			"service_charge":            blobber.ServiceCharge,
			"offers_total":              blobber.OffersTotal,
			"unstake_total":             blobber.UnstakeTotal,
			"reward":                    blobber.Reward,
			"total_service_charge":      blobber.TotalServiceCharge,
			"saved_data":                blobber.SavedData,
			"total_stake":               blobber.TotalStake,
		})
	return result.Error
}

func (edb *EventDb) addOrOverwriteBlobber(blobber Blobber) error {
	exists, err := blobber.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return edb.overwriteBlobber(blobber)
	}

	result := edb.Store.Get().Create(&blobber)
	return result.Error
}

func (bl *Blobber) exists(edb *EventDb) (bool, error) {
	var blobber Blobber
	result := edb.Store.Get().Model(&Blobber{}).Where(&Blobber{BlobberID: bl.BlobberID}).Take(&blobber)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if result.Error != nil {
		return false, fmt.Errorf("failed to check Blobber existence %v, error %v",
			bl, result.Error)
	}
	return true, nil
}
