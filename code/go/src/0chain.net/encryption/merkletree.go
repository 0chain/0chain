/* Based on https://github.com/cbergoon/merkletree with MIT License */

package encryption

import (
	"errors"
	"fmt"
	"strings"
)

/*MerkleTree - used to represent a  merkle tree */
type MerkleTree struct {
	RootNode   *MerkleNode
	MerkleRoot string
	Leafs      []*MerkleNode
}

/*MerkleNode - used to represent a merkle tree node */
type MerkleNode struct {
	Parent *MerkleNode
	Left   *MerkleNode
	Right  *MerkleNode
	Data   string /*For transaction IDs*/
	leaf   bool
}

/*CreateMerkleTree - creating the merkle tree */
func CreateMerkleTree(transactionIDs []string) (*MerkleTree, error) {
	root, leafs, err := CreateMerkleNode(transactionIDs)
	if err != nil {
		return nil, err
	}
	t := &MerkleTree{
		RootNode:   root,
		MerkleRoot: root.Data,
		Leafs:      leafs,
	}
	return t, nil
}

/*CreateMerkleNode - creates a merkle tree with the transactionIDs and returns the root and the Leafs*/
func CreateMerkleNode(transactionIDs []string) (*MerkleNode, []*MerkleNode, error) {
	if len(transactionIDs) == 0 {
		return nil, nil, errors.New("Cannot create a merkle tree with no transactionIDs as content")
	}
	var leafs []*MerkleNode
	for i := 0; i < len(transactionIDs); i++ {
		leafs = append(leafs, &MerkleNode{
			Data: transactionIDs[i],
			leaf: true,
		})
	}

	root := BuildIntermediateNodes(leafs)
	fmt.Printf("The merkle root is %s\n", root.Data)
	return root, leafs, nil
}

/*BuildIntermediateNodes - is a function which takes the leafs as input which is the transaction ids and returns the resulting root node */
func BuildIntermediateNodes(mnode []*MerkleNode) *MerkleNode {
	var nodes []*MerkleNode
	for n := 0; n < len(mnode); n += 2 {
		var left, right int = n, n + 1
		if n+1 == len(mnode) {
			right = n
		}
		leftHash := Hash(mnode[left].Data)
		rightHash := Hash(mnode[right].Data)
		values := []string{leftHash, rightHash}
		prevHash := strings.Join(values, "")
		hash := Hash(prevHash)
		newNode := &MerkleNode{
			Left:  mnode[left],
			Right: mnode[right],
			Data:  hash,
		}
		nodes = append(nodes, newNode)
		mnode[left].Parent = newNode
		mnode[right].Parent = newNode
		if len(mnode) == 2 {
			return newNode
		}
	}
	return BuildIntermediateNodes(nodes)
}

//MerkleRoot returns the Merkle root which is the root node of the tree.
func (m *MerkleTree) GetMerkleRoot() string {
	return m.MerkleRoot
}
