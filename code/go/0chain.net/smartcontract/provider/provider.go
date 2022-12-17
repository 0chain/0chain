package provider

import (
	"errors"
	"fmt"
	"time"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
 "0chain.net/core/datastore"
	"0chain.net/chaincore/chain/state"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/smartcontract/stakepool/spenum"
)

//go:generate msgp -io=false -tests=false -v

// swagger:enum Status
type Status int

const (
	Active Status = iota + 1
	Inactive
	ShutDown
	Killed
	NonExistent
)

var statusString = []string{"unknown", "active", "inactive", "shut_down", "killed", "non_existent"}

func (p Status) String() string {
	return statusString[p]
}

// swagger:model StatusInfo
type StatusInfo struct {
	Status Status `json:"status"`
	Reason string `json:"reason"`
}

type ProviderI interface {
	Status(common.Timestamp, common.Timestamp) (Status, string)
	Kill()
	IsKilled() bool
	IsShutDown() bool
	Id() string
	Save(state.StateContextI) error
	Type() spenum.Provider
	HealthCheck(common.Timestamp, time.Duration, cstate.StateContextI)
	ShutDown()
	EmitUpdate(stakepool.AbstractStakePool, cstate.StateContextI)
}

type Provider struct {
	ID              string           `json:"id"`
	LastHealthCheck common.Timestamp `json:"last_health_check"`
	HasBeenShutDown bool             `json:"is_shut_down"`
	HasBeenKilled   bool             `json:"is_killed"`
}



func GetKey(id string) datastore.Key {
	return "provider:" + id
}

func (p *Provider) Id() string {
	return p.ID
}

func (p *Provider) Status(now, healthCheckPeriod common.Timestamp) (Status, string) {
	if p.IsKilled() {
		return Killed, Killed.String()
	}
	if p.IsShutDown() {
		return ShutDown, ShutDown.String()
	}
	if p.LastHealthCheck < (now - healthCheckPeriod) {
		return Inactive, fmt.Sprintf(" failed health check, last check %v.", p.LastHealthCheck)
	}
	return Active, ""
}

func (p *Provider) IsShutDown() bool {
	return p.HasBeenShutDown
}

func (p *Provider) IsKilled() bool {
	return p.HasBeenKilled
}

func (p *Provider) ShutDown() {
	p.HasBeenShutDown = true
}

func (p *Provider) Kill() {
	p.HasBeenKilled = true
}

func (p *Provider) Save(i state.StateContextI) error {
	return errors.New("save should be called from main provider object")
}

func (p *Provider) Type() spenum.Provider {
	return spenum.Invalid
}

func (p *Provider) EmitUpdate(sp stakepool.AbstractStakePool, balances cstate.StateContextI) {
	updates := dbs.NewDbUpdateProvider(p.Id(), p.Type())
	updates.Updates = map[string]interface{}{
		"primaryKey":        p.Id(),
		"last_health_check": p.LastHealthCheck,
		"is_killed":         p.IsKilled(),
		"is_shut_down":      p.IsShutDown(),
	}
	if sp != nil {
		updates.Updates["delegate_wallet"] = sp.GetSettings().DelegateWallet
		updates.Updates["min_stake"] = sp.GetSettings().MinStake
		updates.Updates["max_stake"] = sp.GetSettings().MaxStake
		updates.Updates["num_delegates"] = sp.GetSettings().MaxNumDelegates
		updates.Updates["service_charge"] = sp.GetSettings().ServiceChargeRatio
	}
	balances.EmitEvent(event.TypeStats, event.TagUpdateProvider, p.ID, updates)
}

func (p *Provider) HealthCheck(
	now common.Timestamp,
	healthCheckPeriod time.Duration,
	balances cstate.StateContextI,
) {
	balances.EmitEvent(
		event.TypeStats,
		event.TagProviderHealthCheck,
		p.Id(),
		dbs.HealthCheck{
			Provider: dbs.Provider{
				ProviderId:   p.Id(),
				ProviderType: p.Type(),
			},
			Now:               now,
			LastHealthCheck:   p.LastHealthCheck,
			HealthCheckPeriod: healthCheckPeriod,
		})

	p.LastHealthCheck = now
}
