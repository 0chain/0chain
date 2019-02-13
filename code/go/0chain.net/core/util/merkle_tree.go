package util

import (
	"fmt"
)

/*MerkleTree - A data structure that implements MerkleTreeI interface */
type MerkleTree struct {
	tree        []string
	leavesCount int
	levels      int
}

func VerifyMerklePath(hash string, path *MTPath, root string) bool {
	mthash := hash
	pathNodes := path.Nodes
	pl := len(pathNodes)
	idx := path.LeafIndex
	for i := 0; i < pl; i++ {
		if idx&1 == 1 {
			mthash = MHash(pathNodes[i], mthash)
		} else {
			mthash = MHash(mthash, pathNodes[i])
		}
		idx = (idx - idx&1) / 2
	}
	return mthash == root
}

func (mt *MerkleTree) computeSize(leaves int) (int, int) {
	if leaves == 1 {
		return 2, 2
	}
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
	if len(hashes) == 1 {
		mt.tree[1] = MHash(mt.tree[0], mt.tree[0])
		return
	}
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
	for i := 0; i < mt.leavesCount; i++ {
		if mt.tree[i] == hs {
			return i
		}
	}
	return -1
}

/*GetPath - get the path that can be used to verify the merkle tree */
func (mt *MerkleTree) GetPath(hash Hashable) *MTPath {
	hidx := mt.GetLeafIndex(hash)
	if hidx < 0 {
		return &MTPath{}
	}
	return mt.GetPathByIndex(hidx)
}

/*VerifyPath - given a leaf node and the path, verify that the node is part of the tree */
func (mt *MerkleTree) VerifyPath(hash Hashable, path *MTPath) bool {
	hs := hash.GetHash()
	return VerifyMerklePath(hs, path, mt.GetRoot())
}

/*GetPathByIndex - get the path of a leaf node at index i */
func (mt *MerkleTree) GetPathByIndex(idx int) *MTPath {
	path := make([]string, mt.levels-1, mt.levels-1)
	mpath := &MTPath{LeafIndex: idx}
	if idx&1 == 1 {
		path[0] = mt.tree[idx-1]
	} else {
		if idx+1 < mt.leavesCount {
			path[0] = mt.tree[idx+1]
		} else {
			path[0] = mt.tree[idx]
		}
	}
	for pl0, plsize, pi := 0, mt.leavesCount, 1; plsize > 2; pl0, plsize, pi = pl0+plsize, (plsize+1)/2, pi+1 {
		l0 := pl0 + plsize
		idx = (idx - idx&1) / 2
		if idx&1 == 1 {
			//path = append(path, mt.tree[l0+idx-1])
			path[pi] = mt.tree[l0+idx-1]
		} else {
			if l0+idx+1 < l0+(plsize+1)/2 {
				//path = append(path, mt.tree[l0+idx+1])
				path[pi] = mt.tree[l0+idx+1]
			} else {
				//path = append(path, mt.tree[l0+idx])
				path[pi] = mt.tree[l0+idx]
			}
		}
	}
	mpath.Nodes = path
	return mpath
}
