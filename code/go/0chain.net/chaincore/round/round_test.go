package round

import (
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/0chain/0chain/code/go/0chain.net/chaincore/node"
	"github.com/0chain/0chain/code/go/0chain.net/core/logging"
)

func init() {
	logging.InitLogging("testing")
}

func TestRoundStableRandomization(t *testing.T) {
	r := Round{Number: 1234}
	pool := node.NewPool(node.NodeTypeMiner)
	nd := &node.Node{Type: node.NodeTypeMiner, SetIndex: 0}
	if err := nd.SetID(hex.EncodeToString([]byte("0"))); err != nil {
		t.Fatal(err)
	}
	pool.AddNode(nd)
	nd = &node.Node{Type: node.NodeTypeMiner, SetIndex: 1}
	if err := nd.SetID(hex.EncodeToString([]byte("1"))); err != nil {
		t.Fatal(err)
	}
	pool.AddNode(nd)
	nd = &node.Node{Type: node.NodeTypeMiner, SetIndex: 2}
	if err := nd.SetID(hex.EncodeToString([]byte("2"))); err != nil {
		t.Fatal(err)
	}
	pool.AddNode(nd)
	pool.ComputeProperties()
	numElements := pool.Size()
	r.SetRandomSeed(2009, numElements)

	p1 := make([]int, numElements)
	copy(p1, r.minerPerm)
	p2 := make([]int, numElements)
	r.computeMinerRanks(pool.Size())
	copy(p2, r.minerPerm)
	if !reflect.DeepEqual(p1, p2) {
		t.Errorf("Permutations are not the same: %v %v\n", p1, p2)
	}
}
