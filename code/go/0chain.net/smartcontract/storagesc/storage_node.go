package storagesc

import (
	"encoding/json"
	"errors"

	"0chain.net/core/datastore"
	"0chain.net/core/util/entitywrapper"
	"0chain.net/smartcontract/provider"
	"0chain.net/smartcontract/stakepool"
	"github.com/0chain/common/core/logging"
)

//msgp:ignore StorageNode StorageAllocation AllocationChallenges storageNodeCommon
//go:generate msgp -io=false -tests=false -unexported -v

func init() {
	entitywrapper.RegisterWrapper(&StorageNode{},
		map[string]entitywrapper.EntityI{
			entitywrapper.DefaultOriginVersion: &storageNodeV1{},
			"v2":                               &storageNodeV2{},
		})
}

type StorageNode struct {
	entitywrapper.Wrapper
}

func (sn *StorageNode) TypeName() string {
	return "storage_node"
}

func (sn *StorageNode) UnmarshalMsg(data []byte) ([]byte, error) {
	return sn.UnmarshalMsgType(data, sn.TypeName())
}

func (sn *StorageNode) UnmarshalJSON(data []byte) error {
	return sn.UnmarshalJSONType(data, sn.TypeName())
}

type storageNodeCommon struct {
	*storageNodeV1
	origin entitywrapper.EntityI
}

func (snc *storageNodeCommon) commitChanges() {
	if snc.storageNodeV1 == snc.origin {
		// the origin struct, no need to move the changes
		return
	}

	if snc.origin.GetVersion() == "v2" {
		sn := snc.origin.(*storageNodeV2)
		sn.ApplyCommonChanges(snc)
	}
}

// common returns the common fields of storage node for read only actions
func (sn *StorageNode) common() *storageNodeCommon {
	switch snv := sn.Entity().(type) {
	case *storageNodeV1:
		return &storageNodeCommon{
			storageNodeV1: snv,
			origin:        snv,
		}
	case *storageNodeV2:
		v1 := &storageNodeV1{
			Provider:                snv.Provider,
			BaseURL:                 snv.BaseURL,
			Terms:                   snv.Terms,
			Capacity:                snv.Capacity,
			Allocated:               snv.Allocated,
			PublicKey:               snv.PublicKey,
			SavedData:               snv.SavedData,
			DataReadLastRewardRound: snv.DataReadLastRewardRound,
			LastRewardDataReadRound: snv.LastRewardDataReadRound,
			StakePoolSettings:       snv.StakePoolSettings,
			RewardRound:             snv.RewardRound,
			NotAvailable:            snv.NotAvailable,
		}
		return &storageNodeCommon{
			storageNodeV1: v1,
			origin:        snv,
		}
	default:
		logging.Logger.Panic("unknown storage node wrapper entity")
		return nil
	}
}

func (sn *StorageNode) commonUpdate(f func(*storageNodeCommon) error) error {
	csn := sn.common()
	if err := f(csn); err != nil {
		return err
	}
	csn.commitChanges()

	sn.SetEntity(csn.origin)
	return nil
}

// validate the blobber configurations
func (sn *StorageNode) validate(conf *Config) error {
	csn := sn.common()
	if err := csn.Terms.validate(conf); err != nil {
		return err
	}

	if csn.Capacity <= conf.MinBlobberCapacity {
		return errors.New("insufficient blobber capacity")
	}

	return validateBaseUrl(&csn.BaseURL)
}

func (sn *StorageNode) GetKey() datastore.Key {
	return provider.GetKey(sn.common().ID)
}

func (sn *StorageNode) GetUrlKey(globalKey string) datastore.Key {
	return GetUrlKey(sn.common().BaseURL, globalKey)
}

func (sn *StorageNode) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *StorageNode) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	return nil
}

type storageNodeV1 struct {
	provider.Provider
	BaseURL                 string  `json:"url"`
	Terms                   Terms   `json:"terms"`     // terms
	Capacity                int64   `json:"capacity"`  // total blobber capacity
	Allocated               int64   `json:"allocated"` // allocated capacity
	PublicKey               string  `json:"-"`
	SavedData               int64   `json:"saved_data"`
	DataReadLastRewardRound float64 `json:"data_read_last_reward_round"` // in GB
	LastRewardDataReadRound int64   `json:"last_reward_data_read_round"` // last round when data read was updated
	// StakePoolSettings used initially to create and setup stake pool.
	StakePoolSettings stakepool.Settings `json:"stake_pool_settings"`
	RewardRound       RewardRound        `json:"reward_round"`
	NotAvailable      bool               `json:"not_available"`
}

func (sn1 *storageNodeV1) GetVersion() string {
	return entitywrapper.DefaultOriginVersion
}

// StorageNode represents Blobber configurations.
type storageNodeV2 struct {
	provider.Provider
	Version                 string  `json:"version"`
	BaseURL                 string  `json:"url"`
	Terms                   Terms   `json:"terms"`     // terms
	Capacity                int64   `json:"capacity"`  // total blobber capacity
	Allocated               int64   `json:"allocated"` // allocated capacity
	PublicKey               string  `json:"-"`
	SavedData               int64   `json:"saved_data"`
	DataReadLastRewardRound float64 `json:"data_read_last_reward_round"` // in GB
	LastRewardDataReadRound int64   `json:"last_reward_data_read_round"` // last round when data read was updated
	// StakePoolSettings used initially to create and setup stake pool.
	StakePoolSettings stakepool.Settings `json:"stake_pool_settings"`
	RewardRound       RewardRound        `json:"reward_round"`
	NotAvailable      bool               `json:"not_available"`
	IsRestricted      *bool              `json:"is_restricted,omitempty"`
}

func (sn2 *storageNodeV2) GetVersion() string {
	return "v2"
}

func (sn2 *storageNodeV2) ApplyCommonChanges(snc *storageNodeCommon) {
	sn2.Provider = snc.Provider
	sn2.BaseURL = snc.BaseURL
	sn2.Terms = snc.Terms
	sn2.Capacity = snc.Capacity
	sn2.Allocated = snc.Allocated
	sn2.PublicKey = snc.PublicKey
	sn2.SavedData = snc.SavedData
	sn2.DataReadLastRewardRound = snc.DataReadLastRewardRound
	sn2.LastRewardDataReadRound = snc.LastRewardDataReadRound
	sn2.StakePoolSettings = snc.StakePoolSettings
	sn2.RewardRound = snc.RewardRound
	sn2.NotAvailable = snc.NotAvailable
}
