package util

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"

	"0chain.net/core/common"
	"0chain.net/core/logging"
	"go.uber.org/atomic"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

// common constant
const (
	// BatchSize - for batching multiple db operations.
	BatchSize = 256
	// FmtIntermediateNodeExists - error format indicating deleted intermediate
	// node still exists.
	FmtIntermediateNodeExists = "removed intermediate node still present" +
		" (%T) %v - new (%T) %v"
)

// common errors
var (
	// ErrNodeNotFound - error indicating that the node is not found.
	ErrNodeNotFound = errors.New("node not found")
	// ErrValueNotPresent - error indicating given path is not present in the
	// db.
	ErrValueNotPresent = errors.New("value not present")
	// ErrNilPartialStateRoot is returned when partialState.ComputeRoot() gets nil root
	ErrNilPartialStateRoot = errors.New("partial state root is nil")
)

// global node db version
var levelNodeVersion atomic.Int64

/*NodeDBIteratorHandler is a nodedb iteration handler function type */
type NodeDBIteratorHandler func(ctx context.Context, key Key, node Node) error

/*NodeDB - an interface that gets, puts and deletes nodes by their key */
type NodeDB interface {
	GetNode(key Key) (Node, error)
	PutNode(key Key, node Node) error
	DeleteNode(key Key) error
	Iterate(ctx context.Context, handler NodeDBIteratorHandler) error
	Size(ctx context.Context) int64

	MultiGetNode(keys []Key) ([]Node, error)
	MultiPutNode(keys []Key, nodes []Node) error
	MultiDeleteNode(keys []Key) error
}

// StrKey - data type for the key used to store the node into some storage
// (this is needed as hashmap keys can't be []byte.
type StrKey string

// MemoryNodeDB - an inmemory node db.
type MemoryNodeDB struct {
	Nodes map[StrKey]Node
	mutex *sync.RWMutex
}

// NewMemoryNodeDB - create a memory node db.
func NewMemoryNodeDB() *MemoryNodeDB {
	mndb := &MemoryNodeDB{}
	mndb.Nodes = make(map[StrKey]Node)
	mndb.mutex = &sync.RWMutex{}
	return mndb
}

// GetDBVersions, not implemented
func (mndb *MemoryNodeDB) GetDBVersions() []int64 {
	return []int64{}
}

// unsafe
func (mndb *MemoryNodeDB) getNode(key Key) (Node, error) {
	node, ok := mndb.Nodes[StrKey(key)]
	if !ok {
		return nil, ErrNodeNotFound
	}
	return node, nil
}

// unsafe
func (mndb *MemoryNodeDB) putNode(key Key, node Node) error {
	if DebugMPTNode {
		logging.Logger.Debug("node put to memory", zap.String("key", ToHex(key)),
			zap.String("stack", string(debug.Stack())),
		)
	}
	mndb.Nodes[StrKey(key)] = node
	return nil
}

// unsafe
func (mndb *MemoryNodeDB) deleteNode(key Key) error {
	if DebugMPTNode {
		logging.Logger.Debug("node delete from memory", zap.String("key", ToHex(key)),
			zap.String("stack", string(debug.Stack())),
		)
	}
	delete(mndb.Nodes, StrKey(key))
	return nil
}

// unsafe
func (mndb *MemoryNodeDB) iterate(ctx context.Context, handler NodeDBIteratorHandler) error {
	for key, node := range mndb.Nodes {
		err := handler(ctx, Key(key), node)
		if err != nil {
			return err
		}
	}
	return nil
}

/*GetNode - implement interface */
func (mndb *MemoryNodeDB) GetNode(key Key) (Node, error) {
	mndb.mutex.RLock()
	defer mndb.mutex.RUnlock()
	return mndb.getNode(key)
}

/*PutNode - implement interface */
func (mndb *MemoryNodeDB) PutNode(key Key, node Node) error {
	mndb.mutex.Lock()
	defer mndb.mutex.Unlock()
	return mndb.putNode(key, node)
}

