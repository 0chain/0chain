package util

import (
	"fmt"
	"math/rand"
	"testing"

	"0chain.net/core/encryption"
)

type Txn struct {
	data string
}

func (t *Txn) GetHash() string {
	return t.data
}

func (t *Txn) GetHashBytes() []byte {
	return encryption.RawHash(t.data)
}

func (t *Txn) Encode() []byte {
	return []byte(t.data)
}

func (t *Txn) Decode(data []byte) error {
	t.data = string(data)
	return nil
}

func TestMerkleTreeComputeTree(t *testing.T) {
	txns := make([]Hashable, 100)
	for i := 0; i < len(txns); i++ {
		txns[i] = &Txn{data: fmt.Sprintf("%v", 1001-i)}
	}
	var mt MerkleTreeI = &MerkleTree{}
	mt.ComputeTree(txns)
	tree := mt.GetTree()
	if len(tree) != 202 {
		fmt.Printf("%v: %v\n", len(tree), tree)
	}
}

func TestMerkleTreeGetNVerifyPath(t *testing.T) {
	txns := make([]Hashable, 101)
	for i := 0; i < len(txns); i++ {
		txns[i] = &Txn{data: fmt.Sprintf("1000%v", i)}
	}
	var mt MerkleTreeI = &MerkleTree{}
	mt.ComputeTree(txns)
	for i := 0; i < len(txns); i++ {
		path := mt.GetPath(txns[i])
		if !mt.VerifyPath(txns[i], path) {
			fmt.Printf("path: %v %v\n", txns[i], path)
		}
	}
}

func TestMerkleTreeSetTree(t *testing.T) {
	txns := make([]Hashable, 100)
	for i := 0; i < len(txns); i++ {
		txns[i] = &Txn{data: fmt.Sprintf("%v", 1001-i)}
	}
	var mt MerkleTreeI = &MerkleTree{}
	mt.ComputeTree(txns)
	var mt2 MerkleTreeI = &MerkleTree{}
	mt2.SetTree(len(txns), mt.GetTree())
	if mt.GetRoot() != mt2.GetRoot() {
		t.Errorf("Merkle roots didn't match")
	}
}

func BenchmarkMerkleTreeComputeTree(b *testing.B) {
	txns := make([]Hashable, 10000)
	for i := 0; i < len(txns); i++ {
		txns[i] = &Txn{data: fmt.Sprintf("%v", len(txns)-i)}
	}
	for i := 0; i < b.N; i++ {
		var mt MerkleTreeI = &MerkleTree{}
		mt.ComputeTree(txns)
	}
}

func BenchmarkMerkleTreeGetPath(b *testing.B) {
	txns := make([]Hashable, 10000)
	for i := 0; i < len(txns); i++ {
		txns[i] = &Txn{data: fmt.Sprintf("%v", len(txns)-i)}
	}
	var mt MerkleTreeI = &MerkleTree{}
	mt.ComputeTree(txns)
	for i := 0; i < b.N; i++ {
		j := rand.Intn(len(txns))
		mt.GetPath(txns[j])
	}
}

func BenchmarkMerkleTreeVerifyPath(b *testing.B) {
	txns := make([]Hashable, 10000)
	for i := 0; i < len(txns); i++ {
		txns[i] = &Txn{data: fmt.Sprintf("%v", len(txns)-i)}
	}
	var mt MerkleTreeI = &MerkleTree{}
	mt.ComputeTree(txns)
	paths := make([]*MTPath, len(txns))
	for j := 0; j < len(txns); j++ {
		paths[j] = mt.GetPath(txns[j])
	}
	for i := 0; i < b.N; i++ {
		j := rand.Intn(len(txns))

		if !mt.VerifyPath(txns[j], paths[j]) {
			fmt.Printf("path verification failed")
			return
		}
	}
}
