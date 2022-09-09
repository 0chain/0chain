package provider

import (
	"errors"
	"fmt"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/zcnsc"

	"0chain.net/smartcontract/stakepool/spenum"

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

var statusString = []string{"active", "inactive", "shut_down", "killed", "non_existent"}

func (p Status) String() string {
	return statusString[p]
}

// swagger:model StatusInfo
type StatusInfo struct {
	Status Status `json:"status"`
	Reason string `json:"reason"`
}

func getKey(id string, pType spenum.Provider) (datastore.Key, error) {
	scAddress, err := scAddress(pType)
	if err != nil {
		return "", err
	}
	return datastore.Key(scAddress + pType.String() + id), nil
}

func scAddress(pType spenum.Provider) (string, error) {
	switch pType {
	case spenum.Miner:
		return minersc.ADDRESS, nil
	case spenum.Sharder:
		return minersc.ADDRESS, nil
	case spenum.Validator:
		return storagesc.ADDRESS, nil
	case spenum.Blobber:
		return storagesc.ADDRESS, nil
	case spenum.Authorizer:
		return zcnsc.ADDRESS, nil
	default:
		return "", fmt.Errorf("unknown provider type %v", pType)
	}
}

type ProviderI interface {
	Status(common.Timestamp, common.Timestamp) (Status, string)
	HealthCheck(common.Timestamp)
	ShutDown()
	Kill()
	IsKilled() bool
	IsShutDown() bool
	Id() string
}

func Type(p ProviderI) spenum.Provider {
	switch p.(type) {
	case storagesc.ValidationNode:
		return spenum.Validator
	case storagesc.StorageNode:
		return spenum.Blobber
	default:
		return spenum.Unknown
	}
}

func Get(
	id string,
	pType spenum.Provider,
	sCtx state.CommonStateContextI,
) (ProviderI, error) {
	var err error
	key, err := getKey(id, pType)
	if err != nil {
		return nil, err
	}
	switch pType {
	case spenum.Miner:
		return nil, fmt.Errorf("miner as provider not implemented yet")
	case spenum.Sharder:
		return nil, fmt.Errorf("sharder as provider not implemented yet")
	case spenum.Validator:
		validator := &storagesc.ValidationNode{}
		return validator, sCtx.GetTrieNode(key, validator)
	case spenum.Blobber:
		blobber := &storagesc.StorageNode{}
		return blobber, sCtx.GetTrieNode(key, blobber)
	case spenum.Authorizer:
		return nil, fmt.Errorf("authoriser as provider not implemented yet")
	default:
		return nil, fmt.Errorf("unknown provider type %v", pType)
	}
}

func Save(p ProviderI, sCtx state.StateContextI) error {
	var err error
	pType := Type(p)
	key, err := getKey(p.Id(), pType)
	if err != nil {
		return err
	}
	switch pType {
	case spenum.Validator:
		validator, ok := p.(storagesc.ValidationNode)
		if !ok {
			return errors.New("error converting provider to validator node")
		}
		_, err = sCtx.InsertTrieNode(key, &validator)
	case spenum.Blobber:
		blobber, ok := p.(storagesc.StorageNode)
		if !ok {
			return errors.New("error converting provider to storage node")
		}
		_, err = sCtx.InsertTrieNode(key, &blobber)
	default:
		err = fmt.Errorf("unknown provider type %s", pType.String())
	}
	return err
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
