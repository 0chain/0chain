package util

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"0chain.net/core/common"
	"go.uber.org/atomic"

	"reflect"

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

	PruneBelowVersion(ctx context.Context, version Sequence) error

	GetDBVersions() []int64
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

/*GetNode - implement interface */
func (mndb *MemoryNodeDB) GetNode(key Key) (Node, error) {
	skey := StrKey(key)
	mndb.mutex.RLock()
	defer mndb.mutex.RUnlock()
	node, ok := mndb.Nodes[skey]
	if !ok {
		return nil, ErrNodeNotFound
	}
	return node, nil
}

/*PutNode - implement interface */
func (mndb *MemoryNodeDB) PutNode(key Key, node Node) error {
	skey := StrKey(key)
	mndb.mutex.Lock()
	defer mndb.mutex.Unlock()
	mndb.Nodes[skey] = node
	return nil
}

/*DeleteNode - implement interface */
func (mndb *MemoryNodeDB) DeleteNode(key Key) error {
	skey := StrKey(key)
	mndb.mutex.Lock()
	defer mndb.mutex.Unlock()
	delete(mndb.Nodes, skey)
	return nil
}

/*MultiGetNode - get multiple nodes */
func (mndb *MemoryNodeDB) MultiGetNode(keys []Key) ([]Node, error) {
	var nodes []Node
	var err error
	for _, key := range keys {
		node, nerr := mndb.GetNode(key)
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
	for idx, key := range keys {
		mndb.PutNode(key, nodes[idx])
	}
	return nil
}

/*MultiDeleteNode - implement interface */
func (mndb *MemoryNodeDB) MultiDeleteNode(keys []Key) error {
	for _, key := range keys {
		mndb.DeleteNode(key)
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

// is node2 reachable from node using only nodes stored on this db
func (mndb *MemoryNodeDB) reachable(node Node, node2 Node) bool {
	switch nodeImpl := node.(type) {
	case *ExtensionNode:
		fn, err := mndb.GetNode(nodeImpl.NodeKey)
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
			childNode, err := mndb.GetNode(child)
			if err != nil {
				continue
			}
			if node2 == childNode {
				return true
			}
			ok := mndb.reachable(childNode, node2)
			if ok {
				return true
			}
		}
	}
	return false
}

/*ComputeRoot - compute root from partial set of nodes in this db */
func (mndb *MemoryNodeDB) ComputeRoot() Node {
	var root Node
	handler := func(ctx context.Context, key Key, node Node) error {
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
	}
	mndb.Iterate(nil, handler)
	return root
}

/*Validate - validate this MemoryNodeDB w.r.t the given root
  It should not contain any node that can't be reachable from the root.
  Note: The root itself can reach nodes not present in this db
*/
func (mndb *MemoryNodeDB) Validate(root Node) error {
	var (
		nodes   = make(map[StrKey]Node)
		iterate func(node Node)
	)
	iterate = func(node Node) {
		switch nodeImpl := node.(type) {
		case *FullNode:
			for _, ckey := range nodeImpl.Children {
				if ckey != nil {
					cnode, err := mndb.GetNode(ckey)
					if err == nil {
						nodes[StrKey(ckey)] = cnode
						iterate(cnode)
					}
				}
			}
		case *ExtensionNode:
			ckey := nodeImpl.NodeKey
			cnode, err := mndb.GetNode(ckey)
			if err == nil {
				nodes[StrKey(ckey)] = cnode
				iterate(cnode)
			}
		}
	}
	nodes[StrKey(root.GetHashBytes())] = root
	iterate(root)
	for _, nd := range mndb.Nodes {
		if _, ok := nodes[StrKey(nd.GetHashBytes())]; !ok {
			Logger.Error("mndb validate",
				zap.String("node_type", fmt.Sprintf("%T", nd)),
				zap.String("node_key", nd.GetHash()))
			return common.NewError("nodes_outside_tree", "not all nodes are from the root")
		}
	}
	return nil
}

// LevelNodeDB - a multi-level node db. It has a current node db and a previous
// node db. This is useful to update without changing the previous db.
type LevelNodeDB struct {
	mu               *sync.RWMutex
	current          NodeDB
	prev             NodeDB
	PropagateDeletes bool // Setting this to false (default) will not propagate delete to lower level db
	DeletedNodes     map[StrKey]bool
	version          int64
	versions         []int64
}

// NewLevelNodeDB - create a level node db
func NewLevelNodeDB(curNDB NodeDB, prevNDB NodeDB, propagateDeletes bool) *LevelNodeDB {
	vs := prevNDB.GetDBVersions()
	v := levelNodeVersion.Add(1)

	if len(vs) == 0 {
		Logger.Error("NewLevelNodeDB new thread",
			zap.Any("predb type", reflect.TypeOf(prevNDB)),
			zap.Any("new start db version", v),
		)
	}

	vs = append(vs, v)
	if len(vs) > 40 {
		vs = vs[len(vs)-40:]
	}

	lndb := &LevelNodeDB{
		current:          curNDB,
		prev:             prevNDB,
		PropagateDeletes: propagateDeletes,
		mu:               &sync.RWMutex{},
		version:          v,
		versions:         vs,
	}
	lndb.DeletedNodes = make(map[StrKey]bool)
	return lndb
}

// GetDBVersion get the current level node db version
func (lndb *LevelNodeDB) GetDBVersion() int64 {
	lndb.mu.Lock()
	defer lndb.mu.Unlock()
	return lndb.version
}

// GetDBVersions returns all tracked db versions
func (lndb *LevelNodeDB) GetDBVersions() []int64 {
	lndb.mu.Lock()
	defer lndb.mu.Unlock()
	vs := make([]int64, len(lndb.versions))
	for i, v := range lndb.versions {
		vs[i] = v
	}
	return vs
}

// GetCurrent returns current node db
func (lndb *LevelNodeDB) GetCurrent() NodeDB {
	lndb.mu.RLock()
	defer lndb.mu.RUnlock()
	return lndb.current
}

// GetPrev returns previous node db
func (lndb *LevelNodeDB) GetPrev() NodeDB {
	lndb.mu.RLock()
	defer lndb.mu.RUnlock()
	return lndb.prev
}

// SetPrev sets  previous node db
func (lndb *LevelNodeDB) SetPrev(prevDB NodeDB) {
	lndb.mu.Lock()
	defer lndb.mu.Unlock()
	lndb.prev = prevDB
}

func (lndb *LevelNodeDB) isCurrentPersistent() bool {
	_, ok := lndb.current.(*PNodeDB)
	return ok
}

/*GetNode - implement interface */
func (lndb *LevelNodeDB) GetNode(key Key) (Node, error) {
	lndb.mu.RLock()
	defer lndb.mu.RUnlock()
	c := lndb.current
	p := lndb.prev
	node, err := c.GetNode(key)
	if err != nil {
		if p != c {
			return p.GetNode(key)
		}
		return nil, err
	}
	return node, nil
}

/*PutNode - implement interface */
func (lndb *LevelNodeDB) PutNode(key Key, node Node) error {
	lndb.mu.RLock()
	defer lndb.mu.RUnlock()
	return lndb.current.PutNode(key, node)
}

/*DeleteNode - implement interface */
func (lndb *LevelNodeDB) DeleteNode(key Key) error {
	lndb.mu.Lock()
	defer lndb.mu.Unlock()

	//Logger.Debug("LevelNodeDB delete node",
	//	zap.String("key", ToHex(key)),
	//	zap.Int64("version", lndb.version),
	//	zap.Int64s("db versions", lndb.versions))

	c := lndb.current
	p := lndb.prev
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

/*MultiGetNode - get multiple nodes */
func (lndb *LevelNodeDB) MultiGetNode(keys []Key) ([]Node, error) {
	var nodes []Node
	var err error
	for _, key := range keys {
		node, nerr := lndb.GetNode(key)
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
	lndb.mu.RLock()
	defer lndb.mu.RUnlock()
	c := lndb.current
	p := lndb.prev
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
	lndb.mu.RLock()
	defer lndb.mu.RUnlock()
	c := lndb.current
	p := lndb.prev
	size := c.Size(ctx)
	if p != c {
		size += p.Size(ctx)
	}
	return size
}

// PruneBelowVersion - implement interface.
func (lndb *LevelNodeDB) PruneBelowVersion(ctx context.Context, version Sequence) error {
	// TODO
	return nil
}

// RebaseCurrentDB - set the current database.
func (lndb *LevelNodeDB) RebaseCurrentDB(ndb NodeDB) {
	lndb.mu.Lock()
	defer lndb.mu.Unlock()
	Logger.Debug("LevelNodeDB rebase db")
	lndb.current = ndb
	lndb.prev = ndb
}

// MergeState - merge the state from another node db.
func MergeState(ctx context.Context, fndb NodeDB, tndb NodeDB) error {
	var nodes []Node
	var keys []Key
	handler := func(ctx context.Context, key Key, node Node) error {
		keys = append(keys, key)
		nodes = append(nodes, node)
		return nil
	}
	err := fndb.Iterate(ctx, handler)
	if err != nil {
		return err
	}
	err = tndb.MultiPutNode(keys, nodes)
	if pndb, ok := tndb.(*PNodeDB); ok {
		pndb.Flush()
	}
	return nil
}
