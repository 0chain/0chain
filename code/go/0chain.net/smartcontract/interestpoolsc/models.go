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

type SimpleGlobalNode struct {
	MinLock int64   `json:"min_lock"`
	APR     float64 `json:"apr"`
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
	MinLockPeriod     time.Duration `json:"min_lock_period"`
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
	var min string
	minlp, ok := objMap["min_lock_period"]
	if ok {
		err = json.Unmarshal(*minlp, &min)
		if err != nil {
			return err
		}
		dur, err := time.ParseDuration(min)
		if err != nil {
			return err
		}
		gn.MinLockPeriod = dur
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

type newPoolRequest struct {
	Duration time.Duration `json:"duration"`
}

func (npr *newPoolRequest) encode() []byte {
	buff, _ := json.Marshal(npr)
	return buff
}

func (npr *newPoolRequest) decode(input []byte) error {
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(input, &objMap)
	if err != nil {
		return err
	}
	var d string
	duration, ok := objMap["duration"]
	if ok {
		err = json.Unmarshal(*duration, &d)
		if err != nil {
			return err
		}
		dur, err := time.ParseDuration(d)
		if err != nil {
			return err
		}
		npr.Duration = dur
	}
	return nil
}

type interestPool struct {
	*tokenpool.ZcnLockingPool `json:"pool"`
	APR                       float64 `json:"apr"`
	TokensEarned              int64   `json:"tokens_earned"`
}

func newInterestPool() *interestPool {
	return &interestPool{ZcnLockingPool: &tokenpool.ZcnLockingPool{}}
}

func (ip *interestPool) encode() []byte {
	buff, _ := json.Marshal(ip)
	return buff
}

func (ip *interestPool) decode(input []byte) error {
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(input, &objMap)
	if err != nil {
		return err
	}
	ir, ok := objMap["apr"]
	if ok {
		var rate float64
		err = json.Unmarshal(*ir, &rate)
		if err != nil {
			return err
		}
		ip.APR = rate
	}
	ie, ok := objMap["tokens_earned"]
	if ok {
		var earned int64
		err = json.Unmarshal(*ie, &earned)
		if err != nil {
			return err
		}
		ip.TokensEarned = earned
	}
	p, ok := objMap["pool"]
	if ok {
		err = ip.ZcnLockingPool.Decode(*p, &tokenLock{})
		if err != nil {
			return err
		}
	}
	return nil
}

type UserNode struct {
	ClientID datastore.Key                   `json:"client_id"`
	Pools    map[datastore.Key]*interestPool `json:"pools"`
}

func newUserNode(clientID datastore.Key) *UserNode {
	un := &UserNode{ClientID: clientID}
	un.Pools = make(map[datastore.Key]*interestPool)
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
			tempPool := newInterestPool()
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

func (un *UserNode) getPool(poolID datastore.Key) *interestPool {
	return un.Pools[poolID]
}

func (un *UserNode) addPool(ip *interestPool) error {
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
	ID           datastore.Key `json:"pool_id"`
	StartTime    string        `json:"start_time"`
	Duartion     string        `json:"duration"`
	TimeLeft     string        `json:"time_left"`
	Locked       bool          `json:"locked"`
	APR          float64       `json:"apr"`
	TokensEarned int64         `json:"tokens_earned"`
	Balance      state.Balance `json:"balance"`
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
