package provider

import (
	"time"

	"0chain.net/core/common"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
)

func HealthCheck(
	now common.Timestamp,
	provider ProviderI,
	healthCheckPeriod time.Duration,
	balances cstate.StateContextI,
) error {
	lastHeathCheck := provider.HealthCheck(now)

	balances.EmitEvent(
		event.TypeStats,
		event.TagProviderHealthCheck,
		provider.Id(),
		dbs.HealthCheck{
			Provider: dbs.Provider{
				ProviderId:   provider.Id(),
				ProviderType: provider.Type(),
			},
			Now:               now,
			LastHealthCheck:   lastHeathCheck,
			HealthCheckPeriod: healthCheckPeriod,
		})
	return nil
}
