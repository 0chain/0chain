package interestpoolsc

import (
	"encoding/json"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

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
