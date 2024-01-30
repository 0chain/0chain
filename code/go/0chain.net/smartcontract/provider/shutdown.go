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
	providerSpecific func(ProviderRequest) (AbstractProvider, stakepool.AbstractStakePool, error),
	refreshProvider func(ProviderRequest) error,
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

	err = nil
	actErr := cstate.WithActivation(balances, "hard_fork_1", func() {
		if p.IsShutDown() {
			err = fmt.Errorf("already shutdown")
		}
		if p.IsKilled() {
			err = fmt.Errorf("already killed")
		}
	}, func() {
		if p.IsKilled() || p.IsShutDown() {
			if refreshProvider != nil {
				err = refreshProvider(req)
			}

			err = AlreadyShutdownError
		}
	})

	if actErr != nil {
		return actErr
	}
	if err != nil {
		return err
	}

	p.ShutDown()

	actErr = cstate.WithActivation(balances, "hard_fork_1", func() {
	}, func() {
		if killErr := sp.Kill(killSlash, p.Id(), p.Type(), balances); killErr != nil {
			err = fmt.Errorf("can't kill the stake pool: %v", killErr)
		}
	})

	if actErr != nil {
		return actErr
	}
	if err != nil {
		return err
	}

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
