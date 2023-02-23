package provider

import (
	"fmt"

	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
)

func ShutDown(
	input []byte,
	clientID, ownerId string,
	providerSpecific func(ProviderRequest) (Abstract, stakepool.AbstractStakePool, error),
	balances cstate.StateContextI,
) error {
	var req ProviderRequest
	if err := req.decode(input); err != nil {
		return err
	}

	p, sp, err := providerSpecific(req)
	if err != nil {
		return err
	}

	var errCode = "shutdown_" + p.Type().String() + "_failed"
	if err := smartcontractinterface.AuthorizeWithOwner(errCode, func() bool {
		return ownerId == clientID
	}); err != nil {
		return err
	}
	if p.IsShutDown() {
		return fmt.Errorf("already shutdown")
	}
	if p.IsKilled() {
		return fmt.Errorf("already killed")
	}

	p.ShutDown()
	if err := p.Save(balances); err != nil {
		return err
	}

	if err = sp.Save(p.Type(), req.ID, balances); err != nil {
		return err
	}

	// todo piers
	//if err := emitUpdateProvider(p, sp, balances); err != nil {
	//	return common.NewError(errCode, fmt.Sprintf("emitting event: %v", err))
	//}

	return nil
}
