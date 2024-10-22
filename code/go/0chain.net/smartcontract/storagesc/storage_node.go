package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util/entitywrapper"
	"0chain.net/smartcontract/provider"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/logging"
)

//msgp:ignore StorageNode StorageAllocation AllocationChallenges storageNodeBase
//go:generate msgp -io=false -tests=false -unexported -v

func init() {
	entitywrapper.RegisterWrapper(&StorageNode{},
		map[string]entitywrapper.EntityI{
			entitywrapper.DefaultOriginVersion: &storageNodeV1{},
			"v2":                               &storageNodeV2{},
			"v3":                               &storageNodeV3{},
			"v4":                               &storageNodeV4{},
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

func (sn *StorageNode) Msgsize() (s int) {
	return sn.Entity().Msgsize()
}

func (sn *StorageNode) mustBase() *storageNodeBase {
	b, ok := sn.Base().(*storageNodeBase)
	if !ok {
		logging.Logger.Panic("invalid storage node base type")
	}
	return b
}

func (sn *StorageNode) mustUpdateBase(f func(*storageNodeBase) error) error {
	return sn.UpdateBase(func(eb entitywrapper.EntityBaseI) error {
		b, ok := eb.(*storageNodeBase)
		if !ok {
			logging.Logger.Panic("invalid storage node base type")
		}

		err := f(b)
		if err != nil {
			return err
		}
		return nil
	})
}

// implement provider.AbstractProvider interface
func (sn *StorageNode) IsActive(now common.Timestamp, healthCheckPeriod time.Duration) (bool, string) {
	return sn.mustBase().IsActive(now, healthCheckPeriod)
}

// implement provider.AbstractProvider interface
func (sn *StorageNode) Kill() {
	//nolint:errcheck
	sn.mustUpdateBase(func(b *storageNodeBase) error {
		b.Kill()
		return nil
	})
}

func (sn *StorageNode) IsShutDown() bool {
	return sn.mustBase().IsShutDown()
}

// implement provider.AbstractProvider interface
func (sn *StorageNode) IsKilled() bool {
	return sn.mustBase().IsKilled()
}

// implement provider.AbstractProvider interface
func (sn *StorageNode) Id() string {
	return sn.mustBase().ID
}

// implement provider.AbstractProvider interface
func (sn *StorageNode) Type() spenum.Provider {
	return sn.mustBase().Type()
}

// implement provider.AbstractProvider interface
func (sn *StorageNode) ShutDown() {
	//nolint:errcheck
	sn.mustUpdateBase(func(b *storageNodeBase) error {
		b.ShutDown()
		return nil
	})
}

// validate the blobber configurations
func (sn *StorageNode) validate(conf *Config) error {
	csn := sn.mustBase()
	if err := csn.Terms.validate(conf); err != nil {
		return err
	}

	if csn.Capacity <= conf.MinBlobberCapacity {
		return errors.New("insufficient blobber capacity")
	}

	return validateBaseUrl(&csn.BaseURL)
}

func (sn *StorageNode) GetKey() datastore.Key {
	return provider.GetKey(sn.mustBase().ID)
}

func (sn *StorageNode) GetUrlKey(globalKey string) datastore.Key {
	return GetUrlKey(sn.mustBase().BaseURL, globalKey)
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

func (sn1 *storageNodeV1) InitVersion() {
	// do nothing cause it's original version of storage node
}

func (sn1 *storageNodeV1) GetBase() entitywrapper.EntityBaseI {
	b := storageNodeBase(*sn1)
	return &b
}

func (sn1 *storageNodeV1) MigrateFrom(e entitywrapper.EntityI) error {
	// nothing to migrate as this is original version of the storage node
	return nil
}

// use storageNodeV1 as the base
type storageNodeBase storageNodeV1

func (sb *storageNodeBase) CommitChangesTo(e entitywrapper.EntityI) {
	switch v := e.(type) {
	case *storageNodeV1:
		*v = storageNodeV1(*sb)
	case *storageNodeV2:
		v.ApplyBaseChanges(*sb)
	case *storageNodeV3:
		v.ApplyBaseChanges(*sb)
	case *storageNodeV4:
		v.ApplyBaseChanges(*sb)
	}
}

// StorageNode represents Blobber configurations.
type storageNodeV2 struct {
	provider.Provider
	Version                 string  `json:"version" msg:"version"`
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

const storageNodeV2Version = "v2"

func (sn2 *storageNodeV2) GetVersion() string {
	return storageNodeV2Version
}

func (sn2 *storageNodeV2) InitVersion() {
	sn2.Version = storageNodeV2Version
}

func (sn2 *storageNodeV2) GetBase() entitywrapper.EntityBaseI {
	return &storageNodeBase{
		Provider:                sn2.Provider,
		BaseURL:                 sn2.BaseURL,
		Terms:                   sn2.Terms,
		Capacity:                sn2.Capacity,
		Allocated:               sn2.Allocated,
		PublicKey:               sn2.PublicKey,
		SavedData:               sn2.SavedData,
		DataReadLastRewardRound: sn2.DataReadLastRewardRound,
		LastRewardDataReadRound: sn2.LastRewardDataReadRound,
		StakePoolSettings:       sn2.StakePoolSettings,
		RewardRound:             sn2.RewardRound,
		NotAvailable:            sn2.NotAvailable,
	}
}

func (sn2 *storageNodeV2) MigrateFrom(e entitywrapper.EntityI) error {
	v1, ok := e.(*storageNodeV1)
	if !ok {
		return errors.New("struct migrate fail, wrong storageNode type")
	}
	sn2.ApplyBaseChanges(storageNodeBase(*v1))
	sn2.Version = "v2"
	return nil
}

func (sn2 *storageNodeV2) ApplyBaseChanges(snc storageNodeBase) {
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

type storageNodeV3 struct {
	provider.Provider
	Version                 string  `json:"version" msg:"version"`
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
	IsRestricted      *bool              `json:"is_restricted"`
	IsEnterprise      *bool              `json:"is_enterprise"`
}

const storageNodeV3Version = "v3"

func (sn3 *storageNodeV3) GetVersion() string {
	return storageNodeV3Version
}

func (sn3 *storageNodeV3) InitVersion() {
	sn3.Version = storageNodeV3Version
}

func (sn3 *storageNodeV3) GetBase() entitywrapper.EntityBaseI {
	return &storageNodeBase{
		Provider:                sn3.Provider,
		BaseURL:                 sn3.BaseURL,
		Terms:                   sn3.Terms,
		Capacity:                sn3.Capacity,
		Allocated:               sn3.Allocated,
		PublicKey:               sn3.PublicKey,
		SavedData:               sn3.SavedData,
		DataReadLastRewardRound: sn3.DataReadLastRewardRound,
		LastRewardDataReadRound: sn3.LastRewardDataReadRound,
		StakePoolSettings:       sn3.StakePoolSettings,
		RewardRound:             sn3.RewardRound,
		NotAvailable:            sn3.NotAvailable,
	}
}

func (sn3 *storageNodeV3) MigrateFrom(e entitywrapper.EntityI) error {
	v2, ok := e.(*storageNodeV2)
	if !ok {
		return errors.New("struct migrate fail, wrong storageNode type")
	}

	base := v2.GetBase().(*storageNodeBase)
	sn3.ApplyBaseChanges(*base)
	sn3.Version = "v3"
	sn3.IsRestricted = v2.IsRestricted
	return nil
}

func (sn3 *storageNodeV3) ApplyBaseChanges(snc storageNodeBase) {
	sn3.Provider = snc.Provider
	sn3.BaseURL = snc.BaseURL
	sn3.Terms = snc.Terms
	sn3.Capacity = snc.Capacity
	sn3.Allocated = snc.Allocated
	sn3.PublicKey = snc.PublicKey
	sn3.SavedData = snc.SavedData
	sn3.DataReadLastRewardRound = snc.DataReadLastRewardRound
	sn3.LastRewardDataReadRound = snc.LastRewardDataReadRound
	sn3.StakePoolSettings = snc.StakePoolSettings
	sn3.RewardRound = snc.RewardRound
	sn3.NotAvailable = snc.NotAvailable
}

type storageNodeV4 struct {
	provider.Provider
	Version                 string  `json:"version" msg:"version"`
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
	IsRestricted      *bool              `json:"is_restricted"`
	IsEnterprise      *bool              `json:"is_enterprise"`

	ManagingWallet *string `json:"managing_wallet"`
	StorageVersion *int    `json:"storage_version"`
}

const storageNodeV4Version = "v4"

func (sn4 *storageNodeV4) GetVersion() string {
	return storageNodeV4Version
}

func (sn4 *storageNodeV4) InitVersion() {
	sn4.Version = storageNodeV4Version
}

func (sn4 *storageNodeV4) GetBase() entitywrapper.EntityBaseI {
	return &storageNodeBase{
		Provider:                sn4.Provider,
		BaseURL:                 sn4.BaseURL,
		Terms:                   sn4.Terms,
		Capacity:                sn4.Capacity,
		Allocated:               sn4.Allocated,
		PublicKey:               sn4.PublicKey,
		SavedData:               sn4.SavedData,
		DataReadLastRewardRound: sn4.DataReadLastRewardRound,
		LastRewardDataReadRound: sn4.LastRewardDataReadRound,
		StakePoolSettings:       sn4.StakePoolSettings,
		RewardRound:             sn4.RewardRound,
		NotAvailable:            sn4.NotAvailable,
	}
}

func (sn4 *storageNodeV4) MigrateFrom(e entitywrapper.EntityI) error {

	if v3, ok := e.(*storageNodeV3); ok {
		base := v3.GetBase().(*storageNodeBase)
		sn4.ApplyBaseChanges(*base)
		sn4.Version = "v4"
		sn4.IsRestricted = v3.IsRestricted
		sn4.IsEnterprise = v3.IsEnterprise
	} else if v2, ok := e.(*storageNodeV2); ok {
		base := v2.GetBase().(*storageNodeBase)
		sn4.ApplyBaseChanges(*base)
		sn4.Version = "v4"
		sn4.IsRestricted = v2.IsRestricted
	} else if v1, ok := e.(*storageNodeV1); ok {
		base := v1.GetBase().(*storageNodeBase)
		sn4.ApplyBaseChanges(*base)
		sn4.Version = "v4"
	} else {
		return fmt.Errorf("struct migrate to storageNodeV4 fail, wrong storageNode type")
	}

	return nil
}

func (sn4 *storageNodeV4) ApplyBaseChanges(snc storageNodeBase) {
	sn4.Provider = snc.Provider
	sn4.BaseURL = snc.BaseURL
	sn4.Terms = snc.Terms
	sn4.Capacity = snc.Capacity
	sn4.Allocated = snc.Allocated
	sn4.PublicKey = snc.PublicKey
	sn4.SavedData = snc.SavedData
	sn4.DataReadLastRewardRound = snc.DataReadLastRewardRound
	sn4.LastRewardDataReadRound = snc.LastRewardDataReadRound
	sn4.StakePoolSettings = snc.StakePoolSettings
	sn4.RewardRound = snc.RewardRound
	sn4.NotAvailable = snc.NotAvailable
}
