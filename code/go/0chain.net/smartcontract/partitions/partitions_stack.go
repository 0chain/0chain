package partitions

import (
	"errors"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/sortedmap"
	"github.com/0chain/common/core/util"
)

//go:generate msgp -io=false -tests=false -v

var ErrEmptyPartition = errors.New("empty partition")

type Stack struct {
	p    *Partitions `msg:"p"`
	head int         `msg:"h"` // head partition location
}

func StackCreateIfNotExist(state state.StateContextI, name string, partitionSize int) (*Stack, error) {
	s := Stack{}
	err := state.GetTrieNode(name, &s)
	switch err {
	case nil:
		return &s, nil
	case util.ErrValueNotPresent:
		p, err := newPartitions(name, partitionSize)
		if err != nil {
			return nil, err
		}
		s.p = p
		if err := s.Save(state); err != nil {
			return nil, err
		}

		return &s, nil
	default:
		return nil, err
	}
}

func (s *Stack) Push(state state.StateContextI, item PartitionItem) error {
	return s.p.Add(state, item)
}

func (s *Stack) Pop(state state.StateContextI, item PartitionItem) error {
	pt, err := s.p.getPartition(state, s.head)
	if err != nil {
		return err
	}

	if len(pt.Items) == 0 {
		return ErrEmptyPartition
	}

	pt.Items = pt.Items[1:]
	pt.Changed = true

	if len(pt.Items) == 0 {
		if _, err := state.DeleteTrieNode(pt.Key); err != nil {
			return err
		}

		s.head++
	}

	return nil
}

func (s *Stack) Save(state state.StateContextI) error {
	keys := sortedmap.NewFromMap(s.p.Partitions).GetKeys()
	for _, k := range keys {
		part := s.p.Partitions[k]
		if part.changed() {
			err := part.save(state)
			if err != nil {
				return err
			}
		}
	}

	_, err := state.InsertTrieNode(s.p.Name, s)
	if err != nil {
		return err
	}
	return nil
}
