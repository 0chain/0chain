package util

import (
	"sort"
)

/*MerkleTree - A data structure that implements MerkleTreeI interface */
type MerkleTree struct {
	tree    []string
	offsets []int
}

func (mt *MerkleTree) computeOffsets(leaves int) int {
	var tsize int
	mt.offsets = make([]int, 1, 10)
	for ll := leaves; ll > 1; ll = (ll + 1) / 2 {
		tsize += ll
		mt.offsets = append(mt.offsets, tsize)
	}
	tsize++
	return tsize
}

/*ComputeTree - given the leaf nodes, compute the merkle tree */
func (mt *MerkleTree) ComputeTree(hashes []Hashable) {
	var tsize int
	tsize = mt.computeOffsets(len(hashes))
	mt.tree = make([]string, tsize)
	for idx, hashable := range hashes {
		mt.tree[idx] = hashable.GetHash()
	}
	sort.SliceStable(mt.tree[0:len(hashes)], func(i int, j int) bool { return mt.tree[i] < mt.tree[j] })
	for l, pl0, l0 := 1, 0, 0; l < len(mt.offsets); l, pl0 = l+1, l0 {
		l0 = mt.offsets[l]
		plsize := l0 - pl0
		for i, j := 0, 0; i < plsize; i, j = i+2, j+1 {
			mt.tree[l0+j] = MHash(mt.tree[pl0+i], mt.tree[pl0+i+1])
		}
		if plsize&1 == 1 {
			mt.tree[l0+plsize/2] = MHash(mt.tree[pl0+plsize-1], mt.tree[pl0+plsize-1])
		}
	}
}

/*GetRoot - get the root of the merkle tree */
func (mt *MerkleTree) GetRoot() string {
	return mt.tree[len(mt.tree)-1]
}

/*GetTree - get the entire merkle tree */
func (mt *MerkleTree) GetTree() []string {
	return mt.tree
}

/*SetTree - set the entire merkle tree */
func (mt *MerkleTree) SetTree(leaves int, tree []string) {
	mt.tree = tree
	mt.computeOffsets(leaves)
}

/*GetLeafIndex - Get the index of the leaf node in the tree */
func (mt *MerkleTree) GetLeafIndex(hash Hashable) int {
	hs := hash.GetHash()
	return sort.SearchStrings(mt.tree[:mt.offsets[1]], hs)
}

/*GetPath - get the path that can be used to verify the merkle tree */
func (mt *MerkleTree) GetPath(hash Hashable) []MTPathNode {
	hidx := mt.GetLeafIndex(hash)
	if hidx < 0 {
		return nil
	}
	return mt.GetPathByIndex(hidx)
}

/*VerifyPath - given a leaf node and the path, verify that the node is part of the tree */
func (mt *MerkleTree) VerifyPath(hash Hashable, path []MTPathNode) bool {
	hs := hash.GetHash()
	mthash := hs
	pl := len(path)
	for i := 0; i < pl; i++ {
		if path[i].Side == Left {
			mthash = MHash(path[i].Hash, mthash)
		} else {
			mthash = MHash(mthash, path[i].Hash)
		}
	}
	return mthash == mt.GetRoot()
}

/*GetPathByIndex - get the path of a leaf node at index i */
func (mt *MerkleTree) GetPathByIndex(idx int) []MTPathNode {
	path := make([]MTPathNode, 1, len(mt.offsets)-1)
	if idx&1 == 1 {
		path[0] = MTPathNode{Hash: mt.tree[idx-1], Side: Left}
	} else {
		if idx+1 < mt.offsets[1] {
			path[0] = MTPathNode{Hash: mt.tree[idx+1], Side: Right}
		} else {
			path[0] = MTPathNode{Hash: mt.tree[idx], Side: Right}
		}
	}
	for l := 1; l < len(mt.offsets)-1; l = l + 1 {
		l0 := mt.offsets[l]
		idx = (idx - idx&1) / 2
		if idx&1 == 1 {
			path = append(path, MTPathNode{Hash: mt.tree[l0+idx-1], Side: Left})
		} else {
			if l0+idx+1 < mt.offsets[l+1] {
				path = append(path, MTPathNode{Hash: mt.tree[l0+idx+1], Side: Right})
			} else {
				path = append(path, MTPathNode{Hash: mt.tree[l0+idx], Side: Right})
			}
		}
	}
	return path
}
