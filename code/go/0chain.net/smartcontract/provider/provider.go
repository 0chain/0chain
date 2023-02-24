package provider

import (
	"fmt"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/stakepool/spenum"
)

//go:generate msgp -io=false -tests=false -v

type Abstract interface {
	IsActive(common.Timestamp, common.Timestamp) (bool, string)
	Kill()
	IsKilled() bool
	IsShutDown() bool
	Id() string
	Type() spenum.Provider
	ShutDown()
}

type Provider struct {
	ID              string           `json:"id" validate:"hexadecimal,len=64"`
	LastHealthCheck common.Timestamp `json:"last_health_check"`
	HasBeenShutDown bool             `json:"is_shut_down"`
	HasBeenKilled   bool             `json:"is_killed"`
	ProviderType    spenum.Provider  `json:"provider_type"`
}

func GetKey(id string) datastore.Key {
	return "provider:" + id
}

func (p *Provider) Id() string {
	return p.ID
}

func (p *Provider) IsActive(now, healthCheckPeriod common.Timestamp) (bool, string) {
	if p.IsKilled() {
		return false, "provider was killed"
	}
	if p.IsShutDown() {
		return false, "provider was shutdown"
	}
	if p.LastHealthCheck < (now - healthCheckPeriod) {
		return false, fmt.Sprintf(" failed health check, last check %v.", p.LastHealthCheck)
	}
	return true, ""
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

func (p *Provider) Type() spenum.Provider {
	return p.ProviderType
}
