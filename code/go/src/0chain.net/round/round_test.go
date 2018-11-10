package round

import (
	"fmt"
	"reflect"
	"testing"

	"0chain.net/node"
)

func TestRoundStableRandomization(t *testing.T) {
	r := Round{Number: 1234}
	r.SetRandomSeed(2009)
	pool := node.NewPool(node.NodeTypeMiner)
	nd := &node.Node{Type: node.NodeTypeMiner, SetIndex: 0}
	nd.SetID("0")
	pool.AddNode(nd)
	nd = &node.Node{Type: node.NodeTypeMiner, SetIndex: 1}
	nd.SetID("1")
	pool.AddNode(nd)
	nd = &node.Node{Type: node.NodeTypeMiner, SetIndex: 2}
	nd.SetID("2")
	pool.AddNode(nd)
	pool.ComputeProperties()
	numElements := pool.Size()
	fmt.Printf("pool size %v\n", numElements)
	r.ComputeMinerRanks(pool)
	p1 := make([]int, numElements)
	copy(p1, r.minerPerm)
	p2 := make([]int, numElements)
	r.ComputeMinerRanks(pool)
	copy(p2, r.minerPerm)
	if !reflect.DeepEqual(p1, p2) {
		t.Errorf("Permutations are not the same: %v %v\n", p1, p2)
	} else {
		t.Logf("Permutations are the same: %v\n", p1)
	}
}
