package bls

import (
	"strconv"
	"testing"
)

type DKGs []BLSSimpleDKG

func newDKGs(t, n int) DKGs {
	dkgs := make([]BLSSimpleDKG, n)
	for i := range dkgs {
		dkgs[i] = MakeSimpleDKG(t, n)
	}
	return dkgs
}

/* Tests to check whether SecKey - secret key, mSec - master secret key and mVec - verification key are set
Here mSec[0] is the secretKey */
func TestMakeSimpleDKG(test *testing.T) {

	variation := func(t, n int) {
		dkg := MakeSimpleDKG(t, n)
		if dkg.mSec == nil {
			test.Errorf("The master secret key not set")
		}
		if dkg.mVec == nil {
			test.Errorf("The verification key not set")
		}
	}

	variation(2, 2)
	variation(2, 3)
	variation(2, 4)

}

/* Tests to check whether the shares are computed correctly through polynomial substitution method.
The List of IDs are given as {1,2,...n} */
func TestComputeKeyShares(test *testing.T) {

	variation := func(t, n int) {
		dkg := MakeSimpleDKG(t, n)
		forIDs := make([]PartyId, n)
		id := 1
		for i := 0; i < n; i++ {
			err := forIDs[i].SetDecString(strconv.Itoa(id))
			if err != nil {
				test.Errorf("Computing ID error %v\n", forIDs)
			}
			id++
		}
		secShares, err := dkg.ComputeKeyShare(forIDs)

		if err != nil {
			test.Errorf("The shares are not derived correctly")
		}

		if secShares == nil {
			test.Errorf("Array of shares are not derived correctly %v\n", secShares)
		}
	}
	variation(2, 2)
	variation(2, 3)
	variation(2, 4)
}

/* Tests to check the shares are derived through polynomial substitution */
func TestComputeKeyShareRecoversSecretKey(test *testing.T) {
	variation := func(t, n int) {
		dkg := MakeSimpleDKG(t, n)
		forIDs := make([]PartyId, n)
		id := 1
		for i := 0; i < n; i++ {
			err := forIDs[i].SetDecString(strconv.Itoa(id))
			if err != nil {
				test.Errorf("Computing ID error")
			}
			id++
		}
		secShares, err := dkg.ComputeKeyShare(forIDs)
		if err != nil {
			test.Errorf("The shares are not derived correctly")
		}
		var sec2 Key
		err = sec2.Recover(secShares, forIDs)
		if err != nil {
			test.Errorf("Recover shares Lagrange Interpolation error %v\n ,  %v\n,  %v\n", secShares, forIDs, sec2)
		}

		if !dkg.mSec[0].IsEqual(&sec2) {
			test.Errorf("Mismatch in recovered secret key:\n  %s\n  %s.", dkg.mSec[0].GetHexString(), sec2.GetHexString())
		}

	}

	variation(2, 2)
	variation(2, 3)
	variation(2, 4)
}
