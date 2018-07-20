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
	GetNodeDB() NodeDB

	GetRoot() Key
	SetRoot(root Key)

	GetNodeValue(path Path) (Serializable, error)
	Insert(path Path, value Serializable, cc ChangeCollectorI) (Key, error)
	Delete(path Path, cc ChangeCollectorI) (Key, error)

	Iterate(ctx context.Context, handler MPTIteratorHandler, visitNodeTypes byte) error

	// useful for pruning the state below a certain origin number
	UpdateOrigin(ctx context.Context, origin Origin) error     // mark
	PruneBelowOrigin(ctx context.Context, origin Origin) error // sweep

	// only for testing and debugging
	PrettyPrint(w io.Writer) error
}
