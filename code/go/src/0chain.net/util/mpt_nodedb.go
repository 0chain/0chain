package util

import (
	"context"
	"errors"

	"0chain.net/config"
	. "0chain.net/logging"
	"go.uber.org/zap"
)

//BatchSize - for batching multiple db operations
const BatchSize = 256

/*ErrNodeNotFound - error indicating that the node is not found */
var ErrNodeNotFound = errors.New("node not found")

/*ErrValueNotPresent - error indicating given path is not present in the db */
var ErrValueNotPresent = errors.New("value not present")

/*NodeDBIteratorHandler is a nodedb iteration handler function type */
type NodeDBIteratorHandler func(ctx context.Context, key Key, node Node) error

/*NodeDB - an interface that gets, puts and deletes nodes by their key */
type NodeDB interface {
	GetNode(key Key) (Node, error)
	PutNode(key Key, node Node) error
	DeleteNode(key Key) error
	Iterate(ctx context.Context, handler NodeDBIteratorHandler) error
	Size(ctx context.Context) int64

	MultiPutNode(keys []Key, nodes []Node) error
	MultiDeleteNode(keys []Key) error

	PruneBelowVersion(ctx context.Context, version Sequence) error
}

/*StrKey - data type for the key used to store the node into some storage (this is needed as hashmap keys can't be []byte */
type StrKey string

/*MemoryNodeDB - an inmemory node db */
type MemoryNodeDB struct {
	Nodes map[StrKey]Node
}

/*NewMemoryNodeDB - create a memory node db */
func NewMemoryNodeDB() *MemoryNodeDB {
	mndb := &MemoryNodeDB{}
	mndb.Nodes = make(map[StrKey]Node)
	return mndb
}

/*GetNode - implement interface */
func (mndb *MemoryNodeDB) GetNode(key Key) (Node, error) {
	skey := StrKey(key)
	node, ok := mndb.Nodes[skey]
	if !ok {
		return nil, ErrNodeNotFound
	}
	return node, nil
}

/*PutNode - implement interface */
func (mndb *MemoryNodeDB) PutNode(key Key, node Node) error {
	skey := StrKey(key)
	mndb.Nodes[skey] = node
	return nil
}

/*DeleteNode - implement interface */
func (mndb *MemoryNodeDB) DeleteNode(key Key) error {
	skey := StrKey(key)
	delete(mndb.Nodes, skey)
	return nil
}

/*MultiPutNode - implement interface */
func (mndb *MemoryNodeDB) MultiPutNode(keys []Key, nodes []Node) error {
	for idx, key := range keys {
		err := mndb.PutNode(key, nodes[idx])
		if err != nil {
			return err
		}
	}
	return nil
}

/*MultiDeleteNode - implement interface */
func (mndb *MemoryNodeDB) MultiDeleteNode(keys []Key) error {
	for _, key := range keys {
		err := mndb.DeleteNode(key)
		if err != nil {
			return err
		}
	}
	return nil
}

/*Iterate - implement interface */
func (mndb *MemoryNodeDB) Iterate(ctx context.Context, handler NodeDBIteratorHandler) error {
	for key, node := range mndb.Nodes {
		err := handler(ctx, Key(key), node)
		if err != nil {
			return err
		}
	}
	return nil
}

/*Size - implement interface */
func (mndb *MemoryNodeDB) Size(ctx context.Context) int64 {
	return int64(len(mndb.Nodes))
}

/*PruneBelowVersion - implement interface */
func (mndb *MemoryNodeDB) PruneBelowVersion(ctx context.Context, version Sequence) error {
	for key, node := range mndb.Nodes {
		if node.GetVersion() < version {
			delete(mndb.Nodes, key)
		}
	}
	return nil
}

func (mndb *MemoryNodeDB) reachable(node Node, node2 Node) (bool, error) {
	switch nodeImpl := node.(type) {
	case *ExtensionNode:
		fn, _ := mndb.GetNode(nodeImpl.NodeKey)
		if fn == nil {
			return false, ErrNodeNotFound
		}
		if node2 == fn {
			return true, nil
		}
		return mndb.reachable(fn, node2)
	case *FullNode:
		for i := byte(0); i < 16; i++ {
			child := nodeImpl.GetChild(nodeImpl.indexToByte(i))
			if child == nil {
				continue
			}
			childNode, err := mndb.GetNode(child)
			if err != nil {
				continue
			}
			if node2 == childNode {
				return true, nil
			}
			ok, err := mndb.reachable(childNode, node2)
			if ok {
				return ok, nil
			}
		}
	}
	return false, nil
}

