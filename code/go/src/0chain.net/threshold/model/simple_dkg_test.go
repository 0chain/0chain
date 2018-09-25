package model

import (
	"os"
	"testing"

	"github.com/pmer/gobls"
)

func TestMain(m *testing.M) {
	gobls.Init(gobls.CurveFp254BNb)
	InitSimpleDKG()
	os.Exit(m.Run())
}

// PartyIds that a DKG party will refer to another party by, such that each DKG
// thinks of itself as being PartyId 0, which is not necessary, but it's how
// academia sometimes models DKGs.
func remote(n, from, to int) PartyId {
	return PartyId((n + to - from) % n)
}

type DKGs []SimpleDKG

func newDKGs(t, n int) DKGs {
	dkgs := make([]SimpleDKG, n)
	for i := range dkgs {
		dkgs[i] = NewSimpleDKG(t, n)
	}
	return dkgs
}

func (ds DKGs) send(from, to int) error {
	share := ds[from].GetShareFor(ds.remote(from, to))
	err := ds[to].ReceiveShare(ds.remote(to, from), share)
	return err
}

func (ds DKGs) sendMany(count int) error {
	for i := 1; i <= count; i++ {
		err := ds.send(i, 0)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ds DKGs) sendSubset(from []int) error {
	for _,i := range from {
		err := ds.send(i, 0)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ds DKGs) complete(i int) error {
	for other := range ds {
		err := ds.send(other, i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ds DKGs) completeAll() error {
	for i := range ds {
		err := ds.complete(i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ds DKGs) done(i int) bool {
	return ds[i].IsDone()
}

func (ds DKGs) remote(from, to int) PartyId {
	n := ds[0].N
	return remote(n, from, to)
}

func TestDKGCanSelfShare(test *testing.T) {
	variation := func(t, n int) {
		dkg := NewSimpleDKG(t, n)
		share := dkg.GetShareFor(0)
		err := dkg.ReceiveShare(0, share)
		if err != nil {
			test.Errorf("DKG(t=%d,n=%d): Receive own share failed: %v", t, n, err)
		}
	}

	variation(1, 1)
	variation(1, 2)
	variation(2, 2)
	variation(2, 3)
}

func TestDKGCanShareAndReceive(test *testing.T) {
	variation := func(t, n int) {
		dkgs := newDKGs(t, n)
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if i == j {
					continue
				}
				err := dkgs.send(i, j)
				if err != nil {
					test.Errorf("DKG(t=%d,n=%d): Key share validation failed: %v", t, n, err)
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

func TestDKGCanFinish(test *testing.T) {
	variation := func(t, n int) {
		dkgs := newDKGs(t, n)
		err := dkgs.sendMany(n - 1)
		if err != nil {
			test.Fatalf("DKG(t=%d,n=%d): Key share validation failed: %v", t, n, err)
		}
		if !dkgs.done(0) {
			test.Errorf("DKG(t=%d,n=%d): Not done after receiving %d remote shares", t, n, n - 1)
		}
	}

	variation(2, 5)
	variation(3, 5)
	variation(4, 5)
	variation(5, 5)
}

func TestDKGValidates(test *testing.T) {
	variation := func(t, n int) {
		dkg := NewSimpleDKG(t, n)
		err := dkg.ReceiveShare(PartyId(n - 1), InvalidKeyShare)
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
	dkgs := newDKGs(3, 5)

	err := dkgs.sendMany(4)
	if err != nil {
		test.Fatalf("DKG(t=3,n=5): Key share validation failed: %v", err)
	}

	if !dkgs.done(0) {
		test.Fatalf("DKG(t=3,n=5): Not done after receiving 4 remote shares")
	}

	// Test
	err = dkgs[0].ReceiveShare(1, InvalidKeyShare)
	if err == nil {
		test.Errorf("DKG(t=3,n=5): Stopped validating after done")
	}
}

const T = 70
const N = 100

func BenchmarkDKGMake(b *testing.B) {
	for n := 0; n < b.N; n++ {
		NewSimpleDKG(T, N)
	}
}

func BenchmarkDKGReceiveShares(b *testing.B) {
	dkgs := newDKGs(T, N)
	locals := make([]SimpleDKG, b.N)
	for i := range locals {
		locals[i] = dkgs[0].clone()
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		for i := range dkgs[1:N] {
			share := dkgs[i].GetShareFor(1)
			locals[n].ReceiveShare(PartyId(i), share)
		}
	}
}
