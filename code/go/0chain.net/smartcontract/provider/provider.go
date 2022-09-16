package provider

import (
	"errors"
	"fmt"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
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
	HealthCheck(common.Timestamp)
	ShutDown()
	Kill()
	IsKilled() bool
	IsShutDown() bool
	Id() string
	Save(state.StateContextI) error
}

type Provider struct {
	ID              string           `json:"id"`
	LastHealthCheck common.Timestamp `json:"last_health_check"`
	HasBeenShutDown bool             `json:"is_shut_down"`
	HasBeenKilled   bool             `json:"is_killed"`
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
		return Inactive, fmt.Sprintf("\tfailed health check, last check %v", p.LastHealthCheck)
	}
	return Active, ""
}

func (p *Provider) IsShutDown() bool {
	return p.HasBeenShutDown
}

func (p *Provider) IsKilled() bool {
	return p.HasBeenKilled
}

func (p *Provider) HealthCheck(now common.Timestamp) {
	p.LastHealthCheck = now
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
