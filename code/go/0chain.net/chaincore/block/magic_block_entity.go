package block

import (
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"sync"

	"0chain.net/chaincore/node"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"
)

//go:generate msgp -io=false -tests=false -v

// swagger:model MagicBlock
type MagicBlock struct {
	datastore.HashIDField
	mutex                  sync.RWMutex        `json:"-" msgpack:"-" msg:"-"`
	PreviousMagicBlockHash string              `json:"previous_hash"`
	MagicBlockNumber       int64               `json:"magic_block_number"`
	StartingRound          int64               `json:"starting_round"`
	Miners                 *node.Pool          `json:"miners"`   //this is the pool of miners participating in the blockchain
	Sharders               *node.Pool          `json:"sharders"` //this is the pool of sharders participaing in the blockchain
	ShareOrSigns           *GroupSharesOrSigns `json:"share_or_signs"`
	Mpks                   *Mpks               `json:"mpks"`
	T                      int                 `json:"t"`
	K                      int                 `json:"k"`
	N                      int                 `json:"n"`
}

func NewMagicBlock() *MagicBlock {
	return &MagicBlock{Mpks: NewMpks(), ShareOrSigns: NewGroupSharesOrSigns()}
}

func (mb *MagicBlock) GetShareOrSigns() *GroupSharesOrSigns {
	mb.mutex.RLock()
	defer mb.mutex.RUnlock()
	return mb.ShareOrSigns
}

func (mb *MagicBlock) SetShareOrSigns(gsos *GroupSharesOrSigns) {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()
	mb.ShareOrSigns = gsos
}

func (mb *MagicBlock) Encode() []byte {
	buff, _ := json.Marshal(mb)
	return buff
}

func (mb *MagicBlock) Decode(input []byte) error {
	return json.Unmarshal(input, mb)
}

func (mb *MagicBlock) GetHash() string {
	return util.ToHex(mb.GetHashBytes())
}

func (mb *MagicBlock) GetHashBytes() []byte {
	data := []byte(strconv.FormatInt(mb.MagicBlockNumber, 10))
	data = append(data, []byte(mb.PreviousMagicBlockHash)...)
	data = append(data, []byte(strconv.FormatInt(mb.StartingRound, 10))...)
	var minerKeys, sharderKeys, mpkKeys []string
	// miner info
	minerKeys = mb.Miners.Keys()
	sort.Strings(minerKeys)
	for _, v := range minerKeys {
		data = append(data, []byte(v)...)
	}
	// sharder info
	sharderKeys = mb.Sharders.Keys()
	sort.Strings(sharderKeys)
	for _, v := range sharderKeys {
		data = append(data, []byte(v)...)
	}
	// share info
	shareBytes, _ := hex.DecodeString(mb.GetShareOrSigns().GetHash())
	data = append(data, shareBytes...)
	// mpk info
	for k := range mb.Mpks.Mpks {
		mpkKeys = append(mpkKeys, k)
	}
	sort.Strings(mpkKeys)
	for _, v := range mpkKeys {
		data = append(data, []byte(v)...)
	}
	data = append(data, []byte(strconv.Itoa(mb.T))...)
	data = append(data, []byte(strconv.Itoa(mb.N))...)
	return encryption.RawHash(data)
}

func (mb *MagicBlock) IsActiveNode(id string, round int64) bool {
	if mb == nil || mb.Miners == nil || mb.Sharders == nil {
		return false
	}
	if mb.Miners.HasNode(id) {
		return mb.StartingRound <= round
	}
	return mb.Sharders.HasNode(id) && mb.StartingRound <= round
}

func (mb *MagicBlock) VerifyMinersSignatures(b *Block) bool {
	for _, bvt := range b.GetVerificationTickets() {
		var sender = mb.Miners.GetNode(bvt.VerifierID)
		if sender == nil {
			return false
		}
		if ok, _ := sender.Verify(bvt.Signature, b.Hash); !ok {
			return false
		}
	}
	return true
}

// Clone returns a clone of MagicBlock instance
func (mb *MagicBlock) Clone() *MagicBlock {
	mb.mutex.RLock()
	defer mb.mutex.RUnlock()
	clone := &MagicBlock{
		HashIDField:            mb.HashIDField,
		PreviousMagicBlockHash: mb.PreviousMagicBlockHash,
		MagicBlockNumber:       mb.MagicBlockNumber,
		StartingRound:          mb.StartingRound,
		T:                      mb.T,
		K:                      mb.K,
		N:                      mb.N,
	}

	if mb.ShareOrSigns != nil {
		clone.ShareOrSigns = mb.ShareOrSigns.Clone()
	}
	if mb.Mpks != nil {
		clone.Mpks = mb.Mpks.Clone()
	}
	if mb.Miners != nil {
		clone.Miners = mb.Miners.Clone()
	}
	if mb.Sharders != nil {
		clone.Sharders = mb.Sharders.Clone()
	}

	return clone
}
