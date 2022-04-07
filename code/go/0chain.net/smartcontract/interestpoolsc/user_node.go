package interestpoolsc

import (
	"encoding/json"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

//go:generate msgp -io=false -tests=false -unexported=true -v

// swagger:model UserNode
type UserNode struct {
	ClientID string                   `json:"client_id"`
	Pools    map[string]*InterestPool `json:"pools"`
}

func newUserNode(clientID datastore.Key) *UserNode {
	return &UserNode{
		ClientID: clientID,
		Pools:    make(map[string]*InterestPool),
	}
}

func (un *UserNode) Encode() []byte {
	// encoding client id
	cIdJson, _ := json.Marshal(un.ClientID)
	cIdRW := json.RawMessage(cIdJson)
	// encoding pools
	poolsJson, _ := json.Marshal(un.Pools)
	poolsRW := json.RawMessage(poolsJson)

	buf, _ := json.Marshal(map[string]*json.RawMessage{
		"client_id": &cIdRW,
		"pools":     &poolsRW,
	})
	return buf
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
		un.Pools = make(map[string]*InterestPool, len(rawMessagesPools))
		for _, raw := range rawMessagesPools {
			tempPool := newInterestPool()
			err = tempPool.decode(*raw)
			if err != nil {
				return err
			}
			if err := un.addPool(tempPool); err != nil {
				return err
			}
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

func (un *UserNode) GetKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + un.ClientID)
}

func (un *UserNode) hasPool(poolID datastore.Key) bool {
	pool := un.Pools[poolID]
	return pool != nil
}

func (un *UserNode) getPool(poolID datastore.Key) *InterestPool {
	return un.Pools[poolID]
}

func (un *UserNode) addPool(ip *InterestPool) error {
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
