package util

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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
}

/*ChangeCollector - node change collector interface implementation */
type ChangeCollector struct {
	Changes map[string]*NodeChange
	Deletes map[string]Node
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
	nhash := newNode.GetHash()
	if _, ok := cc.Deletes[nhash]; ok {
		delete(cc.Deletes, nhash)
	}
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
			if bytes.Compare(newNode.GetHashBytes(), prevChange.Old.GetHashBytes()) == 0 {
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
	ohash := oldNode.GetHash()
	if _, ok := cc.Changes[ohash]; ok {
		delete(cc.Changes, ohash)
	} else {
		cc.Deletes[ohash] = oldNode
	}
}

/*GetChanges - implement interface */
func (cc *ChangeCollector) GetChanges() []*NodeChange {
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
	keys := make([]Key, len(cc.Changes))
	nodes := make([]Node, len(cc.Changes))
	idx := 0
	for _, c := range cc.Changes {
		c.New.SetOrigin(origin)
		keys[idx] = c.New.GetHashBytes()
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
	for key := range cc.Changes {
		if _, ok := cc.Deletes[key]; ok {
			return errors.New("key present in both add and delete")
		}
	}
	return nil
}
