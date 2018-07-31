package util

import "0chain.net/encryption"

/*MerkleTreeI - a merkle tree interface required for constructing and providing verification */
type MerkleTreeI interface {
	//API to create a tree from leaf nodes
	ComputeTree(hashes []Hashable)
	GetRoot() string
	GetTree() []string

	//API to load an existing tree
	SetTree(leavesCount int, tree []string) error

	// API for verification when the leaf node is known
	GetPath(hash Hashable) MTPath               // Server needs to provide this
	VerifyPath(hash Hashable, path MTPath) bool //This is only required by a client but useful for testing

	/* API for random verification when the leaf node is uknown
	(verification of the data to hash used as leaf node is outside this API) */
	GetPathByIndex(idx int) MTPath
}

const (
	//Left - The node is to the left of the previous node in the path
	Left = 0

	//Right - The node is to the right of the previous node in the path
	Right = 1
)

/*MTPath - The merkle tree path*/
type MTPath struct {
	Nodes     []string `json:"nodes"`
	LeafIndex int      `json:"leaf_index"`
}

/*Hash - the hashing used for the merkle tree construction */
func Hash(text string) string {
	return encryption.Hash(text)
}

/*MHash - merkle hashing of a pair of child hashes */
func MHash(h1 string, h2 string) string {
	return Hash(h1 + h2)
}
