package event

import (
	"gorm.io/gorm"
)

type Model interface {
	Create(interface{}) (int64, error)
	Get(string) (interface{}, error)
	Delete(string) error
}

type Authorizer struct {
	gorm.Model

	AuthorizerID string `json:"id" gorm:"uniqueIndex"`
	BaseURL      string `json:"url"`

	// Geolocation
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`

	// Stats
	LastHealthCheck int64 `json:"last_health_check"`

	// stake_pool_settings
	DelegateWallet string  `json:"delegate_wallet"`
	MinStake       int64   `json:"min_stake"`
	MaxStake       int64   `json:"max_stake"`
	NumDelegates   int     `json:"num_delegates"`
	ServiceCharge  float64 `json:"service_charge"`
}

func (edb *EventDb) AddAuthorizer(a *Authorizer) error {
	//TODO implement me
	panic("implement me")
}

func (edb *EventDb) GetAuthorizer(id string) (interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (edb *EventDb) DeleteAuthorizer(id string) error {
	//TODO implement me
	panic("implement me")
}
