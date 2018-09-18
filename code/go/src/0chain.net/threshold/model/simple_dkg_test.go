package model

import (
	"testing"
)

var invalidKeyShare = KeyShare {
	m: Key{1, 1, 1, 1},
	v: VerificationKey{2, 2, 2, 2},
}

// All unique choices of t-1 elements from [1..n-1].
func allReceiptCombosTOfN(t, n int) [][]int {
	sets := make([][]int, 0)
	set := make([]int, t - 1)

	var find func(idx, min int)

	find = func(idx, min int) {
		if idx == t - 1 {
			s := make([]int, t - 1)
			copy(s, set)
			sets = append(sets, s)
			return
		}

		for v := min; v < n; v++ {
			set[idx] = v
			find(idx + 1, v + 1)
		}
	}

	find(0, 1)

	return sets
}

func TestReceiptCombos(test *testing.T) {
	eq := func(a, b [][]int) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if len(a[i]) != len(b[i]) {
				return false
			}
			for j := range a[i] {
				if a[i][j] != b[i][j] {
					return false
				}
			}
		}
		return true
	}

	if !eq(allReceiptCombosTOfN(1, 2), [][]int{{}}) {
		test.Error(1, 2)
	}
	if !eq(allReceiptCombosTOfN(2, 2), [][]int{{1}}) {
		test.Error(2, 2)
	}
	if !eq(allReceiptCombosTOfN(2, 3), [][]int{{1}, {2}}) {
		test.Error(2, 3)
	}
	if !eq(allReceiptCombosTOfN(3, 3), [][]int{{1, 2}}) {
		test.Error(3, 3)
	}
	if !eq(allReceiptCombosTOfN(2, 4), [][]int{{1}, {2}, {3}}) {
		test.Error(2, 4)
	}
	if !eq(allReceiptCombosTOfN(3, 4), [][]int{{1, 2}, {1, 3}, {2, 3}}) {
		test.Error(3, 4)
	}
	if !eq(allReceiptCombosTOfN(4, 4), [][]int{{1, 2, 3}}) {
		test.Error(4, 4)
	}
}

// PartyIds that a DKG party will refer to another party by, such that each DKG
// thinks of itself as being PartyId 0, which is not necessary, but it's how
// academia sometimes models DKGs.
func remote(n, from, to int) PartyId {
	return PartyId((n + to - from) % n)
}

type Group struct {
	t    int
	n    int
	dkgs []SimpleDKG
}

func newGroup(t, n int) Group {
	dkgs := make([]SimpleDKG, n)
	for i := range dkgs {
		dkgs[i] = NewSimpleDKG(t, n)
	}
	return Group{
		t: t,
		n: n,
		dkgs: dkgs,
	}
}

func (g *Group) send(from, to int) error {
	share := g.dkgs[from].GetShareFor(g.remote(from, to))
	err := g.dkgs[to].ReceiveShare(g.remote(to, from), share)
	return err
}

func (g *Group) sendMany(count int) error {
	for i := 1; i <= count; i++ {
		err := g.send(i, 0)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Group) sendSubset(from []int) error {
	for _,i := range from {
		err := g.send(i, 0)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Group) done(i int) bool {
	return g.dkgs[i].IsDone()
}

func (g *Group) remote(from, to int) PartyId {
	return remote(g.n, from, to)
}

func TestDKGCanSelfShare(test *testing.T) {
	variation := func(t, n int) {
		dkg := NewSimpleDKG(t, n)
		share := dkg.GetShareFor(0)
		err := dkg.ReceiveShare(0, share)
		if err != nil {
			test.Errorf("DKG(t=%d,n=%d): Receive own share failed", t, n)
		}
	}

	variation(1, 1)
	variation(1, 2)
	variation(2, 2)
	variation(2, 3)
}

func TestDKGCanShareAndReceive(test *testing.T) {
	variation := func(t, n int) {
		group := newGroup(t, n)
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if i == j {
					continue
				}
				err := group.send(i, j)
				if err != nil {
					test.Errorf("DKG(t=%d,n=%d): Key share validation failed", t, n)
				}
			}
		}
	}

	variation(1, 2)
	variation(2, 2)
	variation(1, 3)
	variation(2, 3)
	variation(3, 3)
}

func TestDKGAlwaysFinishedIfTIs1(test *testing.T) {
	variation := func(n int) {
		dkg := NewSimpleDKG(1, n)
		done := dkg.IsDone()
		if !done {
			test.Errorf("DKG(t=1,n=%d): Should finish without needing to receive a single share", n)
		}
	}

	variation(1)
	variation(3)
}

func TestDKGCanFinish(test *testing.T) {
	variation := func(t, n int) {
		group := newGroup(t, n)
		err := group.sendMany(t - 1)
		if err != nil {
			test.Fatalf("DKG(t=%d,n=%d): Key share validation failed", t, n)
		}
		if !group.done(0) {
			test.Errorf("DKG(t=%d,n=%d): Not done after receiving %d remote shares", t, n, t - 1)
		}
	}

	variation(2, 5)
	variation(3, 5)
	variation(4, 5)
	variation(5, 5)
}

func TestDKGCanFinishWithAnyT(test *testing.T) {
	variation := func(t, n int) {
		combos := allReceiptCombosTOfN(t, n)
		for _,combo := range combos {
			group := newGroup(t, n)
			err := group.sendSubset(combo)
			if err != nil {
				test.Fatalf("DKG(t=%d,n=%d): Key share validation failed", t, n)
			}
			if !group.done(0) {
				test.Errorf("DKG(t=%d,n=%d): Party 0 not done after receiving shares from %v", t, n, combo)
			}
		}
	}

	variation(2, 3)
	variation(2, 4)
	variation(2, 5)
	variation(3, 5)
	variation(4, 5)
}

func TestDKGValidates(test *testing.T) {
	variation := func(t, n int) {
		dkg := NewSimpleDKG(t, n)
		err := dkg.ReceiveShare(PartyId(n - 1), invalidKeyShare)
		if err == nil {
			test.Errorf("DKG(t=%d,n=%d): Did not recognize invalid key share", t, n)
		}
	}

	variation(1, 1)
	variation(1, 3)
	variation(2, 3)
}

func TestDKGValidatesPastDone(test *testing.T) {
	// Initialize
	group := newGroup(3, 5)

	err := group.send(1, 0)
	if err != nil {
		test.Fatalf("DKG(t=3,n=5): Key share validation failed")
	}

	err = group.send(2, 0)
	if err != nil {
		test.Fatalf("DKG(t=3,n=5): Key share validation failed")
	}

	if !group.done(0) {
		test.Fatalf("DKG(t=3,n=5): Not done after receiving 2 remote shares")
	}

	// Test
	err = group.dkgs[0].ReceiveShare(4, invalidKeyShare)
	if err == nil {
		test.Errorf("DKG(t=3,n=5): Stopped validating after 2 remote shares")
	}
}

const T = 70
const N = 100

func BenchmarkMakeDKG(b *testing.B) {
	for n := 0; n < b.N; n++ {
		NewSimpleDKG(T, N)
	}
}

func BenchmarkReceiveShares(b *testing.B) {
	group := newGroup(T, N)
	locals := make([]SimpleDKG, b.N)
	for i := range locals {
		locals[i] = group.dkgs[0].clone()
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		for i := range group.dkgs[1:T] {
			share := group.dkgs[i].GetShareFor(1)
			locals[n].ReceiveShare(PartyId(i), share)
		}
	}
}
