package dto

import (
	"0chain.net/core/common"
	"0chain.net/smartcontract/provider"
	"github.com/0chain/common/core/currency"
)

//go:generate msgp -io=false -tests=false -unexported -v

// NodeType used in pools statistic.
type NodeType int

// MinerDtoNode struct that holds information about the registering miner.
// swagger:model MinerDtoNode
type MinerDtoNode struct {
	*SimpleDtoNode `json:"simple_miner,omitempty"`
	*StakePool     `json:"stake_pool,omitempty"`
}

// swagger:model SimpleDtoNode
type SimpleDtoNode struct {
	provider.Provider
	N2NHost     string                `json:"n2n_host"`
	Host        string                `json:"host"`
	Port        int                   `json:"port"`
	Geolocation SimpleNodeGeolocation `json:"geolocation"`
	Path        string                `json:"path"`
	PublicKey   string                `json:"public_key"`
	ShortName   string                `json:"short_name"`
	BuildTag    string                `json:"build_tag"`
	TotalStaked currency.Coin         `json:"total_stake"`
	Delete      bool                  `json:"delete"`

	// settings and statistic

	// NodeType used for delegate pools statistic.
	NodeType NodeType `json:"node_type,omitempty"`

	// LastHealthCheck used to check for active node
	LastHealthCheck common.Timestamp `json:"last_health_check"`

	// Status will be set either node.NodeStatusActive or node.NodeStatusInactive
	Status int `json:"-" msg:"-"`

	//LastSettingUpdateRound will be set to round number when settings were updated
	LastSettingUpdateRound int64 `json:"last_setting_update_round"`
}

// swagger:model SimpleNodeGeolocation
type SimpleNodeGeolocation struct {
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

func NewMinerDtoNode() *MinerDtoNode {
	return &MinerDtoNode{
		SimpleDtoNode: &SimpleDtoNode{
			Provider: provider.Provider{},
		},
		StakePool: NewStakePool(),
	}
}
