// Experimental
// todo: naming? balance(s)?

package storagesc

import (
	"errors"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/util"
)

// todo: naming?
func checkFill(t *transaction.Transaction, balances cstate.StateContextI) (err error) {
	if t.Value < 0 {
		return errors.New("negative transaction value")
	}

	var balance state.Balance
	balance, err = balances.GetClientBalance(t.ClientID)

	if err != nil && err != util.ErrValueNotPresent {
		return
	}

	if err == util.ErrValueNotPresent {
		return errors.New("no tokens to lock")
	}

	if state.Balance(t.Value) > balance {
		return errors.New("lock amount is greater than balance")
	}

	return
}
