package provider

import (
	"encoding/json"
	"fmt"

	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/core/common"
	"0chain.net/smartcontract/stakepool/spenum"
)

type providerRequest struct {
	ID string `json:"id"`
}

func (pr *providerRequest) decode(p []byte) error {
	return json.Unmarshal(p, pr)
}

func Kill(
	input []byte,
	clientID, ownerId string,
	killSlash float64,
	providerSpecific func(providerRequest) (ProviderI, stakepool.AbstractStakePool, error),
	pType spenum.Provider,
	balances cstate.StateContextI,
) error {
	var errCode = "kill_" + pType.String() + "_failed"
	if err := smartcontractinterface.AuthorizeWithOwner(errCode, func() bool {
		return ownerId == clientID
	}); err != nil {
		return err
	}

	var req providerRequest
	if err := req.decode(input); err != nil {
		return common.NewError(errCode, err.Error())
	}

	p, sp, err := providerSpecific(req)
	if err != nil {
		return err
	}

	if p.IsKilled() {
		return common.NewError(errCode, "already killed")
	}
	p.Kill()
	if err := p.Save(balances); err != nil {
		return common.NewError(errCode, "cannot save: "+err.Error())
	}

	sp.Kill()
	if err := sp.SlashFraction(
		killSlash,
		req.ID,
		pType,
		balances,
	); err != nil {
		return common.NewError(errCode, "can't slash validator: "+err.Error())
	}

	if err = sp.Save(spenum.Validator, req.ID, balances); err != nil {
		return common.NewError(errCode, fmt.Sprintf("saving stake pool: %v", err))
	}

	if err := emitUpdateProvider(p, sp, balances); err != nil {
		return common.NewError(errCode, fmt.Sprintf("emitting event: %v", err))
	}

	return nil
}
