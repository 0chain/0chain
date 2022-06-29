package util

import (
	"context"
	"errors"
	"io"
	"time"
)

//ErrIteratingChildNodes - indicates an error iterting the child nodes
var (
	ErrIteratingChildNodes = errors.New("Error iterating child nodes")
	ErrMissingNodes        = errors.New("missing nodes")
)

//Path - a type for the path of the merkle patricia trie
type Path []byte

//Key - a type for the merkle patricia trie node key
type Key []byte

/*MPTIteratorHandler is a collection iteration handler function type */
type MPTIteratorHandler func(ctx context.Context, path Path, key Key, node Node) error

//MPTMissingNodeHandler - a handler for missing keys during iteration
type MPTMissingNodeHandler func(ctx context.Context, path Path, key Key) error

//MerklePatriciaTrieI - interface of the merkle patricia trie
type MerklePatriciaTrieI interface {
	SetNodeDB(ndb NodeDB)
	GetNodeDB() NodeDB
	SetVersion(version Sequence)
	GetVersion() Sequence

	GetRoot() Key

	GetNodeValue(path Path, v MPTSerializable) error
	// GetNodeValueRaw returns the raw data slice on the given path
	GetNodeValueRaw(path Path) ([]byte, error)
	Insert(path Path, value MPTSerializable) (Key, error)
	Delete(path Path) (Key, error)

	Iterate(ctx context.Context, handler MPTIteratorHandler, visitNodeTypes byte) error

	IterateFrom(ctx context.Context, node Key, handler MPTIteratorHandler, visitNodeTypes byte) error

	// get root, changes and deletes
	GetChanges() (Key, []*NodeChange, []Node, Key)
	GetChangeCount() int
	SaveChanges(ctx context.Context, ndb NodeDB, includeDeletes bool) error

	// useful for syncing up
	GetPathNodes(path Path) ([]Node, error)

	// useful for pruning the state below a certain origin number
	UpdateVersion(ctx context.Context, version Sequence, missingNodeHander MPTMissingNodeHandler) error // mark

	// FindMissingNodes find all missing nodes in a MPT tree
	FindMissingNodes(ctx context.Context) ([]Path, []Key, error)
	HasMissingNodes(ctx context.Context) (bool, error)
	// only for testing and debugging
	PrettyPrint(w io.Writer) error

	Validate() error

	MergeMPTChanges(mpt2 MerklePatriciaTrieI) error
	MergeChanges(newRoot Key, changes []*NodeChange, deletes []Node, startRoot Key) error
	MergeDB(ndb NodeDB, root Key) error
}

//ContextKey - a type for context key
type ContextKey string

/*PruneStatsKey - key used to get the prune stats object from the context */
const PruneStatsKey ContextKey = "prunestatskey"

/*WithPruneStats - return a context with a prune stats object */
func WithPruneStats(ctx context.Context) context.Context {
	ps := &PruneStats{Stage: PruneStateStart}
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

const (
	PruneStateStart     = "started"
	PruneStateUpdate    = "updating"
	PruneStateSynch     = "synching"
	PruneStateDelete    = "deleting"
	PruneStateCommplete = "completed"
	PruneStateAbandoned = "abandoned"
)

/*PruneStats - gathers statistics while pruning */
type PruneStats struct {
	Stage        string        `json:"stg"`
	Version      Sequence      `json:"v"`
	Total        int64         `json:"t"`
	Leaves       int64         `json:"l"`
	BelowVersion int64         `json:"bv"`
	Deleted      int64         `json:"d"`
	MissingNodes int64         `json:"mn"`
	UpdateTime   time.Duration `json:"ut"`
	DeleteTime   time.Duration `json:"dt"`
}
