package smartcontractstate

import (
	"context"
	"errors"
	"io"
	"io/ioutil"

	"0chain.net/config"
	. "0chain.net/logging"
	"0chain.net/util"
	"go.uber.org/zap"
)

/*ErrNodeNotFound - error indicating that the node is not found */
var ErrNodeNotFound = errors.New("node not found")

/*ErrValueNotPresent - error indicating given path is not present in the db */
var ErrValueNotPresent = errors.New("value not present")

/*SCDBIteratorHandler is a iteration handler function type */
type SCDBIteratorHandler func(ctx context.Context, key Key, node Node) error

/*SCDB - an interface that gets, puts and deletes nodes by their key */
type SCDB interface {
	GetNode(key Key) (Node, error)
	PutNode(key Key, node Node) error
	DeleteNode(key Key) error
	Iterate(ctx context.Context, handler SCDBIteratorHandler) error
	Size(ctx context.Context) int64

	MultiPutNode(keys []Key, nodes []Node) error
	MultiDeleteNode(keys []Key) error
}

func PrettyPrint(ctx context.Context, ndb SCDB) {
	handler := func(ctx context.Context, key Key, node Node) error {
		Logger.Info("SmartContractState: ", zap.Any("key", string(key)), zap.Any("value", string(node)))
		return nil
	}
	ndb.Iterate(ctx, handler)
}

/*CreateNode - create a node based on the serialization prefix */
func CreateNode(r io.Reader) (Node, error) {
	buf := []byte{0}
	buf, err := ioutil.ReadAll(r)
	return buf, err
}

func SaveChanges(ctx context.Context, fromNodeDB SCDB, toNodeDB SCDB) error {
	var keys []Key
	var nodes []Node
	handler := func(ctx context.Context, key Key, node Node) error {
		Logger.Info("Putting the keys from the transaction to the block", zap.Any("key", util.ToHex(key)), zap.Any("value", node))
		keys = append(keys, key)
		nodes = append(nodes, node)
		return nil
	}
	fdb := fromNodeDB
	_, ok := fromNodeDB.(*PipedSCDB)
	if ok {
		fdb = fromNodeDB.(*PipedSCDB).C
	}
	if fdb != nil {
		err := fdb.Iterate(ctx, handler)
		if err != nil {
			return err
		}
	}

	return toNodeDB.MultiPutNode(keys, nodes)
}

/*StrKey - data type for the key used to store the node into some storage (this is needed as hashmap keys can't be []byte */
type StrKey string

/*MemorySCDB - an inmemory sc db */
type MemorySCDB struct {
	Nodes map[StrKey]Node
}

/*NewMemorySCDB - create a memory sc db */
func NewMemorySCDB() *MemorySCDB {
	mndb := &MemorySCDB{}
	mndb.Nodes = make(map[StrKey]Node)
	return mndb
}

/*GetNode - implement interface */
func (mndb *MemorySCDB) GetNode(key Key) (Node, error) {
	skey := StrKey(key)
	node, ok := mndb.Nodes[skey]
	if !ok {
		return nil, ErrNodeNotFound
	}
	return node, nil
}

/*PutNode - implement interface */
func (mndb *MemorySCDB) PutNode(key Key, node Node) error {
	skey := StrKey(key)
	mndb.Nodes[skey] = node
	return nil
}

/*DeleteNode - implement interface */
func (mndb *MemorySCDB) DeleteNode(key Key) error {
	skey := StrKey(key)
	delete(mndb.Nodes, skey)
	return nil
}

/*MultiPutNode - implement interface */
func (mndb *MemorySCDB) MultiPutNode(keys []Key, nodes []Node) error {
	for idx, key := range keys {
		err := mndb.PutNode(key, nodes[idx])
		if err != nil {
			return err
		}
	}
	return nil
}

/*MultiDeleteNode - implement interface */
func (mndb *MemorySCDB) MultiDeleteNode(keys []Key) error {
	for _, key := range keys {
		err := mndb.DeleteNode(key)
		if err != nil {
			return err
		}
	}
	return nil
}

/*Iterate - implement interface */
func (mndb *MemorySCDB) Iterate(ctx context.Context, handler SCDBIteratorHandler) error {
	for key, node := range mndb.Nodes {
		err := handler(ctx, Key(key), node)
		if err != nil {
			return err
		}
	}
	return nil
}

/*Size - implement interface */
func (mndb *MemorySCDB) Size(ctx context.Context) int64 {
	return int64(len(mndb.Nodes))
}

/*PipedSCDB - a multi-level node db. It has a current node db and a previous node db. This is useful to update without changing the previous db. */
type PipedSCDB struct {
	C                SCDB
	P                SCDB
	PropagateDeletes bool // Setting this to false (default) will not propagate delete to lower level db
	DeletedNodes     map[StrKey]bool
}

/*NewPipedSCDB - create a level node db */
func NewPipedSCDB(curNDB SCDB, prevNDB SCDB, propagateDeletes bool) *PipedSCDB {
	lndb := &PipedSCDB{C: curNDB, P: prevNDB, PropagateDeletes: propagateDeletes}
	lndb.DeletedNodes = make(map[StrKey]bool)
	return lndb
}

func (lndb *PipedSCDB) isCurrentPersistent() bool {
	_, ok := lndb.C.(*PSCDB)
	return ok
}

/*GetNode - implement interface */
func (lndb *PipedSCDB) GetNode(key Key) (Node, error) {
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
func (lndb *PipedSCDB) PutNode(key Key, node Node) error {
	return lndb.C.PutNode(key, node)
}

/*DeleteNode - implement interface */
func (lndb *PipedSCDB) DeleteNode(key Key) error {
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
func (lndb *PipedSCDB) MultiPutNode(keys []Key, nodes []Node) error {
	for idx, key := range keys {
		err := lndb.PutNode(key, nodes[idx])
		if err != nil {
			return err
		}
	}
	return nil
}

/*MultiDeleteNode - implement interface */
func (lndb *PipedSCDB) MultiDeleteNode(keys []Key) error {
	for _, key := range keys {
		err := lndb.DeleteNode(key)
		if err != nil {
			return err
		}
	}
	return nil
}

/*Iterate - implement interface */
func (lndb *PipedSCDB) Iterate(ctx context.Context, handler SCDBIteratorHandler) error {
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
func (lndb *PipedSCDB) Size(ctx context.Context) int64 {
	c := lndb.C
	p := lndb.P
	size := c.Size(ctx)
	if p != c {
		size += p.Size(ctx)
	}
	return size
}

/*RebaseCurrentDB - set the current database */
func (lndb *PipedSCDB) RebaseCurrentDB(ndb SCDB) {
	lndb.C = ndb
	lndb.P = lndb.C
}
