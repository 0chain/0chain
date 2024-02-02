package provider

import (
	"encoding/json"
	"fmt"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"

	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
)

var AlreadyKilledError = fmt.Errorf("already killed or shutdown")

type ProviderRequest struct {
	ID string `json:"provider_id"`
}

func (pr *ProviderRequest) Encode() []byte {
	b, _ := json.Marshal(pr)
	return b
}

func (pr *ProviderRequest) Decode(p []byte) error {
	return json.Unmarshal(p, pr)
}

func Kill(
	input []byte,
	clientID, ownerId string,
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

	var errCode = "kill_" + p.Type().String() + "_failed"
	if err := smartcontractinterface.AuthorizeWithOwner(errCode, func() bool {
		return ownerId == clientID
	}); err != nil {
		return err
	}

	actErr := cstate.WithActivation(balances, "apollo", func() (e error) {
		if p.IsShutDown() {
			return fmt.Errorf("already shutdown")
		}
		if p.IsKilled() {
			return fmt.Errorf("already killed")
		}
		return nil
	}, func() (e error) {
		if p.IsKilled() || p.IsShutDown() {
			if refreshProvider != nil {
				e = refreshProvider(req)
				if e != nil {
					return e
				}
			}
			e = AlreadyKilledError
		}
		return e
	})
	if actErr != nil {
		return actErr
	}

	p.Kill()

	if err := sp.Kill(killSlash, p.Id(), p.Type(), balances); err != nil {
		return err
	}

	if err = sp.Save(p.Type(), req.ID, balances); err != nil {
		return err
	}

	balances.EmitEvent(event.TypeStats, event.TagKillProvider, p.Id(), dbs.ProviderID{
		ID:   p.Id(),
		Type: p.Type(),
	})

	return nil
}
