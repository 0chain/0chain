package provider

import (
	"0chain.net/core/common"
)

const healthCheckTime = 60 * 60

type Status int

const (
	Active Status = iota
	Inactive
	ShutDown
	Killed
	NonExistent
)

var statusString = []string{"active", "inactive", "shut_down", "killed"}

func (p Status) String() string {
	return statusString[p]
}

type ProviderI interface {
	Status(now common.Timestamp) Status
	HealthCheck(now common.Timestamp) error
	ShutDown() error
	Kill() error
}

type Provider struct {
	LastHealthCheck common.Timestamp `json:"last_health_check"`
	IsShutDown      bool             `json:"is_shut_down"`
	IsKilled        bool             `json:"is_killed"`
}

func (p *Provider) Status(now common.Timestamp, healthCheckPeriod common.Timestamp) Status {
	if p.IsKilled {
		return Killed
	}
	if p.IsShutDown {
		return ShutDown
	}
	if p.LastHealthCheck <= (now - healthCheckPeriod) {
		return Inactive
	}
	return Active
}

func (p *Provider) HealthCheck(now common.Timestamp) {
	p.LastHealthCheck = now
}

func (p *Provider) ShutDown() {
	p.IsShutDown = true
}

func (p *Provider) Kill() {
	p.IsKilled = true
}
