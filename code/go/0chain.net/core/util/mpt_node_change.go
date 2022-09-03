package util

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"runtime/debug"
	"sync"

	"github.com/0chain/common/core/logging"
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
	GetStartRoot() Key

	UpdateChanges(ndb NodeDB, origin Sequence, includeDeletes bool) error

	Validate() error
	Clone() ChangeCollectorI
}

/*ChangeCollector - node change collector interface implementation */
type ChangeCollector struct {
	startRoot Key
	Changes   map[string]*NodeChange
	Deletes   map[string]Node
	mutex     sync.RWMutex
}

/*NewChangeCollector - a constructor to create a change collector */
func NewChangeCollector(startRoot Key) ChangeCollectorI {
	cc := &ChangeCollector{startRoot: startRoot}
	cc.Changes = make(map[string]*NodeChange)
	cc.Deletes = make(map[string]Node)
	return cc
}

func (cc *ChangeCollector) GetStartRoot() Key {
	return cc.startRoot
}

/*AddChange - implement interface */
func (cc *ChangeCollector) AddChange(oldNode Node, newNode Node) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	nhash := newNode.GetHash()
	delete(cc.Deletes, nhash)
	if oldNode == nil {
		change := &NodeChange{}
		change.New = newNode
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
		prevChange.New = newNode
		cc.Changes[nhash] = prevChange
	} else {
		change := &NodeChange{}
		change.New = newNode
		change.Old = oldNode
		cc.Changes[nhash] = change
		cc.Deletes[ohash] = oldNode
	}
}

/*DeleteChange - implement interface */
func (cc *ChangeCollector) DeleteChange(oldNode Node) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	ohash := oldNode.GetHash()
	if _, ok := cc.Changes[ohash]; ok {
		if DebugMPTNode {
			logging.Logger.Debug("DeleteChange existing change",
				zap.String("ohash", ohash),
				zap.String("stack", string(debug.Stack())),
			)
		}
		delete(cc.Changes, ohash)
	} else {
		if DebugMPTNode {
			logging.Logger.Debug("DeleteChange adding to deletes",
				zap.String("ohash", ohash),
				zap.String("stack", string(debug.Stack())),
			)
		}
		cc.Deletes[ohash] = oldNode
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
	deletes := make([]Node, 0, len(cc.Deletes))
	for _, v := range cc.Deletes {
		deletes = append(deletes, v)
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
		// use old key as UpdateVersion would not change the key even the node has been updated
		keys[idx] = c.New.GetHashBytes()
		if origin != c.New.GetOrigin() {
			c.New.SetOrigin(origin)
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
	// TODO: make the calling of Flush() configurable, and
	// call it on production env.
	//if pndb, ok := ndb.(*PNodeDB); ok {
	//	pndb.Flush()
	//}
	//logging.Logger.Debug("update changes - flushed", zap.Duration("duration", time.Since(ts)))
	return nil
}

func PrintChanges(w io.Writer, changes []*NodeChange) {
	for idx, c := range changes {
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
