package util

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/0chain/errors"

	"0chain.net/core/logging"
	"go.uber.org/zap"
)

/*NodeChange - track a change to the node */
type NodeChange struct {
	Old Node
	New Node
}

/*ChangeCollectorI - an interface to collect node changes */
type ChangeCollectorI interface {
	AddChange(oldNode Node, newNode Node)
	DeleteChange(oldNode Node)
	GetChanges() []*NodeChange
	GetDeletes() []Node

	UpdateChanges(ndb NodeDB, origin Sequence, includeDeletes bool) error

	PrintChanges(w io.Writer)

	Validate() error
	Clone() ChangeCollectorI
}

/*ChangeCollector - node change collector interface implementation */
type ChangeCollector struct {
	Changes map[string]*NodeChange
	Deletes map[string]Node
	mutex   sync.RWMutex
}

/*NewChangeCollector - a constructor to create a change collector */
func NewChangeCollector() ChangeCollectorI {
	cc := &ChangeCollector{}
	cc.Changes = make(map[string]*NodeChange)
	cc.Deletes = make(map[string]Node)
	return cc
}

/*AddChange - implement interface */
func (cc *ChangeCollector) AddChange(oldNode Node, newNode Node) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	nhash := newNode.GetHash()
	delete(cc.Deletes, nhash)
	if oldNode == nil {
		change := &NodeChange{}
		change.New = newNode.Clone()
		cc.Changes[nhash] = change
		return
	}
	ohash := oldNode.GetHash()
	prevChange, ok := cc.Changes[ohash]
	if ok {
		delete(cc.Changes, ohash)
		if prevChange.Old != nil {
			if bytes.Equal(newNode.GetHashBytes(), prevChange.Old.GetHashBytes()) {
				return
			}
		}
		prevChange.New = newNode.Clone()
		cc.Changes[nhash] = prevChange
	} else {
		change := &NodeChange{}
		change.New = newNode.Clone()
		change.Old = oldNode.Clone()
		cc.Changes[nhash] = change
		cc.Deletes[ohash] = oldNode.Clone()
	}
}

/*DeleteChange - implement interface */
func (cc *ChangeCollector) DeleteChange(oldNode Node) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	ohash := oldNode.GetHash()
	if _, ok := cc.Changes[ohash]; ok {
		delete(cc.Changes, ohash)
	} else {
		cc.Deletes[ohash] = oldNode.Clone()
	}
}

/*GetChanges - implement interface */
func (cc *ChangeCollector) GetChanges() []*NodeChange {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()
	changes := make([]*NodeChange, len(cc.Changes))
	idx := 0
	for _, v := range cc.Changes {
		changes[idx] = v
		idx++
	}
	return changes
}

/*GetDeletes - implement interface */
func (cc *ChangeCollector) GetDeletes() []Node {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()
	deletes := make([]Node, len(cc.Deletes))
	idx := 0
	for _, v := range cc.Deletes {
		deletes[idx] = v
		idx++
	}
	return deletes
}

/*UpdateChanges - update all the changes collected to a database */
func (cc *ChangeCollector) UpdateChanges(ndb NodeDB, origin Sequence, includeDeletes bool) error {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()
	keys := make([]Key, len(cc.Changes))
	nodes := make([]Node, len(cc.Changes))
	idx := 0
	for _, c := range cc.Changes {
		if _, ok := c.New.(*LeafNode); ok && origin != c.New.GetOrigin() {
			oldHash := c.New.GetHashBytes()
			oldOrigin := c.New.GetOrigin()
			c.New.SetOrigin(origin)
			keys[idx] = c.New.GetHashBytes()
			logging.Logger.Warn("Updating origin of a leaf node may break references ",
				zap.Int64("oldOrigin", int64(oldOrigin)),
				zap.String("oldHash", ToHex(oldHash)),
				zap.Int64("newOrigin", int64(origin)),
				zap.String("newHash", ToHex(keys[idx])),
			)
		} else {
			c.New.SetOrigin(origin)
			keys[idx] = c.New.GetHashBytes()
		}
		nodes[idx] = c.New
		idx++
	}
	err := ndb.MultiPutNode(keys, nodes)
	if err != nil {
		return err
	}
	if includeDeletes {
		for _, d := range cc.Deletes {
			err := ndb.DeleteNode(d.GetHashBytes())
			if err != nil {
				return err
			}
		}
	}
	if len(cc.Changes) == 0 && (!includeDeletes || len(cc.Deletes) == 0) {
		return nil
	}
	if pndb, ok := ndb.(*PNodeDB); ok {
		pndb.Flush()
	}
	return nil
}

//PrintChanges - implement interface
func (cc *ChangeCollector) PrintChanges(w io.Writer) {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()
	for idx, c := range cc.Changes {
		if c.Old != nil {
			fmt.Fprintf(w, "cc(%v): nn=%v on=%v\n", idx, c.New.GetHash(), c.Old.GetHash())
		} else {
			fmt.Fprintf(w, "cc(%v): nn=%v\n", idx, c.New.GetHash())
		}
	}
}

//Validate - validate if this change collector is valid
func (cc *ChangeCollector) Validate() error {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()
	for key := range cc.Changes {
		if _, ok := cc.Deletes[key]; ok {
			return errors.New("key present in both add and delete")
		}
	}
	return nil
}

// Clone returns a copy of the change collector
func (cc *ChangeCollector) Clone() ChangeCollectorI {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()

	c := &ChangeCollector{
		Changes: make(map[string]*NodeChange),
		Deletes: make(map[string]Node),
	}

	for k, v := range cc.Changes {
		change := &NodeChange{
			New: v.New.Clone(),
		}
		if v.Old != nil {
			change.Old = v.Old.Clone()
		}

		c.Changes[k] = change
	}

	for k, v := range cc.Deletes {
		c.Deletes[k] = v.Clone()
	}

	return c
}
