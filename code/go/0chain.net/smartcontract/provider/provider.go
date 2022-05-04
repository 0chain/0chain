package provider

import (
	"0chain.net/core/common"
)

// am_i_active? REST
// update SC
// health check SC
// shut down SC
// kill SC
const healthCheckTime = 60 * 60

type Status int

const (
	Active Status = iota
	Inactive
	ShutDown
	Killed
)

var statusString = []string{"active", "inactive", "shut_down", "killed"}

func (p Status) String() string {
	return statusString[p]
}

type ProviderI interface {
	Status(now common.Timestamp) Status
	PassHealthCheck(now common.Timestamp) error
	ShutDown() error
	Kill() error
}

type Provider struct {
	LastHealthCheck common.Timestamp `json:"last_health_check"`
	IsShutDown      bool             `json:"is_shut_down"`
	IsKilled        bool             `json:"is_killed"`
}

func (p *Provider) Status(now common.Timestamp) Status {
	if p.IsKilled {
		return Killed
	}
	if p.IsShutDown {
		return ShutDown
	}
	if p.LastHealthCheck <= (now - healthCheckTime) {
		return Inactive
	}
	return Active
}

func (p *Provider) PassHealthCheck(now common.Timestamp) {
	p.LastHealthCheck = now
}

func (p *Provider) ShutDown() {
	p.IsShutDown = true
}

func (p *Provider) Kill() {
	p.IsKilled = true
}
