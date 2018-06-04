package util

/*Hashable - any thing that can provide it's hash */
type Hashable interface {
	GetHash() string
}

/*MerkleTree - a merkle tree interface required for constructing and providing verification */
type MerkleTree interface {
	//API to create a tree from leaf nodes
	ComputeTree(hashes []Hashable)
	GetRoot() string
	GetTree() []string

	//API to load an existing tree
	SetTree([]string)

	// API during the verification
	GetPath(hash Hashable) []string          // Server needs to provide this
	VerifyPath(hash Hashable, path []string) //This is only required by a client but useful for testing
	/*GetLeafIndex is not really required but the GetPath implementation would need a way to quickly identify the index */
	GetLeafIndex(hash Hashable) int
}
