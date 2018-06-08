package util

/*Hashable - any thing that can provide it's hash */
type Hashable interface {
	GetHash() string
}

const (
	// The node is to the left of the previous node in the path
	SideLeft = 0

	// The node is to the right of the previous node in the path
	SideRight = 1
)

/*MTPathNode - The merkle tree path node that provides left/right direction */
type MTPathNode struct {
	Hash       string
	SideInPath byte
}

/*MerkleTree - a merkle tree interface required for constructing and providing verification */
type MerkleTree interface {
	//API to create a tree from leaf nodes
	ComputeTree(hashes []Hashable)
	GetRoot() string
	GetTree() []string

	//API to load an existing tree
	SetTree([]string)

	// API for verification when the leaf node is known
	GetPath(hash Hashable) []string          // Server needs to provide this
	VerifyPath(hash Hashable, path []string) //This is only required by a client but useful for testing
	/*GetLeafIndex is not really required but the GetPath implementation would
	need a way to quickly identify the index */
	GetLeafIndex(hash Hashable) int

	/* API for random verification when the leaf node is uknown
	(verification of the data to hash used as leaf node is outside this API) */
	GetPathByIndex(idx int) []MTPathNode
	VerifyPathByIndex(path []MTPathNode) bool
}
