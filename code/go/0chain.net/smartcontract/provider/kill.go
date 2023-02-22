package provider

import (
	"encoding/json"
	"fmt"

	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
)

type ProviderRequest struct {
	ID string `json:"id"`
}

func (pr *ProviderRequest) decode(p []byte) error {
	return json.Unmarshal(p, pr)
}

func Kill(
	input []byte,
	clientID, ownerId string,
	killSlash float64,
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

	var errCode = "kill_" + p.Type().String() + "_failed"
	if err := smartcontractinterface.AuthorizeWithOwner(errCode, func() bool {
		return ownerId == clientID
	}); err != nil {
		return err
	}

	if p.IsKilled() {
		return fmt.Errorf("already killed")
	}
	p.Kill()
	if err := p.Save(balances); err != nil {
		return err
	}

	sp.Kill()
	if err := sp.SlashFraction(
		killSlash,
		req.ID,
		p.Type(),
		balances,
	); err != nil {
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
