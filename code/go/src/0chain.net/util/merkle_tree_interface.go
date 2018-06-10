package util

import "0chain.net/encryption"

/*Hashable - any thing that can provide it's hash */
type Hashable interface {
	GetHash() string
}

/*MerkleTreeI - a merkle tree interface required for constructing and providing verification */
type MerkleTreeI interface {
	//API to create a tree from leaf nodes
	ComputeTree(hashes []Hashable)
	GetRoot() string
	GetTree() []string

	//API to load an existing tree
	SetTree(leaves int, tree []string)

	// API for verification when the leaf node is known
	GetPath(hash Hashable) []MTPathNode               // Server needs to provide this
	VerifyPath(hash Hashable, path []MTPathNode) bool //This is only required by a client but useful for testing

	/* API for random verification when the leaf node is uknown
	(verification of the data to hash used as leaf node is outside this API) */
	GetPathByIndex(idx int) []MTPathNode
}

const (
	//Left - The node is to the left of the previous node in the path
	Left = 0

	//Right - The node is to the right of the previous node in the path
	Right = 1
)

/*MTPathNode - The merkle tree path node that provides left/right direction */
type MTPathNode struct {
	Hash string
	Side byte
}

/*Hash - the hashing used for the merkle tree construction */
func Hash(text string) string {
	return encryption.Hash(text)
}

/*MHash - merkle hashing of a pair of child hashes */
func MHash(h1 string, h2 string) string {
	return Hash(h1 + h2)
}
