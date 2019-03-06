package interestpoolsc

import (
	"encoding/json"
	"time"

	"0chain.net/chaincore/smartcontractstate"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

const (
	INTEREST = 0
	STAKE    = 1
)

type simpleGlobalNode struct {
	MinLock      int64   `json:"min_lock"`
	InterestRate float64 `json:"interest_rate"`
}

func (sgn *simpleGlobalNode) encode() []byte {
	buff, _ := json.Marshal(sgn)
	return buff
}

func (sgn *simpleGlobalNode) decode(input []byte) error {
	err := json.Unmarshal(input, sgn)
	return err
}

type globalNode struct {
	*simpleGlobalNode `json:"simple_global_node"`
	LockPeriod        time.Duration `json:"lock_period"`
}

func newGlobalNode() *globalNode {
	return &globalNode{simpleGlobalNode: &simpleGlobalNode{}}
}

func (gn *globalNode) encode() []byte {
	buff, _ := json.Marshal(gn)
	return buff
}

func (gn *globalNode) decode(input []byte) error {
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(input, &objMap)
	if err != nil {
		return err
	}
	sgn, ok := objMap["simple_global_node"]
	if ok {
		err = gn.simpleGlobalNode.decode(*sgn)
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

func (gn *globalNode) getKey() smartcontractstate.Key {
	return smartcontractstate.Key("interest_sc_global_node")
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

type userNode struct {
	ClientID datastore.Key               `json:"client_id"`
	Pools    map[datastore.Key]*typePool `json:"pools"`
}

func newUserNode(clientID datastore.Key) *userNode {
	un := &userNode{ClientID: clientID}
	un.Pools = make(map[datastore.Key]*typePool)
	return un
}

func (un *userNode) encode() []byte {
	buff, _ := json.Marshal(un)
	return buff
}

func (un *userNode) decode(input []byte) error {
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

func (un *userNode) getKey() smartcontractstate.Key {
	return smartcontractstate.Key("interest_sc_user" + Seperator + un.ClientID)
}

func (un *userNode) hasPool(poolID datastore.Key) bool {
	pool := un.Pools[poolID]
	return pool != nil
}

func (un *userNode) getPool(poolID datastore.Key) *typePool {
	return un.Pools[poolID]
}

func (un *userNode) addPool(ip *typePool) error {
	if un.hasPool(ip.ID) {
		return common.NewError("can't add pool", "user node already has pool")
	}
	un.Pools[ip.ID] = ip
	return nil
}

func (un *userNode) deletePool(poolID datastore.Key) error {
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
	txn, ok := entity.(*transaction.Transaction)
	if ok {
		return common.ToTime(txn.CreationDate).Sub(common.ToTime(tl.StartTime)) < tl.Duration
	}
	return true
}

func (tl tokenLock) LockStats(entity interface{}) []byte {
	txn, ok := entity.(*transaction.Transaction)
	if ok {
		p := &poolStat{StartTime: common.ToTime(tl.StartTime).String(), Duartion: tl.Duration.String(), TimeLeft: (tl.Duration - common.ToTime(txn.CreationDate).Sub(common.ToTime(tl.StartTime))).String(), Locked: tl.IsLocked(txn)}
		return p.encode()
	}
	return nil
}
