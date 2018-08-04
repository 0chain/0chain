package round

import (
	"reflect"
	"testing"
)

func TestRoundStableRandomization(t *testing.T) {
	r := Round{Number: 1234, RandomSeed: 2009}
	numElements := 3
	p1 := make([]int, numElements)
	r.ComputeRanks(3, 3)
	copy(p1, r.minerPerm)
	p2 := make([]int, numElements)
	r.ComputeRanks(3, 3)
	copy(p2, r.sharderPerm)
	if !reflect.DeepEqual(p1, p2) {
		t.Errorf("Permutations are not the same: %v %v\n", p1, p2)
	} else {
		t.Logf("Permutations are the same: %v\n", p1)
	}
}
