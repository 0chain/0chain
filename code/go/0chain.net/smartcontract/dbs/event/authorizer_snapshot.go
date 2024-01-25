package event

import (
	"github.com/0chain/common/core/currency"
	"gorm.io/gorm/clause"
)

// swagger:model AuthorizerSnapshot
type AuthorizerSnapshot struct {
	AuthorizerID string `json:"id" gorm:"uniquIndex"`
	Round        int64  `json:"round"`

	Fee           currency.Coin `json:"fee"`
	TotalStake    currency.Coin `json:"total_stake"`
	TotalRewards  currency.Coin `json:"total_rewards"`
	TotalMint     currency.Coin `json:"total_mint"`
	TotalBurn     currency.Coin `json:"total_burn"`
	ServiceCharge float64       `json:"service_charge"`
	CreationRound int64         `json:"creation_round"`
	IsKilled      bool          `json:"is_killed"`
	IsShutdown    bool          `json:"is_shutdown"`
}

func (as *AuthorizerSnapshot) GetID() string {
	return as.AuthorizerID
}

func (as *AuthorizerSnapshot) GetRound() int64 {
	return as.Round
}

func (as *AuthorizerSnapshot) SetID(id string) {
	as.AuthorizerID = id
}

func (as *AuthorizerSnapshot) SetRound(round int64) {
	as.Round = round
}

func (a *AuthorizerSnapshot) IsOffline() bool {
	return a.IsKilled || a.IsShutdown
}

func (a *AuthorizerSnapshot) GetTotalStake() currency.Coin {
	return a.TotalStake
}

func (a *AuthorizerSnapshot) GetServiceCharge() float64 {
	return a.ServiceCharge
}

func (a *AuthorizerSnapshot) GetTotalRewards() currency.Coin {
	return a.TotalRewards
}

func (a *AuthorizerSnapshot) SetTotalStake(value currency.Coin) {
	a.TotalStake = value
}

func (a *AuthorizerSnapshot) SetServiceCharge(value float64) {
	a.ServiceCharge = value
}

func (a *AuthorizerSnapshot) SetTotalRewards(value currency.Coin) {
	a.TotalRewards = value
}

func (edb *EventDb) addAuthorizerSnapshot(authorizers []*Authorizer, round int64) error {
	var snapshots []*AuthorizerSnapshot
	for _, authorizer := range authorizers {
		snapshots = append(snapshots, createAuthorizerSnapshotFromAuthorizer(authorizer, round))
	}

	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "authorizer_id"}},
		UpdateAll: true,
	}).Create(&snapshots).Error
}

func createAuthorizerSnapshotFromAuthorizer(authorizer *Authorizer, round int64) *AuthorizerSnapshot {
	return &AuthorizerSnapshot{
		AuthorizerID:  authorizer.ID,
		Round:         round,
		Fee:           authorizer.Fee,
		TotalStake:    authorizer.TotalStake,
		ServiceCharge: authorizer.ServiceCharge,
		CreationRound: authorizer.CreationRound,
		TotalRewards:  authorizer.Rewards.TotalRewards,
		TotalMint:     authorizer.TotalMint,
		TotalBurn:     authorizer.TotalBurn,
		IsKilled:      authorizer.IsKilled,
		IsShutdown:    authorizer.IsShutdown,
	}
}