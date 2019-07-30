package minersc

import (
	"encoding/json"
	"errors"
	"net/url"

	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	// "0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

var allMinersKey = datastore.Key(ADDRESS + encryption.Hash("all_miners"))

//MinerNode struct that holds information about the registering miner
type MinerNode struct {
	ID              string                       `json:"id"`
	BaseURL         string                       `json:"url"`
	PublicKey       string                       `json:"-"`
	MinerPercentage float64                      `json:"miner_percentage"`
	Pending         map[string]*sci.DelegatePool `json:"pending"`
	Active          map[string]*sci.DelegatePool `json:"active"`
	Deleting        map[string]*sci.DelegatePool `json:"deleting"`
}

func NewMinerNode() *MinerNode {
	mn := &MinerNode{}
	mn.Pending = make(map[string]*sci.DelegatePool)
	mn.Active = make(map[string]*sci.DelegatePool)
	mn.Deleting = make(map[string]*sci.DelegatePool)
	return mn
}

func (mn *MinerNode) getKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + mn.ID)
}

func (mn *MinerNode) Encode() []byte {
	buff, _ := json.Marshal(mn)
	return buff
}

func (mn *MinerNode) decodeFromValues(params url.Values) error {
	mn.BaseURL = params.Get("baseurl")
	mn.ID = params.Get("id")

	if mn.BaseURL == "" || mn.ID == "" {
		return errors.New("BaseURL or ID is not specified")
	}
	return nil

}

func (mn *MinerNode) Decode(input []byte) error {
	return json.Unmarshal(input, mn)
}

func (mn *MinerNode) GetHash() string {
	return util.ToHex(mn.GetHashBytes())
}

func (mn *MinerNode) GetHashBytes() []byte {
	return encryption.RawHash(mn.Encode())
}

func (mn *MinerNode) TotalStaked() state.Balance {
	var staked state.Balance
	for _, p := range mn.Active {
		staked += p.Balance
	}
	return staked
}

type ViewchangeInfo struct {
	ChainId         string `json:"chain_id"`
	ViewchangeRound int64  `json:"viewchange_round"`
	//the round when call for dkg with viewchange members and round will be announced
	ViewchangeCFDRound int64 `json:"viewchange_cfd_round"`
}

func (vc *ViewchangeInfo) encode() []byte {
	buff, _ := json.Marshal(vc)
	return buff
}

type MinerNodes struct {
	Nodes []*MinerNode
}

func (mn *MinerNodes) Encode() []byte {
	buff, _ := json.Marshal(mn)
	return buff
}

func (mn *MinerNodes) Decode(input []byte) error {
	err := json.Unmarshal(input, mn)
	if err != nil {
		return err
	}
	return nil
}

func (mn *MinerNodes) GetHash() string {
	return util.ToHex(mn.GetHashBytes())
}

func (mn *MinerNodes) GetHashBytes() []byte {
	return encryption.RawHash(mn.Encode())
}

type globalNode struct {
	ID           datastore.Key
	LastRound    int64
	MaxStake     int64
	MinStake     int64
	InterestRate float64
	ViewChange   int64
	FreezeBefore int64
}

func (gn *globalNode) Encode() []byte {
	buff, _ := json.Marshal(gn)
	return buff
}

func (gn *globalNode) Decode(input []byte) error {
	return json.Unmarshal(input, gn)
}

func (gn *globalNode) GetKey() datastore.Key {
	return datastore.Key(gn.ID + gn.ID)
}

func (gn *globalNode) GetHash() string {
	return util.ToHex(gn.GetHashBytes())
}

func (gn *globalNode) GetHashBytes() []byte {
	return encryption.RawHash(gn.Encode())
}

type ViewChangeLock struct {
	DeleteViewChangeSet bool          `json:"delete_on_vc_set"`
	DeleteRound         int64         `json:"delete_on_round"`
	Owner               datastore.Key `json:"owner"`
}

func (vcl *ViewChangeLock) IsLocked(entity interface{}) bool {
	round, ok := entity.(int64)
	if ok {
		return !vcl.DeleteViewChangeSet || round < vcl.DeleteRound
	}
	return true
}

func (vcl *ViewChangeLock) LockStats(entity interface{}) []byte {
	round, ok := entity.(int64)
	if ok {
		p := &poolStat{ViewChangeLock: vcl, CurrentRound: round, Locked: vcl.IsLocked(round)}
		return p.encode()
	}
	return nil
}

type poolStat struct {
	*ViewChangeLock
	CurrentRound int64 `json:"current_round"`
	Locked       bool  `json:"locked"`
}

func (ps *poolStat) encode() []byte {
	buff, _ := json.Marshal(ps)
	return buff
}

func (ps *poolStat) decode(input []byte) error {
	return json.Unmarshal(input, ps)
}

type UserNode struct {
	ID    string               `json:"id"`
	Pools map[string]*poolInfo `json:"pool_map"`
}

func NewUserNode() *UserNode {
	return &UserNode{Pools: make(map[string]*poolInfo)}
}

func (un *UserNode) Encode() []byte {
	buff, _ := json.Marshal(un)
	return buff
}

func (un *UserNode) Decode(input []byte) error {
	return json.Unmarshal(input, un)
}

func (un *UserNode) GetKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + un.ID)
}

func (un *UserNode) GetHash() string {
	return util.ToHex(un.GetHashBytes())
}

func (un *UserNode) GetHashBytes() []byte {
	return encryption.RawHash(un.Encode())
}

type poolInfo struct {
	MinerID string `json:"miner_id"`
	Balance int64  `json:"balance"`
}

type userPoolsResponse struct {
	*poolInfo
	StakeDiversity float64 `json:"stake_diversity"`
	TxnHash        string  `json:"txn_hash"`
}

type userResponse struct {
	Pools []*userPoolsResponse `json:"pools"`
}

func (ur *userResponse) Encode() []byte {
	buff, _ := json.Marshal(ur)
	return buff
}

func (ur *userResponse) Decode(input []byte) error {
	return json.Unmarshal(input, ur)
}
