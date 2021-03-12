package util

import (
	"fmt"
	"math/rand"
	"reflect"
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

func TestMerkleTree_computeSize(t *testing.T) {
	type fields struct {
		tree        []string
		leavesCount int
		levels      int
	}
	type args struct {
		leaves int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
		want1  int
	}{
		{
			name:  "Test_MerkleTree_computeSize_OK",
			args:  args{leaves: 1},
			want:  2,
			want1: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &MerkleTree{
				tree:        tt.fields.tree,
				leavesCount: tt.fields.leavesCount,
				levels:      tt.fields.levels,
			}
			got, got1 := mt.computeSize(tt.args.leaves)
			if got != tt.want {
				t.Errorf("computeSize() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("computeSize() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestMerkleTree_ComputeTree(t *testing.T) {
	txn := &Txn{data: encryption.Hash("data")}

	type fields struct {
		tree        []string
		leavesCount int
		levels      int
	}
	type args struct {
		hashes []Hashable
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantTree []string
	}{
		{
			name: "Test_MerkleTree_ComputeTree_OK",
			args: args{
				[]Hashable{
					txn,
				},
			},
			wantTree: []string{
				txn.GetHash(),
				MHash(txn.GetHash(), txn.GetHash()),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &MerkleTree{
				tree:        tt.fields.tree,
				leavesCount: tt.fields.leavesCount,
				levels:      tt.fields.levels,
			}
			mt.ComputeTree(tt.args.hashes)

			if !reflect.DeepEqual(mt.tree, tt.wantTree) {
				t.Errorf("ComputeTree() got = %v, want = %v", mt.tree, tt.wantTree)
			}
		})
	}
}

func TestMerkleTree_SetTree(t *testing.T) {
	type fields struct {
		tree        []string
		leavesCount int
		levels      int
	}
	type args struct {
		leavesCount int
		tree        []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_MerkleTree_SetTree_ERR",
			args:    args{leavesCount: 1, tree: make([]string, 0)},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &MerkleTree{
				tree:        tt.fields.tree,
				leavesCount: tt.fields.leavesCount,
				levels:      tt.fields.levels,
			}
			if err := mt.SetTree(tt.args.leavesCount, tt.args.tree); (err != nil) != tt.wantErr {
				t.Errorf("SetTree() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMerkleTree_GetLeafIndex(t *testing.T) {
	type fields struct {
		tree        []string
		leavesCount int
		levels      int
	}
	type args struct {
		hash Hashable
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name: "Test_MerkleTree_GetLeafIndex_Not_Found_OK",
			args: args{&Txn{}},
			want: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &MerkleTree{
				tree:        tt.fields.tree,
				leavesCount: tt.fields.leavesCount,
				levels:      tt.fields.levels,
			}
			if got := mt.GetLeafIndex(tt.args.hash); got != tt.want {
				t.Errorf("GetLeafIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMerkleTree_GetPath(t *testing.T) {
	type fields struct {
		tree        []string
		leavesCount int
		levels      int
	}
	type args struct {
		hash Hashable
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *MTPath
	}{
		{
			name: "Test_MerkleTree_GetPath_OK",
			args: args{&Txn{}},
			want: &MTPath{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &MerkleTree{
				tree:        tt.fields.tree,
				leavesCount: tt.fields.leavesCount,
				levels:      tt.fields.levels,
			}
			if got := mt.GetPath(tt.args.hash); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
