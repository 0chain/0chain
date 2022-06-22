package storagesc

import (
	"encoding/json"
	"fmt"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs/event"
)

func writeMarkerToValidationNode(vn *ValidationNode) *event.Validator {
	return &event.Validator{
		ValidatorID: vn.ID,
		BaseUrl:     vn.BaseURL,
		PublicKey:   vn.PublicKey,
		// TO-DO: Update stake in eventDB
		Stake: 0,

		DelegateWallet: vn.StakePoolSettings.DelegateWallet,
		MinStake:       vn.StakePoolSettings.MinStake,
		MaxStake:       vn.StakePoolSettings.MaxStake,
		NumDelegates:   vn.StakePoolSettings.MaxNumDelegates,
		ServiceCharge:  vn.StakePoolSettings.ServiceChargeRatio,
	}
}

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
	data, err := json.Marshal(&dbs.DbUpdates{
		Id: vn.ID,
		Updates: map[string]interface{}{
			"base_url":        vn.BaseURL,
			"delegate_wallet": vn.StakePoolSettings.DelegateWallet,
			"min_stake":       vn.StakePoolSettings.MinStake,
			"max_stake":       vn.StakePoolSettings.MaxStake,
			"num_delegates":   vn.StakePoolSettings.MaxNumDelegates,
			"service_charge":  vn.StakePoolSettings.ServiceChargeRatio,
		},
	})
	if err != nil {
		return fmt.Errorf("marshalling update: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, vn.ID, string(data))
	return nil
}

func (vn *ValidationNode) emitAdd(balances cstate.StateContextI) error {
	data, err := json.Marshal(&event.Validator{
		ValidatorID:    vn.ID,
		BaseUrl:        vn.BaseURL,
		DelegateWallet: vn.StakePoolSettings.DelegateWallet,
		MinStake:       vn.StakePoolSettings.MinStake,
		MaxStake:       vn.StakePoolSettings.MaxStake,
		NumDelegates:   vn.StakePoolSettings.MaxNumDelegates,
		ServiceCharge:  vn.StakePoolSettings.ServiceChargeRatio,
	})
	if err != nil {
		return fmt.Errorf("marshalling validator: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagAddValidator, vn.ID, string(data))
	return nil
}
