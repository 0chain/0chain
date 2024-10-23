package block

import (
	"encoding/hex"
	"encoding/json"
	"sort"
	"sync"

	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"
)

//go:generate msgp -io=false -tests=false -v

// swagger:model GroupSharesOrSigns
type GroupSharesOrSigns struct {
	mutex  sync.RWMutex             `json:"-" msgpack:"-" msg:"-"`
	Shares map[string]*ShareOrSigns `json:"shares"`
}

func NewGroupSharesOrSigns() *GroupSharesOrSigns {
	return &GroupSharesOrSigns{Shares: make(map[string]*ShareOrSigns)}
}

func (gsos *GroupSharesOrSigns) Get(id string) (*ShareOrSigns, bool) {
	gsos.mutex.RLock()
	defer gsos.mutex.RUnlock()
	share, ok := gsos.Shares[id]
	return share, ok
}

func (gsos *GroupSharesOrSigns) GetShares() map[string]*ShareOrSigns {
	gsos.mutex.RLock()
	defer gsos.mutex.RUnlock()
	result := make(map[string]*ShareOrSigns, len(gsos.Shares))
	for k, v := range gsos.Shares {
		result[k] = v
	}
	return result
}

func (gsos *GroupSharesOrSigns) Delete(id string) {
	gsos.mutex.Lock()
	delete(gsos.Shares, id)
	gsos.mutex.Unlock()
}

func (gsos *GroupSharesOrSigns) Encode() []byte {
	buff, _ := json.Marshal(gsos)
	return buff
}

func (gsos *GroupSharesOrSigns) Decode(input []byte) error {
	return json.Unmarshal(input, gsos)
}

func (gsos *GroupSharesOrSigns) GetHash() string {
	return util.ToHex(gsos.GetHashBytes())
}

func (gsos *GroupSharesOrSigns) GetHashBytes() []byte {
	var data []byte
	var keys []string
	for k := range gsos.Shares {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		bytes, _ := hex.DecodeString(gsos.Shares[k].Hash())
		data = append(data, bytes...)
	}
	return encryption.RawHash(data)
}

// Clone returns a clone of GroupSharesOrSigns instance
func (gsos *GroupSharesOrSigns) Clone() *GroupSharesOrSigns {
	//Shares map[string]*ShareOrSigns `json:"shares"`
	gsos.mutex.RLock()
	defer gsos.mutex.RUnlock()
	clone := &GroupSharesOrSigns{Shares: make(map[string]*ShareOrSigns, len(gsos.Shares))}
	//ShareOrSigns map[string]*bls.DKGKeyShare `json:"share_or_sign"`
	for k, v := range gsos.Shares {
		sos := *v
		sos.ShareOrSigns = make(map[string]*bls.DKGKeyShare, len(v.ShareOrSigns))
		for sk, sv := range v.ShareOrSigns {
			nsv := *sv
			sos.ShareOrSigns[sk] = &nsv
		}
		clone.Shares[k] = &sos
	}

	return clone
}
