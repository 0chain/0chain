package storagesc

import (
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs/event"
)

func validatorTableToValidationNode(v event.Validator) *ValidationNode {
	return &ValidationNode{
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
			"base_url":        vn.BaseURL,
			"delegate_wallet": vn.StakePoolSettings.DelegateWallet,
			"min_stake":       vn.StakePoolSettings.MinStake,
			"max_stake":       vn.StakePoolSettings.MaxStake,
			"num_delegates":   vn.StakePoolSettings.MaxNumDelegates,
			"service_charge":  vn.StakePoolSettings.ServiceChargeRatio,
		},
	}

	balances.EmitEvent(event.TypeSmartContract, event.TagUpdateValidator, vn.ID, data)
	return nil
}

func (vn *ValidationNode) emitAddOrOverwrite(balances cstate.StateContextI) error {
	data := &event.Validator{
		ValidatorID:    vn.ID,
		BaseUrl:        vn.BaseURL,
		DelegateWallet: vn.StakePoolSettings.DelegateWallet,
		MinStake:       vn.StakePoolSettings.MinStake,
		MaxStake:       vn.StakePoolSettings.MaxStake,
		NumDelegates:   vn.StakePoolSettings.MaxNumDelegates,
		ServiceCharge:  vn.StakePoolSettings.ServiceChargeRatio,
		Rewards:        event.ProviderRewards{ProviderID: vn.ID},
	}

	balances.EmitEvent(event.TypeSmartContract, event.TagAddOrOverwiteValidator, vn.ID, data)
	return nil
}
