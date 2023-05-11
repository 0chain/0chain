package provider

import (
	"fmt"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"

	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
)

func ShutDown(
	id string,
	providerSpecific func() (AbstractProvider, stakepool.AbstractStakePool, error),
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

	if id != sp.GetSettings().DelegateWallet {
		return fmt.Errorf("access denied, allowed for delegate_wallet owner only")
	}

	balances.EmitEvent(event.TypeStats, event.TagShutdownProvider, p.Id(), dbs.ProviderID{
		ID:   p.Id(),
		Type: p.Type(),
	})

	return nil
}
