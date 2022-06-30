package stakepool

import (
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
	data := &event.DelegatePool{
		Balance:      dp.Balance,
		PoolID:       poolId,
		ProviderType: int(providerType),
		ProviderID:   providerId,
		DelegateID:   dp.DelegateID,

		Status:       int(dp.Status),
		RoundCreated: balances.GetBlock().Round,
	}

	balances.EmitEvent(
		event.TypeStats,
		event.TagAddOrOverwriteDelegatePool,
		providerId,
		data,
	)
	return nil
}

func (dpu DelegatePoolUpdate) emitUpdate(
	balances cstate.StateContextI,
) error {

	balances.EmitEvent(
		event.TypeStats,
		event.TagUpdateDelegatePool,
		dpu.PoolId,
		dpu,
	)
	return nil
}
