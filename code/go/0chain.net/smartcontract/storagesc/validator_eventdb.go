package storagesc

import (
	"0chain.net/core/common"
	"0chain.net/smartcontract/provider"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs/event"
)

func validatorTableToValidationNode(v event.Validator) *ValidationNode {
	return &ValidationNode{
		Provider: &provider.Provider{
			LastHealthCheck: common.Timestamp(v.LastHealthCheck),
			HasBeenKilled:   v.IsKilled,
			HasBeenShutDown: v.IsShutDown,
		},
		ID:        v.ValidatorID,
		BaseURL:   v.BaseUrl,
		PublicKey: v.PublicKey,
		StakePoolSettings: stakepool.Settings{
			DelegateWallet:     v.DelegateWallet,
			MinStake:           v.MinStake,
			MaxStake:           v.MaxStake,
			MaxNumDelegates:    v.NumDelegates,
			ServiceChargeRatio: v.ServiceCharge,
		},
	}
}

func getValidators(validatorIDs []string, edb *event.EventDb) ([]*ValidationNode, error) {
	validators, err := edb.GetValidatorsByIDs(validatorIDs)
	if err != nil {
		return nil, err
	}
	vNodes := make([]*ValidationNode, len(validators))
	for i := range validators {
		vNodes[i] = validatorTableToValidationNode(validators[i])
	}

	return vNodes, nil
}

func (vn *ValidationNode) emitUpdate(balances cstate.StateContextI) error {
	data := &dbs.DbUpdates{
		Id: vn.ID,
		Updates: map[string]interface{}{
			"base_url":          vn.BaseURL,
			"delegate_wallet":   vn.StakePoolSettings.DelegateWallet,
			"min_stake":         vn.StakePoolSettings.MinStake,
			"max_stake":         vn.StakePoolSettings.MaxStake,
			"num_delegates":     vn.StakePoolSettings.MaxNumDelegates,
			"service_charge":    vn.StakePoolSettings.ServiceChargeRatio,
			"last_health_check": int64(vn.LastHealthCheck),
			"is_killed":         vn.IsKilled,
			"is_shut_down":      vn.IsShutDown,
		},
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateValidator, vn.ID, data)
	return nil
}

func (vn *ValidationNode) emitAdd(balances cstate.StateContextI) error {
	data := &event.Validator{
		ValidatorID:     vn.ID,
		BaseUrl:         vn.BaseURL,
		DelegateWallet:  vn.StakePoolSettings.DelegateWallet,
		MinStake:        vn.StakePoolSettings.MinStake,
		MaxStake:        vn.StakePoolSettings.MaxStake,
		NumDelegates:    vn.StakePoolSettings.MaxNumDelegates,
		ServiceCharge:   vn.StakePoolSettings.ServiceChargeRatio,
		LastHealthCheck: int64(vn.LastHealthCheck),
		IsShutDown:      vn.IsShutDown(),
		IsKilled:        vn.IsKilled(),
	}

	balances.EmitEvent(event.TypeStats, event.TagAddValidator, vn.ID, data)
	return nil
}
