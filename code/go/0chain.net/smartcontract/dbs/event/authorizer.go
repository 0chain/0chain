package event

import (
	"errors"
	"fmt"
	"time"

	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"github.com/0chain/common/core/currency"
)

const ActiveAuthorizerTimeLimit = 5 * time.Minute // 5 Minutes

type Authorizer struct {
	Provider

	URL string `json:"url"`

	// Configuration
	Fee currency.Coin `json:"fee"`

	// Geolocation
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`

	TotalMint currency.Coin `json:"total_mint"`
	TotalBurn currency.Coin `json:"total_burn"`

	CreationRound int64 `json:"creation_round" gorm:"index:idx_authorizer_creation_round"`
}

func (a *Authorizer) GetTotalStake() currency.Coin {
	return a.TotalStake
}

func (a *Authorizer) GetUnstakeTotal() currency.Coin {
	return a.UnstakeTotal
}

func (a *Authorizer) GetServiceCharge() float64 {
	return a.ServiceCharge
}

func (a *Authorizer) GetTotalRewards() currency.Coin {
	return a.Rewards.TotalRewards
}

func (a *Authorizer) SetTotalStake(value currency.Coin) {
	a.TotalStake = value
}

func (a *Authorizer) SetUnstakeTotal(value currency.Coin) {
	a.UnstakeTotal = value
}

func (a *Authorizer) SetServiceCharge(value float64) {
	a.ServiceCharge = value
}

func (a *Authorizer) SetTotalRewards(value currency.Coin) {
	a.Rewards.TotalRewards = value
}

func (edb *EventDb) AddAuthorizer(a *Authorizer) error {
	exists, err := a.exists(edb)
	if err != nil {
		return err
	}

	if exists {
		return errors.New("authorizer already exists")
	}

	result := edb.Store.Get().Create(a)

	return result.Error
}

func (edb *EventDb) GetAuthorizerCount() (int64, error) {
	var count int64
	res := edb.Store.Get().Model(Authorizer{}).Count(&count)

	return count, res.Error
}

func (edb *EventDb) GetAuthorizer(id string) (*Authorizer, error) {
	var auth Authorizer

	result := edb.Store.Get().
		Model(&Authorizer{}).
		Where(&Authorizer{Provider: Provider{ID: id}}).
		First(&auth)

	if result.Error != nil {
		return nil, fmt.Errorf(
			"error retrieving authorizer %v, error %v",
			id, result.Error,
		)
	}

	return &auth, nil
}

func (edb *EventDb) GetActiveAuthorizers() ([]Authorizer, error) {
	now := common.Now()
	var authorizers []Authorizer
	result := edb.Store.Get().
		Model(&Authorizer{}).
		Where("last_health_check > ?", common.ToTime(now).Add(-ActiveAuthorizerTimeLimit).Unix()).
		Find(&authorizers)
	return authorizers, result.Error
}

func (edb *EventDb) GetAuthorizers() ([]Authorizer, error) {
	var authorizers []Authorizer
	result := edb.Store.Get().
		Model(&Authorizer{}).
		Find(&authorizers)
	return authorizers, result.Error
}

func (edb *EventDb) DeleteAuthorizer(id string) error {
	result := edb.Store.Get().
		Where("authorizer_id = ?", id).
		Delete(&Authorizer{})
	return result.Error
}

func (a *Authorizer) exists(edb *EventDb) (bool, error) {
	var count int64

	result := edb.Get().
		Model(&Authorizer{}).
		Where(&Authorizer{Provider: Provider{ID: a.ID}}).
		Count(&count)

	if result.Error != nil {
		return false,
			fmt.Errorf(
				"error searching for authorizer %v, error %v",
				a.ID, result.Error,
			)
	}
	return count > 0, nil
}

func NewUpdateAuthorizerTotalStakeEvent(ID string, totalStake currency.Coin) (tag EventTag, data interface{}) {
	return TagUpdateAuthorizerTotalStake, Authorizer{
		Provider: Provider{
			ID:         ID,
			TotalStake: totalStake,
		},
	}
}

func NewUpdateAuthorizerTotalUnStakeEvent(ID string, totalUnstake currency.Coin) (tag EventTag, data interface{}) {
	return TagUpdateAuthorizerTotalStake, Authorizer{
		Provider: Provider{
			ID:         ID,
			UnstakeTotal: totalUnstake,
		},
	}
}

func (edb *EventDb) updateAuthorizersTotalStakes(authorizer []Authorizer) error {
	var provs []Provider
	for _, a := range authorizer {
		provs = append(provs, a.Provider)
	}
	return edb.updateProviderTotalStakes(provs, "authorizers")
}

func (edb *EventDb) updateAuthorizersTotalMint(mints []state.Mint) error {
	var (
		ids []string
		totalMint []int64
	)
	for _, m := range mints {
		ids = append(ids, m.ToClientID)
		amt, err := m.Amount.Int64()
		if err != nil {
			return err
		}
		totalMint = append(totalMint, amt)
	}

	return CreateBuilder("authorizers", "id", ids).
		AddUpdate("total_mint", totalMint, "authorizers.total_mint + t.total_mint").
		Exec(edb).Debug().Error
}

func (edb *EventDb) updateAuthorizersTotalBurn(burns []state.Burn) error {
	var (
		ids []string
		totalBurns []int64
	)
	for _, m := range burns {
		ids = append(ids, m.Burner)
		amt, err := m.Amount.Int64()
		if err != nil {
			return err
		}
		totalBurns = append(totalBurns, amt)
	}

	return CreateBuilder("authorizers", "id", ids).
		AddUpdate("total_burn", totalBurns, "authorizers.total_burn + t.total_burn").
		Exec(edb).Debug().Error
}

func (edb *EventDb) updateAuthorizersTotalUnStakes(authorizer []Authorizer) error {
	var provs []Provider
	for _, a := range authorizer {
		provs = append(provs, a.Provider)
	}
	return edb.updateProvidersTotalUnStakes(provs, "authorizers")
}

func mergeUpdateAuthorizerTotalStakesEvents() *eventsMergerImpl[Authorizer] {
	return newEventsMerger[Authorizer](TagUpdateAuthorizerTotalStake, withUniqueEventOverwrite())
}

func mergeUpdateAuthorizerTotalUnStakesEvents() *eventsMergerImpl[Authorizer] {
	return newEventsMerger[Authorizer](TagUpdateAuthorizerTotalUnStake, withUniqueEventOverwrite())
}

func mergeAuthorizerHealthCheckEvents() *eventsMergerImpl[dbs.DbHealthCheck] {
	return newEventsMerger[dbs.DbHealthCheck](TagAuthorizerHealthCheck, withUniqueEventOverwrite())
}

func mergeAuthorizerBurnEvents() *eventsMergerImpl[state.Burn] {
	return newEventsMerger[state.Burn](TagAuthorizerBurn, withUniqueEventOverwrite())
}
