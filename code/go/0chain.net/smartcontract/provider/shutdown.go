package provider

import (
	"fmt"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"

	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
)

var AlreadyShutdownError = fmt.Errorf("already killed or shutdown")

func ShutDown(
	input []byte,
	clientId, ownerId string,
	killSlash float64,
	providerSpecific func(ProviderRequest) (AbstractProvider, string, stakepool.AbstractStakePool, error),
	refreshProvider func(ProviderRequest) error,
	balances cstate.StateContextI,
) error {
	var req ProviderRequest
	if err := req.Decode(input); err != nil {
		return err
	}

	p, authWallet, sp, err := providerSpecific(req)
	if err != nil {
		return err
	}

	if p.IsKilled() || p.IsShutDown() {
		if refreshProvider != nil {
			err = refreshProvider(req)
			if err != nil {
				return err
			}
		}

		return AlreadyShutdownError
	}

	p.ShutDown()

	if err = sp.Kill(killSlash, p.Id(), p.Type(), balances); err != nil {
		return fmt.Errorf("can't kill the stake pool: %v", err)
	}

	if jasonActErr := cstate.WithActivation(balances, "jason", func() error {
		return sp.Save(p.Type(), clientId, balances)
	}, func() error {
		return sp.Save(p.Type(), req.ID, balances)
	}); jasonActErr != nil {
		return jasonActErr
	}

	var errCode = "shutdown_" + p.Type().String() + "_failed"
	if err := smartcontractinterface.AuthorizeWithOwner(errCode, func() bool {
		return ownerId == clientId || clientId == authWallet
	}); err != nil {
		return err
	}

	balances.EmitEvent(event.TypeStats, event.TagShutdownProvider, p.Id(), dbs.ProviderID{
		ID:   p.Id(),
		Type: p.Type(),
	})

	return nil
}
