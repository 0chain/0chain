package util

import (
	"fmt"
	"sort"
)

/*MerkleTree - A data structure that implements MerkleTreeI interface */
type MerkleTree struct {
	tree        []string
	leavesCount int
	levels      int
}

func (mt *MerkleTree) computeSize(leaves int) (int, int) {
	var tsize int
	var levels int
	for ll := leaves; ll > 1; ll = (ll + 1) / 2 {
		tsize += ll
		levels++
	}
	tsize++
	levels++
	return tsize, levels
}

/*ComputeTree - given the leaf nodes, compute the merkle tree */
func (mt *MerkleTree) ComputeTree(hashes []Hashable) {
	var tsize int
	tsize, mt.levels = mt.computeSize(len(hashes))
	mt.leavesCount = len(hashes)
	mt.tree = make([]string, tsize)
	for idx, hashable := range hashes {
		mt.tree[idx] = hashable.GetHash()
	}
	sort.Strings(mt.tree[:mt.leavesCount])
	for pl0, plsize := 0, mt.leavesCount; plsize > 1; pl0, plsize = pl0+plsize, (plsize+1)/2 {
		l0 := pl0 + plsize
		for i, j := 0, 0; i < plsize; i, j = i+2, j+1 {
			mt.tree[pl0+plsize+j] = MHash(mt.tree[pl0+i], mt.tree[pl0+i+1])
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
func (mt *MerkleTree) SetTree(leavesCount int, tree []string) error {
	size, levels := mt.computeSize(leavesCount)
	if size != len(tree) {
		return fmt.Errorf("Merkle tree with leaves %v should have size %v but only %v is given", leavesCount, size, len(tree))
	}
	mt.levels = levels
	mt.tree = tree
	mt.leavesCount = leavesCount
	return nil
}

/*GetLeafIndex - Get the index of the leaf node in the tree */
func (mt *MerkleTree) GetLeafIndex(hash Hashable) int {
	hs := hash.GetHash()
	return sort.SearchStrings(mt.tree[:mt.leavesCount], hs)
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
	path := make([]MTPathNode, 1, mt.levels-1)
	if idx&1 == 1 {
		path[0] = MTPathNode{Hash: mt.tree[idx-1], Side: Left}
	} else {
		if idx+1 < mt.leavesCount {
			path[0] = MTPathNode{Hash: mt.tree[idx+1], Side: Right}
		} else {
			path[0] = MTPathNode{Hash: mt.tree[idx], Side: Right}
		}
	}
	for pl0, plsize := 0, mt.leavesCount; plsize > 2; pl0, plsize = pl0+plsize, (plsize+1)/2 {
		l0 := pl0 + plsize
		idx = (idx - idx&1) / 2
		if idx&1 == 1 {
			path = append(path, MTPathNode{Hash: mt.tree[l0+idx-1], Side: Left})
		} else {
			if l0+idx+1 < l0+(plsize+1)/2 {
				path = append(path, MTPathNode{Hash: mt.tree[l0+idx+1], Side: Right})
			} else {
				path = append(path, MTPathNode{Hash: mt.tree[l0+idx], Side: Right})
			}
		}
	}
	return path
}
