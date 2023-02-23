package provider

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
)

func (p *Provider) HealthCheck(
	now common.Timestamp,
	tag event.EventTag,
	balances cstate.StateContextI,
) {
	balances.EmitEvent(
		event.TypeStats,
		tag,
		p.Id(),
		dbs.DbHealthCheck{
			ID:              p.Id(),
			LastHealthCheck: p.LastHealthCheck,
			Downtime:        common.Downtime(p.LastHealthCheck, t.CreationDate),
		})

	p.LastHealthCheck = now
}
