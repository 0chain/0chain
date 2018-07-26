package util

/*NodeChange - track a change to the node */
type NodeChange struct {
	Old Node
	New Node
}

/*ChangeCollectorI - an interface to collect node changes */
type ChangeCollectorI interface {
	SetRoot(key Key)
	GetRoot() Key
	AddChange(oldNode Node, newNode Node)
	DeleteChange(oldNode Node)
	GetChanges() []*NodeChange
	GetDeletes() []Node

	UpdateChanges(ndb NodeDB, origin Origin, includeDeletes bool) error
}

/*ChangeCollector - node change collector interface implementation */
type ChangeCollector struct {
	Root    Key
	Changes map[Node]Node
	Deletes []Node
}

/*NewChangeCollector - a constructor to create a change collector */
func NewChangeCollector() ChangeCollectorI {
	cc := &ChangeCollector{}
	cc.Changes = make(map[Node]Node)
	return cc
}

/*SetRoot - implement interface */
func (cc *ChangeCollector) SetRoot(root Key) {
	cc.Root = root
}

/*GetRoot - implement interface */
func (cc *ChangeCollector) GetRoot() Key {
	return cc.Root
}

/*AddChange - implement interface */
func (cc *ChangeCollector) AddChange(oldNode Node, newNode Node) {
	if oldNode == nil {
		cc.Changes[newNode] = nil
		return
	}
	prevOldNode, ok := cc.Changes[oldNode]
	if ok {
		delete(cc.Changes, oldNode)
		cc.Changes[newNode] = prevOldNode
	} else {
		cc.Changes[newNode] = oldNode
	}
}

/*DeleteChange - implement interface */
func (cc *ChangeCollector) DeleteChange(oldNode Node) {
	cc.Deletes = append(cc.Deletes, oldNode)
}

/*GetChanges - implement interface */
func (cc *ChangeCollector) GetChanges() []*NodeChange {
	changes := make([]*NodeChange, len(cc.Changes))
	idx := 0
	for k, v := range cc.Changes {
		changes[idx] = &NodeChange{Old: v, New: k}
		idx++
	}
	return changes
}

/*GetDeletes - implement interface */
func (cc *ChangeCollector) GetDeletes() []Node {
	return cc.Deletes
}

/*UpdateChanges - update all the changes collected to a database */
func (cc *ChangeCollector) UpdateChanges(ndb NodeDB, origin Origin, includeDeletes bool) error {
	// TODO: it's possible to do batch changes instead of individual changes for PNodeDB
	for u := range cc.Changes {
		u.SetOrigin(origin)
		err := ndb.PutNode(u.GetHashBytes(), u)
		if err != nil {
			return err
		}
	}
	if !includeDeletes {
		return nil
	}
	for _, d := range cc.Deletes {
		err := ndb.DeleteNode(d.GetHashBytes())
		if err != nil {
			return err
		}
	}
	if pndb, ok := ndb.(*PNodeDB); ok {
		pndb.Flush()
	}
	return nil
}
