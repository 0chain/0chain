package util

import (
	"context"
	"io"
)

//Path - a type for the path of the merkle patricia trie
type Path []byte

//Key - a type for the merkle patricia trie node key
type Key []byte

/*MPTIteratorHandler is a collection iteration handler function type */
type MPTIteratorHandler func(ctx context.Context, path Path, key Key, node Node) error

//MerklePatriciaTrieI - interface of the merkle patricia trie
type MerklePatriciaTrieI interface {
	SetNodeDB(ndb NodeDB)
	GetNodeDB() NodeDB

	GetRoot() Key
	SetRoot(root Key)

	GetNodeValue(path Path) (Serializable, error)
	Insert(path Path, value Serializable) (Key, error)
	Delete(path Path) (Key, error)

	Iterate(ctx context.Context, handler MPTIteratorHandler, visitNodeTypes byte) error

	GetChangeCollector() ChangeCollectorI
	ResetChangeCollector(root Key)
	SaveChanges(ndb NodeDB, origin Origin, includeDeletes bool) error

	// useful for syncing up
	GetPathNodes(path Path) ([]Node, error)

	// useful for pruning the state below a certain origin number
	UpdateOrigin(ctx context.Context, origin Origin) error // mark

	// only for testing and debugging
	PrettyPrint(w io.Writer) error
}

//ContextKey - a type for context key
type ContextKey string

/*PruneStatsKey - key used to get the prune stats object from the context */
const PruneStatsKey ContextKey = "prunestatskey"

/*WithPruneStats - return a context with a prune stats object */
func WithPruneStats(ctx context.Context) context.Context {
	ps := &PruneStats{}
	return context.WithValue(ctx, PruneStatsKey, ps)
}

/*GetPruneStats - returns a prune stats object from the context */
func GetPruneStats(ctx context.Context) *PruneStats {
	v := ctx.Value(PruneStatsKey)
	if v == nil {
		return nil
	}
	return v.(*PruneStats)
}

/*PruneStats - gathers statistics while pruning */
type PruneStats struct {
	Origin      Origin
	Total       int64
	Leaves      int64
	BelowOrigin int64
	Deleted     int64
}
