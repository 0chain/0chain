package bls

import "testing"

type DKGs []BLSSimpleDKG

func newDKGs(t, n int) DKGs {
	dkgs := make([]BLSSimpleDKG, n)
	for i := range dkgs {
		dkgs[i] = MakeSimpleDKG(t, n)
	}
	return dkgs
}

//TBR
func TestMakeSimpleDKG(test *testing.T) {

	variation := func(t, n int) {
		dkg := MakeSimpleDKG(t, n)
		if dkg.T == 0 {
			test.Errorf("The t not set properly\n")
		}
	}

	variation(1, 2)

}
