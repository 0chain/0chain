package encryption

import (
	"sort"

	"0chain.net/util"
)

type MerkleTree struct {
	MerkleRoot string
	arr        []string
}

/*CreateTree - creating the merkle tree which is implemented as an array*/
func ComputeTree(hashes []util.Hashable) (*MerkleTree, error) {
	allocateSize := 2*len(hashes) - 1
	var merkleArray = make([]string, allocateSize)
	j := len(merkleArray) - 1
	for _, h := range hashes {
		merkleArray[j] = h.GetHash()
		j = j - 1
	}
	sort.Strings(merkleArray)
	//fmt.Printf("Merkle array after sorting : %s\n", merkleArray)
	i := len(merkleArray) - 1
	for i > 0 {
		parentIndex := GetParentIndex(i)
		merkleArray[parentIndex] = Hash(merkleArray[GetSiblingIndex(i)] + merkleArray[i])
		i = i - 2
	}
	t := &MerkleTree{
		MerkleRoot: merkleArray[0],
		arr:        merkleArray,
	}
	return t, nil
}

func (m *MerkleTree) GetRoot() string {
	return m.MerkleRoot
}

func GetSiblingIndex(i int) int {
	if i%2 == 0 {
		return i - 1
	} else {
		return i + 1
	}
}

func GetParentIndex(i int) int {
	return (i - 1) / 2
}
