package minersc

import (
	"encoding/json"
	"errors"
	"net/url"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	// "0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

var allMinersKey = datastore.Key(ADDRESS + encryption.Hash("all_miners"))

type simpleMinerNode struct {
	ID              string  `json:"id"`
	BaseURL         string  `json:"url"`
	PublicKey       string  `json:"-"`
	MinerPercentage float64 `json:"miner_percentage"`
}

func (smn *simpleMinerNode) Encode() []byte {
	buff, _ := json.Marshal(smn)
	return buff
}

func (smn *simpleMinerNode) Decode(input []byte) error {
	return json.Unmarshal(input, smn)
}

//MinerNode struct that holds information about the registering miner
type MinerNode struct {
	*simpleMinerNode `json:"simple_miner_node"`
	Pending          map[string]*DelegatePool `json:"pending"`
	Active           map[string]*DelegatePool `json:"active"`
	Deleting         map[string]*DelegatePool `json:"deleting"`
}

func NewMinerNode() *MinerNode {
	mn := &MinerNode{simpleMinerNode: &simpleMinerNode{}}
	mn.Pending = make(map[string]*DelegatePool)
	mn.Active = make(map[string]*DelegatePool)
	mn.Deleting = make(map[string]*DelegatePool)
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
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(input, &objMap)
	if err != nil {
		return err
	}
	smnbytes, ok := objMap["simple_miner_node"]
	if ok {
		var smn *simpleMinerNode
		err = json.Unmarshal(*smnbytes, &smn)
		if err != nil {
			return err
		}
		mn.simpleMinerNode = smn
	}
	active, ok := objMap["active"]
	if ok {
		err := DecodePool(active, mn, mn.Active)
		if err != nil {
			return err
		}
	}
	pend, ok := objMap["pending"]
	if ok {
		err := DecodePool(pend, mn, mn.Pending)
		if err != nil {
			return err
		}
	}
	delete, ok := objMap["deleting"]
	if ok {
		err := DecodePool(delete, mn, mn.Deleting)
		if err != nil {
			return err
		}
	}
	return nil
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
	ChainId         string `json:chain_id`
	ViewchangeRound int64  `json:viewchange_round`
	//the round when call for dkg with viewchange members and round will be announced
	ViewchangeCFDRound int64 `json:viewchange_cfd_round`
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

type PoolStats struct {
	DelegateID   string        `json:"delegate_id"`
	High         state.Balance `json:"high"`
	Low          state.Balance `json:"low"`
	InterestRate float64       `json:"interest_rate"`
	TotalPaid    state.Balance `json:"total_paid"`
	NumRounds    int64         `json:"number_rounds"`
}

func (ps *PoolStats) Encode() []byte {
	buff, _ := json.Marshal(ps)
	return buff
}

func (ps *PoolStats) Decode(input []byte) error {
	return json.Unmarshal(input, ps)
}

type DelegatePool struct {
	*PoolStats                `json:"stats"`
	*tokenpool.ZcnLockingPool `json:"pool"`
}

func NewDelegatePool() *DelegatePool {
	return &DelegatePool{ZcnLockingPool: &tokenpool.ZcnLockingPool{}, PoolStats: &PoolStats{Low: -1}}
}

func (dp *DelegatePool) Encode() []byte {
	buff, _ := json.Marshal(dp)
	return buff
}

func (dp *DelegatePool) Decode(input []byte) error {
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(input, &objMap)
	if err != nil {
		return err
	}
	s, ok := objMap["stats"]
	if ok {
		err = dp.PoolStats.Decode(*s)
		if err != nil {
			return err
		}
	}
	p, ok := objMap["pool"]
	if ok {
		err = dp.ZcnLockingPool.Decode(*p, &ViewChangeLock{})
		if err != nil {
			return err
		}
	}
	return err
}

func DecodePool(pool *json.RawMessage, mn *MinerNode, pools map[string]*DelegatePool) error {
	var rawMessagesPools map[string]*json.RawMessage
	err := json.Unmarshal(*pool, &rawMessagesPools)
	if err != nil {
		return err
	}
	for _, raw := range rawMessagesPools {
		tempPool := NewDelegatePool()
		err = tempPool.Decode(*raw)
		if err != nil {
			return err
		}
		if _, ok := pools[tempPool.ID]; !ok {
			pools[tempPool.ID] = tempPool
		}
	}
	return err
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
