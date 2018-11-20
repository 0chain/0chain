package util

import (
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
}

/*ChangeCollector - node change collector interface implementation */
type ChangeCollector struct {
	Changes map[string]*NodeChange
	Deletes []Node
}

/*NewChangeCollector - a constructor to create a change collector */
func NewChangeCollector() ChangeCollectorI {
	cc := &ChangeCollector{}
	cc.Changes = make(map[string]*NodeChange)
	return cc
}

/*AddChange - implement interface */
func (cc *ChangeCollector) AddChange(oldNode Node, newNode Node) {
	nhash := newNode.GetHash()
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
		prevChange.New = newNode
		cc.Changes[nhash] = prevChange
	} else {
		change := &NodeChange{}
		change.New = newNode
		change.Old = oldNode
		cc.Changes[nhash] = change
	}
}

/*DeleteChange - implement interface */
func (cc *ChangeCollector) DeleteChange(oldNode Node) {
	cc.Deletes = append(cc.Deletes, oldNode)
	ohash := oldNode.GetHash()
	if _, ok := cc.Changes[ohash]; ok {
		delete(cc.Changes, ohash)
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
	return cc.Deletes
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