/*ComputeRoot - compute root from partial set of nodes in this db */
func (mndb *MemoryNodeDB) ComputeRoot() Node {
	var root Node
	handler := func(ctx context.Context, key Key, node Node) error {
		if !IncludesNodeType(NodeTypeFullNode|NodeTypeExtensionNode, node.GetNodeType()) {
			return nil
		}
		if root == nil {
			root = node
			return nil
		}
		ok, err := mndb.reachable(root, node)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		ok, err = mndb.reachable(node, root)
		if err != nil {
			return err
		}
		if ok {
			root = node
		}
		return nil
	}
	mndb.Iterate(nil, handler)
	return root
}

/*LevelNodeDB - a multi-level node db. It has a current node db and a previous node db. This is useful to update without changing the previous db. */
type LevelNodeDB struct {
	C                NodeDB
	P                NodeDB
	PropagateDeletes bool // Setting this to false (default) will not propagate delete to lower level db
	DeletedNodes     map[StrKey]bool
}

/*NewLevelNodeDB - create a level node db */
func NewLevelNodeDB(curNDB NodeDB, prevNDB NodeDB, propagateDeletes bool) *LevelNodeDB {
	lndb := &LevelNodeDB{C: curNDB, P: prevNDB, PropagateDeletes: propagateDeletes}
	lndb.DeletedNodes = make(map[StrKey]bool)
	return lndb
}

func (lndb *LevelNodeDB) isCurrentPersistent() bool {
	_, ok := lndb.C.(*PNodeDB)
	return ok
}

/*GetNode - implement interface */
func (lndb *LevelNodeDB) GetNode(key Key) (Node, error) {
	c := lndb.C
	p := lndb.P
	node, err := c.GetNode(key)
	if err != nil && p != c {
		node, err = p.GetNode(key)
		if err != nil {
			if config.DevConfiguration.State {
				Logger.Error("get node", zap.String("key", ToHex(key)), zap.Error(err))
			}
		}
		return node, err
	}
	return node, nil
}

/*PutNode - implement interface */
func (lndb *LevelNodeDB) PutNode(key Key, node Node) error {
	return lndb.C.PutNode(key, node)
}

/*DeleteNode - implement interface */
func (lndb *LevelNodeDB) DeleteNode(key Key) error {
	c := lndb.C
	p := lndb.P
	_, err := c.GetNode(key)
	if err != nil {
		if lndb.PropagateDeletes && p != c {
			return p.DeleteNode(key)
		}
		skey := StrKey(key)
		lndb.DeletedNodes[skey] = true
		return nil
	}
	return c.DeleteNode(key)
}

/*MultiPutNode - implement interface */
func (lndb *LevelNodeDB) MultiPutNode(keys []Key, nodes []Node) error {
	for idx, key := range keys {
		err := lndb.PutNode(key, nodes[idx])
		if err != nil {
			return err
		}
	}
	return nil
}

/*MultiDeleteNode - implement interface */
func (lndb *LevelNodeDB) MultiDeleteNode(keys []Key) error {
	for _, key := range keys {
		err := lndb.DeleteNode(key)
		if err != nil {
			return err
		}
	}
	return nil
}

/*Iterate - implement interface */
func (lndb *LevelNodeDB) Iterate(ctx context.Context, handler NodeDBIteratorHandler) error {
	c := lndb.C
	p := lndb.P
	err := c.Iterate(ctx, handler)
	if err != nil {
		return err
	}
	if p != c && !lndb.isCurrentPersistent() {
		return p.Iterate(ctx, handler)
	}
	return nil
}

/*Size - implement interface */
func (lndb *LevelNodeDB) Size(ctx context.Context) int64 {
	c := lndb.C
	p := lndb.P
	size := c.Size(ctx)
	if p != c {
		size += p.Size(ctx)
	}
	return size
}

/*PruneBelowVersion - implement interface */
func (lndb *LevelNodeDB) PruneBelowVersion(ctx context.Context, version Sequence) error {
	// TODO
	return nil
}

/*RebaseCurrentDB - set the current database */
func (lndb *LevelNodeDB) RebaseCurrentDB(ndb NodeDB) {
	lndb.C = ndb
	lndb.P = lndb.C
}
