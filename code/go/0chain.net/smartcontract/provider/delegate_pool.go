package provider

import (
	"fmt"

	"0chain.net/core/common"
	"github.com/0chain/common/core/currency"

	"0chain.net/smartcontract/provider/spenum"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
)

//go:generate msgp -io=false -tests=false -unexported -v

type DelegatePool struct {
	Balance      currency.Coin     `json:"balance"`
	Reward       currency.Coin     `json:"reward"`
	Status       spenum.PoolStatus `json:"status"`
	RoundCreated int64             `json:"round_created"` // used for cool down
	DelegateID   string            `json:"delegate_id"`
	StakedAt     common.Timestamp  `json:"staked_at"`
}

type Settings struct {
	DelegateWallet     string        `json:"delegate_wallet"`
	MinStake           currency.Coin `json:"min_stake"`
	MaxStake           currency.Coin `json:"max_stake"`
	MaxNumDelegates    int           `json:"num_delegates"`
	ServiceChargeRatio float64       `json:"service_charge"`
}

type DelegatePoolUpdate dbs.DelegatePoolUpdate

func NewDelegatePoolUpdate(poolID, pId string, pType spenum.Provider) *DelegatePoolUpdate {
	var spu DelegatePoolUpdate
	spu.PoolId = poolID
	spu.ProviderId = pId
	spu.ProviderType = int(pType)
	spu.Updates = make(map[string]interface{})
	return &spu
}

func (dp DelegatePool) EmitNew(
	poolId, providerId string,
	providerType spenum.Provider,
	balances cstate.StateContextI,
) {
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
		fmt.Sprintf("%d:%s:%s", providerType, providerId, poolId),
		data,
	)
}

func (dpu DelegatePoolUpdate) EmitUpdate(
	balances cstate.StateContextI,
) {
	balances.EmitEvent(
		event.TypeStats,
		event.TagUpdateDelegatePool,
		dpu.PoolId,
		delegatePoolUpdateToDbsDelegatePoolUpdate(dpu),
	)
}

func delegatePoolUpdateToDbsDelegatePoolUpdate(dpu DelegatePoolUpdate) dbs.DelegatePoolUpdate {
	return dbs.DelegatePoolUpdate{
		DelegatePoolId: dpu.DelegatePoolId,
		Updates:        dpu.Updates,
	}
}
