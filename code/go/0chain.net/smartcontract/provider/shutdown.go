package provider

import (
	"fmt"

	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
)

func ShutDown(
	id string,
	providerSpecific func() (Abstract, stakepool.AbstractStakePool, error),
	balances cstate.StateContextI,
) error {
	p, sp, err := providerSpecific()
	if err != nil {
		return err
	}

	if p.IsShutDown() {
		return fmt.Errorf("already shutdown")
	}
	if p.IsKilled() {
		return fmt.Errorf("already killed")
	}

	p.ShutDown()

	if err = sp.Save(p.Type(), id, balances); err != nil {
		return err
	}

	// todo piers
	//if err := emitUpdateProvider(p, sp, balances); err != nil {
	//	return common.NewError(errCode, fmt.Sprintf("emitting event: %v", err))
	//}

	return nil
}
