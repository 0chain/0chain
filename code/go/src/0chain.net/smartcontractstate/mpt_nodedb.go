package smartcontractstate

import (
	"context"
	"errors"

	"0chain.net/config"
	. "0chain.net/logging"
	"0chain.net/util"
	"go.uber.org/zap"
)

/*ErrNodeNotFound - error indicating that the node is not found */
var ErrNodeNotFound = errors.New("node not found")

/*ErrValueNotPresent - error indicating given path is not present in the db */
var ErrValueNotPresent = errors.New("value not present")

/*NodeDBIteratorHandler is a nodedb iteration handler function type */
type NodeDBIteratorHandler func(ctx context.Context, key util.Key, node Node) error

/*NodeDB - an interface that gets, puts and deletes nodes by their key */
type NodeDB interface {
	GetNode(key util.Key) (Node, error)
	PutNode(key util.Key, node Node) error
	DeleteNode(key util.Key) error
	Iterate(ctx context.Context, handler NodeDBIteratorHandler) error
	Size(ctx context.Context) int64

	MultiPutNode(keys []util.Key, nodes []Node) error
	MultiDeleteNode(keys []util.Key) error
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
func (mndb *MemoryNodeDB) GetNode(key util.Key) (Node, error) {
	skey := StrKey(key)
	node, ok := mndb.Nodes[skey]
	if !ok {
		return nil, ErrNodeNotFound
	}
	return node, nil
}

/*PutNode - implement interface */
func (mndb *MemoryNodeDB) PutNode(key util.Key, node Node) error {
	skey := StrKey(key)
	mndb.Nodes[skey] = node
	return nil
}

/*DeleteNode - implement interface */
func (mndb *MemoryNodeDB) DeleteNode(key util.Key) error {
	skey := StrKey(key)
	delete(mndb.Nodes, skey)
	return nil
}

/*MultiPutNode - implement interface */
func (mndb *MemoryNodeDB) MultiPutNode(keys []util.Key, nodes []Node) error {
	for idx, key := range keys {
		err := mndb.PutNode(key, nodes[idx])
		if err != nil {
			return err
		}
	}
	return nil
}

/*MultiDeleteNode - implement interface */
func (mndb *MemoryNodeDB) MultiDeleteNode(keys []util.Key) error {
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
		err := handler(ctx, util.Key(key), node)
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
func (lndb *LevelNodeDB) GetNode(key util.Key) (Node, error) {
	c := lndb.C
	p := lndb.P
	node, err := c.GetNode(key)
	if err != nil && p != c {
		node, err = p.GetNode(key)
		if err != nil {
			if config.DevConfiguration.State {
				Logger.Error("get node", zap.String("key", util.ToHex(key)), zap.Error(err))
			}
		}
		return node, err
	}
	return node, nil
}

/*PutNode - implement interface */
func (lndb *LevelNodeDB) PutNode(key util.Key, node Node) error {
	return lndb.C.PutNode(key, node)
}

/*DeleteNode - implement interface */
func (lndb *LevelNodeDB) DeleteNode(key util.Key) error {
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
func (lndb *LevelNodeDB) MultiPutNode(keys []util.Key, nodes []Node) error {
	for idx, key := range keys {
		err := lndb.PutNode(key, nodes[idx])
		if err != nil {
			return err
		}
	}
	return nil
}

/*MultiDeleteNode - implement interface */
func (lndb *LevelNodeDB) MultiDeleteNode(keys []util.Key) error {
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

/*RebaseCurrentDB - set the current database */
func (lndb *LevelNodeDB) RebaseCurrentDB(ndb NodeDB) {
	lndb.C = ndb
	lndb.P = lndb.C
}
