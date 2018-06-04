/* Based on https://github.com/cbergoon/merkletree with MIT License */

package encryption

import (
	"errors"
	"fmt"
	"strings"
)

//MerkleTreeInterface is used to create the merkle root for a generic type */
type MerkleTreeInterface interface {
	GetHashID() string
}

/*MerkleTree - used to represent a  merkle tree */
type MerkleTree struct {
	RootNode   *MerkleNode
	MerkleRoot string
	Leafs      []*MerkleNode
}

/*MerkleNode - used to represent a merkle tree node */
type MerkleNode struct {
	Parent     *MerkleNode
	Left       *MerkleNode
	Right      *MerkleNode
	Data       string /*For transaction IDs*/
	leaf       bool
	MInterface MerkleTreeInterface
}

/*CreateMerkleTree - creating the merkle tree */
func CreateMerkleTree(mInterface []MerkleTreeInterface) (*MerkleTree, error) {
	root, leafs, err := CreateMerkleNode(mInterface)
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
func CreateMerkleNode(mInterface []MerkleTreeInterface) (*MerkleNode, []*MerkleNode, error) {
	if len(mInterface) == 0 {
		return nil, nil, errors.New("Cannot create a merkle tree with no transactionIDs as content")
	}
	var leafs []*MerkleNode
	for _, c := range mInterface {
		leafs = append(leafs, &MerkleNode{
			Data:       c.GetHashID(),
			MInterface: c,
			leaf:       true,
		})
	}

	if len(leafs)%2 == 1 {
		duplicateMerkleNode := &MerkleNode{
			Data:       leafs[len(leafs)-1].Data,
			MInterface: leafs[len(leafs)-1].MInterface,
			leaf:       true,
		}
		leafs = append(leafs, duplicateMerkleNode)
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
