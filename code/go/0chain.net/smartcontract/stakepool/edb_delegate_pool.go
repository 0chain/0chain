package stakepool

import (
	"encoding/json"

	"0chain.net/smartcontract/stakepool/spenum"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
)

type DelegatePoolUpdate dbs.DelegatePoolUpdate

func newDelegatePoolUpdate(pId string, pType spenum.Provider) *DelegatePoolUpdate {
	var spu DelegatePoolUpdate
	spu.ProviderId = pId
	spu.ProviderType = int(pType)
	spu.Updates = make(map[string]interface{})
	return &spu
}

func (dp DelegatePool) emitNew(
	poolId, providerId string,
	providerType spenum.Provider,
	balances cstate.StateContextI,
) error {
	dpBalance, err := dp.Balance.Int64()
	if err != nil {
		return err
	}
	data, err := json.Marshal(&event.DelegatePool{
		Balance:      dpBalance,
		PoolID:       poolId,
		ProviderType: int(providerType),
		ProviderID:   providerId,
		DelegateID:   dp.DelegateID,

		Status:       int(dp.Status),
		RoundCreated: balances.GetBlock().Round,
	})
	if err != nil {
		return err
	}
	balances.EmitEvent(
		event.TypeStats,
		event.TagAddOrOverwriteDelegatePool,
		providerId,
		string(data),
	)
	return nil
}

func (dpu DelegatePoolUpdate) emitUpdate(
	balances cstate.StateContextI,
) error {
	data, err := json.Marshal(&dpu)
	if err != nil {
		return err
	}
	balances.EmitEvent(
		event.TypeStats,
		event.TagUpdateDelegatePool,
		dpu.PoolId,
		string(data),
	)
	return nil
}
