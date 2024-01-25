package provider

import (
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
	"encoding/json"
	"fmt"

	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
)

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

	if p.IsKilled() {
		if refreshProvider != nil {
			if err := refreshProvider(req); err != nil {
				return err
			}
		}

		return fmt.Errorf("already killed")
	}

	p.Kill()

	if clientID == sp.GetSettings().DelegateWallet {
		killSlash /= 2 // Penalise 50% only in case of provider is shutting down by delegate wallet
	}

	if err := sp.Kill(killSlash, p.Id(), p.Type(), balances); err != nil {
		return err
	}

	if err = sp.Save(p.Type(), req.ID, balances); err != nil {
		return err
	}

	if refreshProvider != nil {
		if err := refreshProvider(req); err != nil {
			return err
		}
	}

	balances.EmitEvent(event.TypeStats, event.TagKillProvider, p.Id(), dbs.ProviderID{
		ID:   p.Id(),
		Type: p.Type(),
	})

	return nil
}
