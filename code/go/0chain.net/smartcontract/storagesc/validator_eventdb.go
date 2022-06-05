package storagesc

import (
	"encoding/json"
	"fmt"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
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

func emitAddOrOverwriteValidatorTable(vn *ValidationNode, balances cstate.StateContextI, t *transaction.Transaction) error {

	data, err := json.Marshal(writeMarkerToValidationNode(vn))
	if err != nil {
		return fmt.Errorf("failed to marshal writemarker: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagAddOrOverwriteValidator, t.Hash, string(data))

	return nil
}

func emitUpdateValidator(sn *ValidationNode, balances cstate.StateContextI) error {
	data, err := json.Marshal(&dbs.DbUpdates{
		Id: sn.ID,
		Updates: map[string]interface{}{
			"base_url":        sn.BaseURL,
			"delegate_wallet": sn.StakePoolSettings.DelegateWallet,
			"min_stake":       int64(sn.StakePoolSettings.MinStake),
			"max_stake":       int64(sn.StakePoolSettings.MaxStake),
			"num_delegates":   sn.StakePoolSettings.MaxNumDelegates,
			"service_charge":  sn.StakePoolSettings.ServiceChargeRatio,
		},
	})
	if err != nil {
		return fmt.Errorf("marshalling update: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagUpdateValidator, sn.ID, string(data))
	return nil
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
