package storagesc

import (
	"encoding/json"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract/dbs/event"
)

func writeMarkerToValidationNode(vn *ValidationNode) *event.Validator {
	return &event.Validator{
		ValidatorID: vn.ID,
		BaseUrl:     vn.BaseURL,
		// TO-DO: Update stake in eventDB
		Stake: 0,

		DelegateWallet: vn.StakePoolSettings.DelegateWallet,
		MinStake:       vn.StakePoolSettings.MinStake,
		MaxStake:       vn.StakePoolSettings.MaxStake,
		NumDelegates:   vn.StakePoolSettings.MaxNumDelegates,
		ServiceCharge:  vn.StakePoolSettings.ServiceCharge,
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
