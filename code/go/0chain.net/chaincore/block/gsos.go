package block

import (
	"encoding/hex"
	"encoding/json"
	"sort"
	"sync"

	"github.com/0chain/0chain/code/go/0chain.net/core/encryption"
	"github.com/0chain/0chain/code/go/0chain.net/core/util"
)

type GroupSharesOrSigns struct {
	mutex  sync.RWMutex
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
