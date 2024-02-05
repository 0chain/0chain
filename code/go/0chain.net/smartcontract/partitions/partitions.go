package partitions

import (
	"math/rand"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"github.com/0chain/common/core/util"
)

const (
	ErrItemNotFoundCode = "item not found"
	ErrItemExistCode    = "item already exist"
)

// ErrItemNotFound checks if error is common.Error and code is 'item not found'
func ErrItemNotFound(err error) bool {
	cErr, ok := err.(*common.Error)
	if !ok {
		return false
	}

	return cErr.Code == ErrItemNotFoundCode
}

// ErrItemExist checks if error is common.Error and code is 'item already exist'
func ErrItemExist(err error) bool {
	cErr, ok := err.(*common.Error)
	if !ok {
		return false
	}

	return cErr.Code == ErrItemExistCode
}

type Partitions interface {
	Add(state state.StateContextI, item PartitionItem) error
	Get(state state.StateContextI, id string, v PartitionItem) (int, error)
	UpdateItem(state state.StateContextI, it PartitionItem) error
	Update(state state.StateContextI, key string, f func(data []byte) ([]byte, error)) (int, error)
	Remove(state state.StateContextI, id string) error
	GetRandomItems(state state.StateContextI, r *rand.Rand, vs interface{}) error
	Size(state state.StateContextI) (int, error)
	Exist(state state.StateContextI, id string) (bool, error)
	Save(state state.StateContextI) error
	MarshalMsg(o []byte) ([]byte, error)
	UnmarshalMsg(b []byte) ([]byte, error)
	Msgsize() int
}

type PartitionItem interface {
	util.MPTSerializableSize
	GetID() string
}
