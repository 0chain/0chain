package provider

import (
	"fmt"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"

	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
)

func ShutDown(
	input []byte,
	clientId, ownerId string,
	providerSpecific func(ProviderRequest) (AbstractProvider, stakepool.AbstractStakePool, error),
	balances cstate.StateContextI,
) error {
	var req ProviderRequest
	if err := req.Decode(input); err != nil {
		return err
	}

	p, sp, err := providerSpecific(req)
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

	if err = sp.Save(p.Type(), clientId, balances); err != nil {
		return err
	}

	var errCode = "shutdown_" + p.Type().String() + "_failed"
	if err := smartcontractinterface.AuthorizeWithOwner(errCode, func() bool {
		return ownerId == clientId || clientId == sp.GetSettings().DelegateWallet
	}); err != nil {
		return err
	}

	balances.EmitEvent(event.TypeStats, event.TagShutdownProvider, p.Id(), dbs.ProviderID{
		ID:   p.Id(),
		Type: p.Type(),
	})

	return nil
}