/*DeleteNode - implement interface */
func (mndb *MemoryNodeDB) DeleteNode(key Key) error {
	mndb.mutex.Lock()
	defer mndb.mutex.Unlock()
	return mndb.deleteNode(key)
}

/*MultiGetNode - get multiple nodes */
func (mndb *MemoryNodeDB) MultiGetNode(keys []Key) (nodes []Node, err error) {
	mndb.mutex.RLock()
	defer mndb.mutex.RUnlock()
	for _, key := range keys {
		node, nerr := mndb.getNode(key)
		if nerr != nil {
			err = nerr
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes, err
}

/*MultiPutNode - implement interface */
func (mndb *MemoryNodeDB) MultiPutNode(keys []Key, nodes []Node) error {
	mndb.mutex.Lock()
	defer mndb.mutex.Unlock()
	for idx, key := range keys {
		err := mndb.putNode(key, nodes[idx])
		if err != nil {
			return err
		}
	}
	return nil
}

/*MultiDeleteNode - implement interface */
func (mndb *MemoryNodeDB) MultiDeleteNode(keys []Key) error {
	mndb.mutex.Lock()
	defer mndb.mutex.Unlock()
	for _, key := range keys {
		err := mndb.deleteNode(key)
		if err != nil {
			return err
		}
	}
	return nil
}

/*Iterate - implement interface */
func (mndb *MemoryNodeDB) Iterate(ctx context.Context, handler NodeDBIteratorHandler) error {
	mndb.mutex.RLock()
	defer mndb.mutex.RUnlock()
	return mndb.iterate(ctx, handler)
}

// Size - implement interface
func (mndb *MemoryNodeDB) Size(_ context.Context) int64 {
	mndb.mutex.RLock()
	defer mndb.mutex.RUnlock()
	return int64(len(mndb.Nodes))
}

/*PruneBelowVersion - implement interface */
func (mndb *MemoryNodeDB) PruneBelowVersion(ctx context.Context, version Sequence) error {
	mndb.mutex.Lock()
	defer mndb.mutex.Unlock()
	return mndb.iterate(ctx, func(ctx context.Context, key Key, node Node) error {
		if node.GetVersion() < version {
			return mndb.deleteNode(key)
		}
		return nil
	})
}

func (mndb *MemoryNodeDB) RecordDeadNodes(nodes []Node) (int, error) {
	return 0, nil
}

// unsafe
func (mndb *MemoryNodeDB) reachable(node, node2 Node) (ok bool) {
	switch nodeImpl := node.(type) {
	case *ExtensionNode:
		fn, err := mndb.getNode(nodeImpl.NodeKey)
		if err != nil && err != ErrNodeNotFound {
			panic(err)
		}
		if fn == nil {
			return false
		}
		if node2 == fn {
			return true
		}
		return mndb.reachable(fn, node2)
	case *FullNode:
		for i := byte(0); i < 16; i++ {
			child := nodeImpl.GetChild(nodeImpl.indexToByte(i))
			if child == nil {
				continue
			}
			childNode, err := mndb.getNode(child)
			if err != nil {
				continue
			}
			if node2 == childNode {
				return true
			}
			if mndb.reachable(childNode, node2) {
				return true
			}
		}
	}
	return false
}

// Reachable - checks if Node "to" reachable from Node "from" only using
// nodes stored on this db
func (mndb *MemoryNodeDB) Reachable(from, to Node) (ok bool) {
	mndb.mutex.RLock()
	defer mndb.mutex.RUnlock()
	return mndb.reachable(from, to)
}

// ComputeRoot - compute root from partial set of nodes in this db */
func (mndb *MemoryNodeDB) ComputeRoot() (root Node, err error) {
	mndb.mutex.RLock()
	defer mndb.mutex.RUnlock()
	_ = mndb.iterate(context.TODO(), func(ctx context.Context, key Key, node Node) error {
		if root == nil {
			root = node
			return nil
		}
		if !IncludesNodeType(NodeTypeFullNode|NodeTypeExtensionNode, node.GetNodeType()) {
			return nil
		}
		if mndb.reachable(root, node) {
			return nil
		}
		if mndb.reachable(node, root) {
			root = node
		}
		return nil
	})

	if root == nil {
		return nil, ErrNilPartialStateRoot
	}

	err = mndb.validate(root)
	if err != nil {
		return nil, err
	}

	return root, nil
}

// Validate validates the nodes
func (mndb *MemoryNodeDB) Validate(root Node) error {
	mndb.mutex.RLock()
	defer mndb.mutex.RUnlock()
	return mndb.validate(root)
}

// validate - validate this MemoryNodeDB w.r.t the given root
//  It should not contain any node that can't be reachable from the root.
//  Note: The root itself can reach nodes not present in this db
func (mndb *MemoryNodeDB) validate(root Node) error {
	nodes := map[StrKey]Node{
		StrKey(root.GetHashBytes()): root,
	}
	var iterate func(node Node)
	iterate = func(node Node) {
		switch nodeImpl := node.(type) {
		case *FullNode:
			for _, ckey := range nodeImpl.Children {
				if ckey != nil {
					cnode, err := mndb.getNode(ckey)
					if err == nil {
						nodes[StrKey(ckey)] = cnode
						iterate(cnode)
					}
				}
			}
		case *ExtensionNode:
			ckey := nodeImpl.NodeKey
			cnode, err := mndb.getNode(ckey)
			if err == nil {
				nodes[StrKey(ckey)] = cnode
				iterate(cnode)
			}
		}
	}
	iterate(root)
	return mndb.iterate(context.TODO(), func(ctx context.Context, key Key, node Node) error {
		if _, ok := nodes[StrKey(node.GetHashBytes())]; !ok {
			Logger.Error("mndb validate",
				zap.String("node_type", fmt.Sprintf("%T", node)),
				zap.String("node_key", node.GetHash()))
			return common.NewError("nodes_outside_tree", "not all nodes are from the root")
		}
		return nil
	})
}

// LevelNodeDB - a multi-level node db. It has a current node db and a previous
// node db. This is useful to update without changing the previous db.
type LevelNodeDB struct {
	mutex            *sync.RWMutex
	current          NodeDB
	prev             NodeDB
	PropagateDeletes bool // Setting this to false (default) will not propagate delete to lower level db
	DeletedNodes     map[StrKey]bool
	version          int64
}

// NewLevelNodeDB - create a level node db
func NewLevelNodeDB(curNDB NodeDB, prevNDB NodeDB, propagateDeletes bool) *LevelNodeDB {
	v := levelNodeVersion.Add(1)

	lndb := &LevelNodeDB{
		current:          curNDB,
		prev:             prevNDB,
		PropagateDeletes: propagateDeletes,
		mutex:            &sync.RWMutex{},
		version:          v,
	}
	lndb.DeletedNodes = make(map[StrKey]bool)
	return lndb
}

// GetDBVersion get the current level node db version
func (lndb *LevelNodeDB) GetDBVersion() int64 {
	lndb.mutex.RLock()
	defer lndb.mutex.RUnlock()
	return lndb.version
}

// GetCurrent returns current node db
func (lndb *LevelNodeDB) GetCurrent() NodeDB {
	lndb.mutex.RLock()
	defer lndb.mutex.RUnlock()
	return lndb.current
}

// GetPrev returns previous node db
func (lndb *LevelNodeDB) GetPrev() NodeDB {
	lndb.mutex.RLock()
	defer lndb.mutex.RUnlock()
	return lndb.prev
}

// SetPrev sets  previous node db
func (lndb *LevelNodeDB) SetPrev(prevDB NodeDB) {
	lndb.mutex.Lock()
	defer lndb.mutex.Unlock()
	lndb.prev = prevDB
}

func (lndb *LevelNodeDB) isCurrentPersistent() (ok bool) {
	_, ok = lndb.current.(*PNodeDB)
	return
}

// unsafe
func (lndb *LevelNodeDB) getNode(key Key) (Node, error) {
	p, c := lndb.prev, lndb.current
	node, err := c.GetNode(key)
	if err != nil {
		if p != c {
			return p.GetNode(key)
		}
		return nil, err
	}
	return node, nil
}

// unsafe
func (lndb *LevelNodeDB) putNode(key Key, node Node) error {
	return lndb.current.PutNode(key, node)
}

// unsafe
func (lndb *LevelNodeDB) deleteNode(key Key) error {
	p, c := lndb.prev, lndb.current
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

/*GetNode - implement interface */
func (lndb *LevelNodeDB) GetNode(key Key) (Node, error) {
	lndb.mutex.RLock()
	defer lndb.mutex.RUnlock()
	return lndb.getNode(key)
}

/*PutNode - implement interface */
func (lndb *LevelNodeDB) PutNode(key Key, node Node) error {
	lndb.mutex.Lock()
	defer lndb.mutex.Unlock()
	return lndb.putNode(key, node)
}

/*DeleteNode - implement interface */
func (lndb *LevelNodeDB) DeleteNode(key Key) error {
	lndb.mutex.Lock()
	defer lndb.mutex.Unlock()
	return lndb.deleteNode(key)
}

/*MultiGetNode - get multiple nodes */
func (lndb *LevelNodeDB) MultiGetNode(keys []Key) (nodes []Node, err error) {
	lndb.mutex.RLock()
	defer lndb.mutex.RUnlock()
	for _, key := range keys {
		node, nerr := lndb.getNode(key)
		if nerr != nil {
			err = nerr
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes, err
}

/*MultiPutNode - implement interface */
func (lndb *LevelNodeDB) MultiPutNode(keys []Key, nodes []Node) error {
	lndb.mutex.Lock()
	defer lndb.mutex.Unlock()
	for idx, key := range keys {
		err := lndb.putNode(key, nodes[idx])
		if err != nil {
			return err
		}
	}
	return nil
}

/*MultiDeleteNode - implement interface */
func (lndb *LevelNodeDB) MultiDeleteNode(keys []Key) error {
	lndb.mutex.Lock()
	defer lndb.mutex.Unlock()
	for _, key := range keys {
		err := lndb.deleteNode(key)
		if err != nil {
			return err
		}
	}
	return nil
}

/*Iterate - implement interface */
func (lndb *LevelNodeDB) Iterate(ctx context.Context, handler NodeDBIteratorHandler) error {
	lndb.mutex.RLock()
	defer lndb.mutex.RUnlock()
	p, c := lndb.prev, lndb.current
	err := c.Iterate(ctx, handler)
	if err != nil {
		return err
	}
	if p != c && !lndb.isCurrentPersistent() { // Why is it skipped when current is PNodeDB?
		return p.Iterate(ctx, handler)
	}
	return nil
}

/*Size - implement interface */
func (lndb *LevelNodeDB) Size(ctx context.Context) int64 {
	lndb.mutex.RLock()
	defer lndb.mutex.RUnlock()
	p, c := lndb.prev, lndb.current
	size := c.Size(ctx)
	if p != c {
		size += p.Size(ctx)
	}
	return size
}

func (lndb *LevelNodeDB) RecordDeadNodes(nodes []Node) (int, error) {
	return 0, nil
}

// PruneBelowVersion - implement interface.
func (lndb *LevelNodeDB) PruneBelowVersion(ctx context.Context, version Sequence) error {
	return nil
}

// RebaseCurrentDB - set the current database.
func (lndb *LevelNodeDB) RebaseCurrentDB(ndb NodeDB) {
	lndb.mutex.Lock()
	defer lndb.mutex.Unlock()
	Logger.Debug("LevelNodeDB rebase db")
	lndb.prev, lndb.current = ndb, ndb
}

// MergeState - merge the state from another node db.
func MergeState(ctx context.Context, fndb NodeDB, tndb NodeDB) error {
	var keys []Key
	var nodes []Node
	err := fndb.Iterate(ctx, func(ctx context.Context, key Key, node Node) error {
		keys, nodes = append(keys, key), append(nodes, node)
		return nil
	})
	if err != nil {
		return err
	}
	err = tndb.MultiPutNode(keys, nodes)
	if err != nil {
		return err
	}
	if pndb, ok := tndb.(*PNodeDB); ok {
		pndb.Flush()
	}
	return nil
}
