package util

import (
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"context"
	"go.uber.org/zap"
	"io"
	"reflect"
	"sync"
)

const (
	CacheSize   = 1000
	Concurrency = 4
)

//Caching proxy that wraps MerklePatriciaTrieI. Flush is never called automatically.
//User of this proxy should be aware, that GetRoot, MergeDB, Iterate and other methods should be called only after Flush, to get recent updates.
type MPTCachingProxy struct {
	mpt        MerklePatriciaTrieI
	cache      map[string]MPTSerializable
	flusher    *common.WithContextFunc
	flush      func(path Path, value MPTSerializable)
	cacheGuard sync.Mutex
}

func NewMPTCachingProxy(ctx context.Context, mpt MerklePatriciaTrieI) *MPTCachingProxy {
	p := &MPTCachingProxy{mpt: mpt}
	p.flusher = common.NewWithContextFunc(Concurrency)

	p.cache = make(map[string]MPTSerializable, CacheSize)
	p.flush = func(path Path, value MPTSerializable) {
		err := p.flusher.Run(ctx, func() error {
			_, err := p.mpt.Insert(path, value)
			return err
		})
		if err != nil {
			logging.Logger.Error("error while flushing", zap.Error(err))
		}
	}

	return p
}

func (p *MPTCachingProxy) Flush() {
	p.cacheGuard.Lock()
	defer p.cacheGuard.Unlock()
	for key, val := range p.cache {
		p.flush(Path(key), val)
	}
	p.cache = make(map[string]MPTSerializable, CacheSize)
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
	//TODO: think about force flush here, to refresh root
	return p.mpt.GetRoot()
}

func (p *MPTCachingProxy) GetNodeValue(path Path, v MPTSerializable) error {
	p.cacheGuard.Lock()
	defer p.cacheGuard.Unlock()

	get, ok := p.cache[string(path)]
	if !ok {
		err := p.mpt.GetNodeValue(path, v)
		if err != nil {
			return err
		}
		if len(p.cache) > CacheSize {
			logging.Logger.Warn("Cache is overflown, use direct write")
			return nil
		}
		p.cache[string(path)] = v
		return nil
	}

	value := reflect.ValueOf(v).Elem()
	value.Set(reflect.ValueOf(get).Elem())
	return nil
}

func (p *MPTCachingProxy) GetNodeValueRaw(path Path) ([]byte, error) {
	return p.mpt.GetNodeValueRaw(path)
}

//TODO remove key return here
func (p *MPTCachingProxy) Insert(path Path, value MPTSerializable) (Key, error) {
	p.cacheGuard.Lock()
	defer p.cacheGuard.Unlock()

	if len(p.cache) > CacheSize {
		logging.Logger.Warn("Cache is overflown, use direct write")
		return p.mpt.Insert(path, value)
	}
	p.cache[string(path)] = value
	//TODO problem, no key
	return nil, nil
}

//TODO remove key return here
func (p *MPTCachingProxy) Delete(path Path) (Key, error) {
	p.cacheGuard.Lock()
	defer p.cacheGuard.Unlock()

	_, ok := p.cache[string(path)]
	if ok {
		delete(p.cache, string(path))
		key, err := p.mpt.Delete(path)
		//this value could be added to cache and wasn't flushed yet
		if err == ErrValueNotPresent {
			return key, nil
		}
		return key, err
	}
	return p.mpt.Delete(path)
}

func (p *MPTCachingProxy) Iterate(ctx context.Context, handler MPTIteratorHandler, visitNodeTypes byte) error {
	//TODO: think about force flush here, to refresh iterated nodes
	return p.mpt.Iterate(ctx, handler, visitNodeTypes)
}

func (p *MPTCachingProxy) IterateFrom(ctx context.Context, node Key, handler MPTIteratorHandler, visitNodeTypes byte) error {
	//TODO: think about force flush here, to refresh root
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
