package event

import (
	"0chain.net/smartcontract/dbs"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type Blobber struct {
	gorm.Model `json:"gorm_._model"`
	BlobberID  string `json:"id" gorm:"uniqueIndex" json:"blobber_id,omitempty"`
	BaseURL    string `json:"url" json:"base_url,omitempty"`

	//provider
	LastHealthCheck int64 `json:"last_health_check" json:"last_health_check,omitempty"`
	IsKilled        bool  `json:"is_killed,omitempty"`
	IsShutDown      bool  `json:"is_shut_down,omitempty"`

	// geolocation
	Latitude  float64 `json:"latitude" json:"latitude,omitempty"`
	Longitude float64 `json:"longitude" json:"longitude,omitempty"`

	// terms
	ReadPrice               int64   `json:"read_price" json:"read_price,omitempty"`
	WritePrice              int64   `json:"write_price" json:"write_price,omitempty"`
	MinLockDemand           float64 `json:"min_lock_demand" json:"min_lock_demand,omitempty"`
	MaxOfferDuration        string  `json:"max_offer_duration" json:"max_offer_duration,omitempty"`
	ChallengeCompletionTime string  `json:"challenge_completion_time" json:"challenge_completion_time,omitempty"`

	Capacity        int64 `json:"capacity" json:"capacity,omitempty"`                   // total blobber capacity
	Used            int64 `json:"used" json:"used,omitempty"`                           // allocated capacity
	TotalDataStored int64 `json:"total_data_stored" json:"total_data_stored,omitempty"` // total of files saved on blobber

	SavedData int64 `json:"saved_data" json:"saved_data,omitempty"`

	// stake_pool_settings
	DelegateWallet string  `json:"delegate_wallet" json:"delegate_wallet,omitempty"`
	MinStake       int64   `json:"min_stake" json:"min_stake,omitempty"`
	MaxStake       int64   `json:"max_stake" json:"max_stake,omitempty"`
	NumDelegates   int     `json:"num_delegates" json:"num_delegates,omitempty"`
	ServiceCharge  float64 `json:"service_charge" json:"service_charge,omitempty"`

	OffersTotal        int64 `json:"offers_total" json:"offers_total,omitempty"`
	UnstakeTotal       int64 `json:"unstake_total" json:"unstake_total,omitempty"`
	Reward             int64 `json:"reward" json:"reward,omitempty"`
	TotalServiceCharge int64 `json:"total_service_charge" json:"total_service_charge,omitempty"`
	TotalStake         int64 `json:"total_stake" json:"total_stake,omitempty"`

	Name        string `json:"name" gorm:"name" json:"name,omitempty"`
	WebsiteUrl  string `json:"website_url" gorm:"website_url" json:"website_url,omitempty"`
	LogoUrl     string `json:"logo_url" gorm:"logo_url" json:"logo_url,omitempty"`
	Description string `json:"description" gorm:"description" json:"description,omitempty"`

	WriteMarkers []WriteMarker `gorm:"foreignKey:BlobberID;references:BlobberID" json:"write_markers,omitempty"`
	ReadMarkers  []ReadMarker  `gorm:"foreignKey:BlobberID;references:BlobberID" json:"read_markers,omitempty"`
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

func (edb *EventDb) GetBlobber(id string) (*Blobber, error) {
	var blobber Blobber
	err := edb.Store.Get().Model(&Blobber{}).Where("blobber_id = ?", id).First(&blobber).Error
	if err != nil {
		return nil, fmt.Errorf("error retrieving blobber %v, error %v", id, err)
	}

	return &blobber, nil
}

func (edb *EventDb) blobberAggregateStats(id string) (*blobberAggregateStats, error) {
	var blobber blobberAggregateStats
	err := edb.Store.Get().Model(&Blobber{}).Where("blobber_id = ?", id).First(&blobber).Error
	if err != nil {
		return nil, fmt.Errorf("error retrieving blobber %v, error %v", id, err)
	}

	return &blobber, nil
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

func (edb *EventDb) GetAllBlobberLatLong() ([]BlobberLatLong, error) {
	var blobbers []BlobberLatLong
	result := edb.Store.Get().Model(&Blobber{}).Find(&blobbers)

	return blobbers, result.Error
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
			"used":                      blobber.Used,
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
