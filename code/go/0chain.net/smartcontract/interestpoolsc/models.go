package interestpoolsc

import (
	"encoding/json"
	"time"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	// "0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"

	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

const (
	INTEREST = 0
	STAKE    = 1
)

type SimpleGlobalNode struct {
	MinLock      int64   `json:"min_lock"`
	InterestRate float64 `json:"interest_rate"`
}

func (sgn *SimpleGlobalNode) Encode() []byte {
	buff, _ := json.Marshal(sgn)
	return buff
}

func (sgn *SimpleGlobalNode) Decode(input []byte) error {
	err := json.Unmarshal(input, sgn)
	return err
}

type GlobalNode struct {
	ID                datastore.Key
	*SimpleGlobalNode `json:"simple_global_node"`
	LockPeriod        time.Duration `json:"lock_period"`
}

func newGlobalNode() *GlobalNode {
	return &GlobalNode{ID: ADDRESS, SimpleGlobalNode: &SimpleGlobalNode{}}
}

func (gn *GlobalNode) Encode() []byte {
	buff, _ := json.Marshal(gn)
	return buff
}

func (gn *GlobalNode) Decode(input []byte) error {
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(input, &objMap)
	if err != nil {
		return err
	}
	sgn, ok := objMap["simple_global_node"]
	if ok {
		err = gn.SimpleGlobalNode.Decode(*sgn)
		if err != nil {
			return err
		}
	}
	var s string
	lp, ok := objMap["lock_period"]
	if ok {
		err = json.Unmarshal(*lp, &s)
		if err != nil {
			return err
		}
		dur, err := time.ParseDuration(s)
		if err != nil {
			return err
		}
		gn.LockPeriod = dur
	}
	return nil
}

func (gn *GlobalNode) GetHash() string {
	return util.ToHex(gn.GetHashBytes())
}

func (gn *GlobalNode) GetHashBytes() []byte {
	return encryption.RawHash(gn.Encode())
}

func (gn *GlobalNode) getKey() datastore.Key {
	return datastore.Key(gn.ID + gn.ID)
}

type typePool struct {
	*tokenpool.ZcnLockingPool `json:"pool"`
	Type                      int `json:"pool_type"`
}

func newTypePool() *typePool {
	return &typePool{ZcnLockingPool: &tokenpool.ZcnLockingPool{}}
}

func (tp *typePool) encode() []byte {
	buff, _ := json.Marshal(tp)
	return buff
}

func (tp *typePool) decode(input []byte) error {
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(input, &objMap)
	if err != nil {
		return err
	}
	pt, ok := objMap["pool_type"]
	if ok {
		var ty int
		err = json.Unmarshal(*pt, &ty)
		if err != nil {
			return err
		}
		tp.Type = ty
	}
	p, ok := objMap["pool"]
	if ok {
		err = tp.ZcnLockingPool.Decode(*p, &tokenLock{})
		if err != nil {
			return err
		}
	}
	return nil
}

type UserNode struct {
	ClientID datastore.Key               `json:"client_id"`
	Pools    map[datastore.Key]*typePool `json:"pools"`
}

func newUserNode(clientID datastore.Key) *UserNode {
	un := &UserNode{ClientID: clientID}
	un.Pools = make(map[datastore.Key]*typePool)
	return un
}

func (un *UserNode) Encode() []byte {
	buff, _ := json.Marshal(un)
	return buff
}

func (un *UserNode) Decode(input []byte) error {
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(input, &objMap)
	if err != nil {
		return err
	}
	cid, ok := objMap["client_id"]
	if ok {
		var id datastore.Key
		err = json.Unmarshal(*cid, &id)
		if err != nil {
			return err
		}
		un.ClientID = id
	}
	p, ok := objMap["pools"]
	if ok {
		var rawMessagesPools map[string]*json.RawMessage
		err = json.Unmarshal(*p, &rawMessagesPools)
		if err != nil {
			return err
		}
		for _, raw := range rawMessagesPools {
			tempPool := newTypePool()
			err = tempPool.decode(*raw)
			if err != nil {
				return err
			}
			un.addPool(tempPool)
		}
	}
	return nil
}

func (un *UserNode) GetHash() string {
	return util.ToHex(un.GetHashBytes())
}

func (un *UserNode) GetHashBytes() []byte {
	return encryption.RawHash(un.Encode())
}

func (un *UserNode) getKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + un.ClientID)
}

func (un *UserNode) hasPool(poolID datastore.Key) bool {
	pool := un.Pools[poolID]
	return pool != nil
}

func (un *UserNode) getPool(poolID datastore.Key) *typePool {
	return un.Pools[poolID]
}

func (un *UserNode) addPool(ip *typePool) error {
	if un.hasPool(ip.ID) {
		return common.NewError("can't add pool", "user node already has pool")
	}
	un.Pools[ip.ID] = ip
	return nil
}

func (un *UserNode) deletePool(poolID datastore.Key) error {
	if !un.hasPool(poolID) {
		return common.NewError("can't delete pool", "pool doesn't exist")
	}
	delete(un.Pools, poolID)
	return nil
}

type transferResponses struct {
	Responses []string `json:"responses"`
}

func (tr *transferResponses) addResponse(response string) {
	tr.Responses = append(tr.Responses, response)
}

func (tr *transferResponses) encode() []byte {
	buff, _ := json.Marshal(tr)
	return buff
}

func (tr *transferResponses) decode(input []byte) error {
	err := json.Unmarshal(input, tr)
	return err
}

type poolStats struct {
	Stats []*poolStat `json:"stats"`
}

func (ps *poolStats) encode() []byte {
	buff, _ := json.Marshal(ps)
	return buff
}

func (ps *poolStats) decode(input []byte) error {
	err := json.Unmarshal(input, ps)
	return err
}

func (ps *poolStats) addStat(p *poolStat) {
	ps.Stats = append(ps.Stats, p)
}

type poolStat struct {
	ID        datastore.Key `json:"pool_id"`
	StartTime string        `json:"start_time"`
	Duartion  string        `json:duration`
	TimeLeft  string        `json:"time_left"`
	Locked    bool          `json:"locked"`
	PoolType  int           `json:"pool_type"`
	Balance   state.Balance `json:"balance"`
}

func (ps *poolStat) encode() []byte {
	buff, _ := json.Marshal(ps)
	return buff
}

func (ps *poolStat) decode(input []byte) error {
	err := json.Unmarshal(input, ps)
	return err
}

type tokenLock struct {
	StartTime common.Timestamp `json:"start_time"`
	Duration  time.Duration    `json:"duration"`
	Owner     datastore.Key    `json:"owner"`
	// for future use
	// Leaser          datastore.Key   `json:"leaser"`
	// LockExecutors   []datastore.Key `json:"lock_executors"`
	// PayoutExecutors []datastore.Key `json:"payout_executors"`
}

func (tl tokenLock) IsLocked(entity interface{}) bool {
	tm, ok := entity.(time.Time)
	if ok {
		return tm.Sub(common.ToTime(tl.StartTime)) < tl.Duration
	}
	return true
}

func (tl tokenLock) LockStats(entity interface{}) []byte {
	tm, ok := entity.(time.Time)
	if ok {
		p := &poolStat{StartTime: common.ToTime(tl.StartTime).String(), Duartion: tl.Duration.String(), TimeLeft: (tl.Duration - tm.Sub(common.ToTime(tl.StartTime))).String(), Locked: tl.IsLocked(tm)}
		return p.encode()
	}
	return nil
}
