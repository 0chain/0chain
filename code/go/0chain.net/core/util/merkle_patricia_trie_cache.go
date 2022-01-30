package util

import (
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"context"
	"errors"
	"go.uber.org/zap"
	"io"
)

const (
	CacheSize   = 1000
	Concurrency = 4
)

var (
	PathBadType  = errors.New("key is not path type")
	ValueBadType = errors.New("value is not Serializable type")
)

type MPTCachingProxy struct {
	mpt     MerklePatriciaTrieI
	cache   map[string]Serializable
	flusher *common.WithContextFunc
	flush   func(key, value interface{})
}

func NewMPTCachingProxy(ctx context.Context, mpt MerklePatriciaTrieI) *MPTCachingProxy {
	p := &MPTCachingProxy{mpt: mpt}
	p.flusher = common.NewWithContextFunc(Concurrency)

	p.cache = make(map[string]Serializable, CacheSize)
	p.flush = func(key, value interface{}) {
		path, ok := key.(Path)
		if !ok {
			logging.Logger.Error("mpt_cache_flush", zap.Error(PathBadType))
			return
		}
		ser, ok := value.(Serializable)
		if !ok {
			logging.Logger.Error("mpt_cache_flush", zap.Error(ValueBadType))
			return
		}
		err := p.flusher.Run(ctx, func() error {
			_, err := p.mpt.Insert(path, ser)
			return err
		})
		if err != nil {
			logging.Logger.Error("error while flushing", zap.Error(err))
		}
	}

	return p
}

func (p *MPTCachingProxy) Flush() {
	for key, val := range p.cache {
		p.flush(key, val)
	}
	p.cache = make(map[string]Serializable, CacheSize)
}

func (p *MPTCachingProxy) SetNodeDB(ndb NodeDB) {
	p.mpt.SetNodeDB(ndb)
}

func (p *MPTCachingProxy) GetNodeDB() NodeDB {
	return p.mpt.GetNodeDB()
}

func (p *MPTCachingProxy) SetVersion(version Sequence) {
	p.mpt.SetVersion(version)
}

func (p *MPTCachingProxy) GetVersion() Sequence {
	return p.mpt.GetVersion()
}

func (p *MPTCachingProxy) GetRoot() Key {
	return p.mpt.GetRoot()
}

func (p *MPTCachingProxy) GetNodeValue(path Path) (Serializable, error) {
	get, ok := p.cache[string(path)]
	if !ok {
		value, err := p.mpt.GetNodeValue(path)
		if err != nil {
			return value, err
		}
		p.cache[string(path)] = value
		return value, nil
	}
	return get, nil
}

//TODO remove key return here
func (p *MPTCachingProxy) Insert(path Path, value Serializable) (Key, error) {
	key, err := p.mpt.Insert(path, value)
	if err != nil || len(p.cache) > CacheSize {
		return key, err
	}
	p.cache[string(path)] = value
	return key, err
}

//TODO remove key return here
func (p *MPTCachingProxy) Delete(path Path) (Key, error) {
	delete(p.cache, string(path))
	return p.mpt.Delete(path)
}

func (p *MPTCachingProxy) Iterate(ctx context.Context, handler MPTIteratorHandler, visitNodeTypes byte) error {
	return p.mpt.Iterate(ctx, handler, visitNodeTypes)
}

func (p *MPTCachingProxy) IterateFrom(ctx context.Context, node Key, handler MPTIteratorHandler, visitNodeTypes byte) error {
	return p.mpt.IterateFrom(ctx, node, handler, visitNodeTypes)
}

func (p *MPTCachingProxy) GetChanges() (Key, []*NodeChange, []Node, Key) {
	return p.mpt.GetChanges()
}

func (p *MPTCachingProxy) GetChangeCount() int {
	return p.mpt.GetChangeCount()
}

func (p *MPTCachingProxy) SaveChanges(ctx context.Context, ndb NodeDB, includeDeletes bool) error {
	return p.mpt.SaveChanges(ctx, ndb, includeDeletes)
}

func (p *MPTCachingProxy) GetPathNodes(path Path) ([]Node, error) {
	return p.mpt.GetPathNodes(path)
}

func (p *MPTCachingProxy) UpdateVersion(ctx context.Context, version Sequence, missingNodeHander MPTMissingNodeHandler) error {
	return p.mpt.UpdateVersion(ctx, version, missingNodeHander)
}

func (p *MPTCachingProxy) FindMissingNodes(ctx context.Context) ([]Path, []Key, error) {
	return p.mpt.FindMissingNodes(ctx)
}

func (p *MPTCachingProxy) PrettyPrint(w io.Writer) error {
	return p.mpt.PrettyPrint(w)
}

func (p *MPTCachingProxy) Validate() error {
	return p.mpt.Validate()
}

func (p *MPTCachingProxy) MergeMPTChanges(mpt2 MerklePatriciaTrieI) error {
	return p.mpt.MergeMPTChanges(mpt2)
}

func (p *MPTCachingProxy) MergeChanges(newRoot Key, changes []*NodeChange, deletes []Node, startRoot Key) error {
	return p.mpt.MergeChanges(newRoot, changes, deletes, startRoot)
}

func (p *MPTCachingProxy) MergeDB(ndb NodeDB, root Key) error {
	return p.mpt.MergeDB(ndb, root)
}
